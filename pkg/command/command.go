package command

import (
	"bufio"
	"fmt"
	"io"
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

func (c *Command) TraceFiles(v *venv.Venv) (bool, map[string]bool, error) {
	var err error
    files := make(map[string]bool, 0)

	c.mu.Lock()
	n := c.n
	c.n += 1
	c.mu.Unlock()

	stderrPath := filepath.Join(c.logDir, fmt.Sprintf("stderr-%d.log", n))
	stdoutPath := filepath.Join(c.logDir, fmt.Sprintf("stdout-%d.log", n))
	stracePath := filepath.Join(c.logDir, fmt.Sprintf("strace-%d.log", n))

    args := make([]string, 0)
    args = append(args, "--output", stracePath)
    args = append(args, "--trace", "%file")
    args = append(args, v.PythonInterpreterPath())
    args = append(args, c.args...)

	cmd := exec.Command("strace", args...)
	cmd.Env = os.Environ()

	cmd.Stderr, err = os.Create(stderrPath)
	if err != nil {
		log.Printf("error creating %s for %s: %s", stderrPath, c.String(), err)
		return false, files, err
	}

	cmd.Stdout, err = os.Create(stdoutPath)
	if err != nil {
		log.Printf("error creating %s for %s: %s", stdoutPath, c.String(), err)
		return false, files, err
	}

	err = cmd.Run()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode() == 0, files, nil
		} else {
			return false, files, err
		}
	}

    f, err := os.Open(stracePath)
    if err != nil {
        log.Printf("error opening strace file %s for %s: %s", stracePath, c.String(), err)
        return false, files, err
    }
    defer f.Close()

    s := bufio.NewScanner(f)
    for s.Scan() {
        line := s.Text()
        parts := strings.Split(line, "\"")
        if len(parts) > 1 {
            if relative := strings.TrimPrefix(parts[1], v.LibRoot()); relative != parts[1] && relative != "" {
                files[relative[1:]] = true
            }
        }
    }

	return true, files, nil
}

func (c *Command) String() string {
	return fmt.Sprintf("python %s", strings.Join(c.args, " "))
}

func (c *Command) dumpLogs(i int) {
    if (i >= c.n) {
        return
    }

	stderrPath := filepath.Join(c.logDir, fmt.Sprintf("stderr-%d.log", i))
    stderrFile, err := os.Open(stderrPath)
    defer stderrFile.Close()
    if err != nil {
        log.Printf("command %s failed to open stderr at %s\n", c.String(), stderrPath)
    }

	stdoutPath := filepath.Join(c.logDir, fmt.Sprintf("stdout-%d.log", i))
    stdoutFile, err := os.Open(stdoutPath)
    defer stdoutFile.Close()
    if err != nil {
        log.Printf("command %s failed to open stderr at %s\n", c.String(), stdoutPath)
    }

    io.Copy(os.Stdout, stderrFile)
    io.Copy(os.Stdout, stdoutFile)
}

func (c *Command) Dump() {
    c.dumpLogs(c.n - 1)
}
