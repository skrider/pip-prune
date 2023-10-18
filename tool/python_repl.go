package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	flag.Parse()
	python := filepath.Join(flag.Args()[0], "bin", "python")
	cmd := exec.Command(python)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Run()
}
