package venv

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// grow the "fringe" outwards

type Venv struct {
	refLibPath        string
	libPath           string
	rootPath          string
	pythonCommandArgs []string
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return nil
}

// one thread traverses the reference venv and enqueues potential states to try
func MakeVenv(referencePath string, pythonCommandArgs []string) *Venv {
	root, err := os.MkdirTemp("", "pip-prune-venv-")
	fmt.Printf("Creating proxy venv at %s\n", root)
	if err != nil {
		return nil
	}

	// setup redundant symlinks
	entries, err := os.ReadDir(referencePath)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		refPath := filepath.Join(referencePath, e.Name())
		rootPath := filepath.Join(root, e.Name())
		if e.Name() != "lib" && e.Name() != "pyvenv.cfg" && e.Name() != "lib64" {
			err = os.Symlink(refPath, rootPath)
		} else if e.Name() == "pyvenv.cfg" {
			// copy pyenv.cfg to root
			err = copyFile(refPath, rootPath)
		}
	}

	// setup lib
	err = os.Mkdir(filepath.Join(root, "lib"), 0777)
	if err != nil {
		return nil
	}
	err = os.Symlink(filepath.Join(root, "lib"), filepath.Join(root, "lib64"))
	if err != nil {
		return nil
	}
	entries, err = os.ReadDir(filepath.Join(referencePath, "lib"))
	python := entries[0].Name()
	err = os.Mkdir(filepath.Join(root, "lib", python), 0777)
	if err != nil {
		return nil
	}

	v := &Venv{
		refLibPath:        filepath.Join(referencePath, "lib", python, "site-packages"),
		libPath:           filepath.Join(root, "lib", python, "site-packages"),
		pythonCommandArgs: pythonCommandArgs,
		rootPath:          root,
	}

	// importantly, do not create site-packages

	return v
}

func DestroyVenv(v *Venv) error {
	return os.RemoveAll(v.libPath)
}

var NotSymlinkError error

// if the path is a symlink, then replaces with a dir and symlinks
// all contents
func (v *Venv) expand(path string) error {
	rootPath := filepath.Join(v.libPath, path)
	refPath := filepath.Join(v.refLibPath, path)

	stats, err := os.Lstat(rootPath)
	if err != nil {
		return err
	}
	if stats.Mode()&os.ModeSymlink == 0 {
		return NotSymlinkError
	}
	err = os.Remove(rootPath)
	if err != nil {
		return err
	}
	err = os.Mkdir(rootPath, 0777)
	if err != nil {
		return err
	}

	// create all entries as symlinks to reference dir
	entries, err := os.ReadDir(refPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		from := filepath.Join(refPath, e.Name())
		to := filepath.Join(rootPath, e.Name())
		err := os.Symlink(from, to)
		if err != nil {
			return err
		}
	}

	return nil
}

// attempt to remove the provided path from the venv.
// path is provided relative to root.
func (v *Venv) Prune(path string) error {
	rootPath := filepath.Join(v.libPath, path)

	// find the first symlink
	parentAcc := ""
	parentDirs := strings.Split(path, "/")

	// call expand all the way out to the parent
	for _, dir := range parentDirs {
		err := v.expand(parentAcc)
		// ignore not symlink error
		if err != NotSymlinkError {
			return err
		}
		// increment at the end to account for the base case
		// where we prune the root directory itself
		parentAcc = filepath.Join(parentAcc, dir)
	}

	// now rootPath is guarantueed to be a symlink, and we can remove
	// it straight up
	err := os.Remove(rootPath)
	if err != nil {
		return err
	}

	return nil
}

// unprune re-inserts path into the venv tree. Only pruned
// paths will be unpruned, so this is simple and safe to do.
func (v *Venv) Unprune(path string) error {
	return os.Symlink(filepath.Join(v.refLibPath, path), filepath.Join(v.libPath, path))
}

func (v *Venv) ReferencePath() string {
	return v.refLibPath
}

// attempts to see if the given venv will work
func (v *Venv) Test() (bool, error) {
	args := make([]string, 1)
	args[0] = "-u"
	args = append(args, v.pythonCommandArgs...)
    python := filepath.Join(v.rootPath, "bin", "python")
	cmd := exec.Command(python, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            return exitError.ExitCode() == 0, nil
        } else {
            return false, err
        }
	}
    return true, nil
}
