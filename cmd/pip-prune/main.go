package main

import (
	"log"
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
    pip-prune -command COMMAND --base IMAGE --requirements requirements.txt
`

func printUsage() {
	log.Fatal(USAGE)
}

func main() {
    printUsage()
}
