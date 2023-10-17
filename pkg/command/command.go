package command

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/skrider/python-image-pruner/pkg/venv"
)

// keeps track of the execution of a particular python command
type Command struct {
	args   []string
	logDir string
	n      int
	mu     sync.Mutex
}

func MakeCommand(args []string) *Command {
	c := &Command{args: args, n: 0}

	var err error
	c.logDir, err = os.MkdirTemp("", "pip-prune-command-")
	if err != nil {
		log.Printf("error creating log directory for %s: %s", c.String(), err)
		return nil
	}
	log.Printf("logging output of %s to %s", c.String(), c.logDir)

	return c
}

func (c *Command) Run(v *venv.Venv) (bool, error) {
	var err error

	c.mu.Lock()
	n := c.n
	c.n += 1
	c.mu.Unlock()

	stderrPath := filepath.Join(c.logDir, fmt.Sprintf("stderr-%d.log", n))
	stdoutPath := filepath.Join(c.logDir, fmt.Sprintf("stdout-%d.log", n))

	cmd := exec.Command(v.PythonInterpreterPath(), c.args...)
	cmd.Env = os.Environ()

	cmd.Stderr, err = os.Create(stderrPath)
	if err != nil {
		log.Printf("error creating %s for %s: %s", stderrPath, c.String(), err)
		return false, err
	}

	cmd.Stdout, err = os.Create(stdoutPath)
	if err != nil {
		log.Printf("error creating %s for %s: %s", stdoutPath, c.String(), err)
		return false, err
	}

	err = cmd.Run()
    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            return exitError.ExitCode() == 0, nil
        } else {
            return false, err
        }
    }
    return true, nil
}

func (c *Command) String() string {
	return fmt.Sprintf("python %s", strings.Join(c.args, " "))
}
