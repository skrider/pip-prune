package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/skrider/python-image-pruner/pkg/command"
	"github.com/skrider/python-image-pruner/pkg/ignore"
	"github.com/skrider/python-image-pruner/pkg/venv"
)

const USAGE string = `softgrep 0.0.1
Stephen Krider <skrider@berkeley.edu>

pip-prune uses a greedy algorithm to optimize the dependency footprint of a
python-based container image. Given a base docker image, a requirements.txt
file, and a command to run to determine whether the service works, pip-prune
attempts to remove as much of the pip install output as possible while still
ensuring that the command exits successfully.

Modules and files that can be successfully pruned are then output as a list.
They can then be deleted as part of your docker build phase.

USAGE: 
    pip-prune <python-args>
    pip-prune -requirements requirements.txt main.py
`

func printUsage() {
	log.Fatal(USAGE)
}

var (
	requirementsArg string
	cleanupArg      bool
	depthArg        int
)

func init() {
	flag.StringVar(&requirementsArg, "requirements", "requirements.txt", "requirements file to use")
	flag.BoolVar(&cleanupArg, "cleanup", true, "cleanup temporary venvs")
	flag.IntVar(&depthArg, "depth", 1, "max depth to search")
	flag.Usage = printUsage
}

func main() {
	flag.Parse()
	ignore.InitIgnores()

	// read in the req file and calculate the hash
	file, err := os.Open(requirementsArg)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Fatal(err)
	}

	venvPath := filepath.Join(os.TempDir(), fmt.Sprintf("pip-prune-venv-ref-%s", hex.EncodeToString(h.Sum(nil))[:16]))
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		// create the venv
		fmt.Println("Creating venv at", venvPath)
		cmd := exec.Command("python3", "-m", "venv", venvPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}

		// install the requirements
		cmd = exec.Command(filepath.Join(venvPath, "bin", "pip"), "install", "-r", requirementsArg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Using existing venv at", venvPath)
	}

	vvenv := venv.MakeVenv(venvPath)
	if cleanupArg {
		defer vvenv.Destroy()
	}
	fmt.Println(flag.Args())

	refRoot := vvenv.ReferencePath()

	fringe := make([]string, 1)
	fringe[0] = ""

	depth := 0

	cmd := command.MakeCommand(flag.Args())

	for len(fringe) > 0 {
		path := fringe[0]
		depth = strings.Count(path, "/")
		if depth > depthArg {
			break
		}
		absPath := filepath.Join(refRoot, path)
		fringe = fringe[1:]
		// attempt to prune the path
		err = vvenv.Prune(path)
		if err != nil {
			log.Fatal(err)
		}
		ok, err := cmd.Run(vvenv)
		if err != nil {
			log.Fatal(err)
		}

		if ok {
			fmt.Println("Pruned path:", path)
		} else {
			// step further into the directory tree
			fmt.Println("Failed to prune:", path)
			err = vvenv.Unprune(path)
			if err != nil {
				log.Fatal(err)
			}

			entries, err := os.ReadDir(absPath)
			if err != nil {
				log.Fatal(err)
			}
			for _, e := range entries {
				relPath := filepath.Join(path, e.Name())
				if e.IsDir() && !ignore.Match(relPath) {
					fringe = append(fringe, relPath)
				}
			}
		}
	}

	ok, err := cmd.Run(vvenv)
	if !ok {
		log.Fatal("not ok at termination")
	}
}
