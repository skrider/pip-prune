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
    pip-prune <options> -- <pip install args> -- <python args>
`

func printUsage() {
	log.Fatal(USAGE)
}

var (
	nocleanupArg bool
	absoluteArg  bool
	depthArg     int
	outputArg    string
)

func init() {
	flag.BoolVar(&nocleanupArg, "nocleanup", false, "do not cleanup temporary venvs")
	flag.BoolVar(&absoluteArg, "absolute", false, "output absolute path")
	flag.IntVar(&depthArg, "depth", 1, "max depth to search")
	flag.StringVar(&outputArg, "output", "prune.txt", "output file")
	flag.Usage = printUsage
}

func main() {
	flag.Parse()
	ignore.InitIgnores()

	args := flag.Args()
	split := -1
	for i, a := range args {
		if a == "--" {
			split = i
		}
	}
	if split == -1 || split == 0 || split == len(args)-1 {
		flag.Usage()
	}
	pipArgs := args[:split]
	pythonArgs := args[split+1:]

	// read in the req file and calculate the hash
	venvPath, err := initRefVenv(pipArgs)
	if err != nil {
		log.Fatal(err)
	}

	vvenv := venv.MakeVenv(venvPath)
	if vvenv == nil {
		log.Fatal("failed to create venv")
	}
	if !nocleanupArg {
		defer vvenv.Destroy()
	}
	log.Printf("Created venv. Initial size: %s", vvenv.SizeH(""))

	vvenv.PurgePycache()
	cmd := command.MakeCommand(pythonArgs)

	log.Printf("Tracing file open calls")
	ok, tracedFiles, err := cmd.TraceFiles(vvenv)
	if err != nil {
		cmd.Dump()
		log.Fatal(err)
	} else if !ok {
		log.Fatal("command did not work first try")
	}

	log.Println("removing unused files")

	prunedFiles := make([]string, 0)
	for _, f := range vvenv.Contents("") {
		if _, ok := tracedFiles[f]; !ok && !ignore.Match(f) {
			err := vvenv.Prune(f)
			prunedFiles = append(prunedFiles, f)
			if err != nil {
				log.Println(err)
			}
		}
	}

	vvenv.PurgePycache()
	ok, err = cmd.Run(vvenv)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		cmd.Dump()
		log.Fatal("command did not work after pruning. Try ignoring files.")
	}

	log.Printf("Pruning successful. Final size: %s", vvenv.SizeH(""))

	// write out the pruned files
	f, err := os.Create(outputArg)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	for _, p := range prunedFiles {
		fmt.Fprintf(f, "%s\n", p)
	}

	log.Printf("Pruned files written to %s", outputArg)
}

func initRefVenv(pipArgs []string) (string, error) {
	h := sha256.New()
	for _, a := range pipArgs {
		if a[0] != '-' {
			if _, err := os.Stat(a); err == nil {
				file, err := os.Open(a)
				if err != nil {
					log.Fatal(err)
				}
				io.Copy(h, file)
				file.Close()
			} else {
				fmt.Fprintf(h, "%s", a)
			}
		}
	}

	hDigest := hex.EncodeToString(h.Sum(nil))[:16]

	venvPath := filepath.Join(os.TempDir(), fmt.Sprintf("pip-prune-ref-%s", hDigest))
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		// create the venv
		fmt.Println("Creating venv at", venvPath)
		cmd := exec.Command("python3", "-m", "venv", venvPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}

		// install the requirements
		args := make([]string, 0)
		args = append(args, "install")
		args = append(args, pipArgs...)
		cmd = exec.Command(filepath.Join(venvPath, "bin", "pip"), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
	} else {
		fmt.Println("Using existing venv at", venvPath)
	}

	return venvPath, nil
}
