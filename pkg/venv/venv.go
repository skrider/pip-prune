package venv

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// grow the "fringe" outwards

type Venv struct {
    // TODO replace with an enum specifying level
	lower      string
	merged     string
	upper      string
	workdir    string
	pythonName string
}

func MakeVenv(refPath string) *Venv {
	v := &Venv{lower: refPath}
	var err error

	root, err := os.MkdirTemp("", "pip-prune-venv-")
	if err != nil {
		return nil
	}
	fmt.Printf("Creating proxy venv at %s\n", root)

	dirs := make([]string, 3)
	for i, dir := range []string{"upper", "workdir", "merged"} {
		dirs[i] = filepath.Join(root, dir)
	}
	for _, dir := range dirs {
		err = os.Mkdir(dir, 0777)
		if err != nil {
			log.Printf("Failed to create %s: %s\n", dir, err)
			return nil
		}
	}
	v.upper = dirs[0]
	v.workdir = dirs[1]
	v.merged = dirs[2]

	err = v.mount()
	if err != nil {
		log.Printf("Failed to mount overlay: %s\n", err)
		return nil
	}

	entries, err := os.ReadDir(filepath.Join(refPath, "lib"))
	if err != nil {
		log.Printf("Failed to read lib dir: %s\n", err)
		return nil
	}
	v.pythonName = entries[0].Name()

	return v
}

func (v *Venv) mount() error {
    args := make([]string, 0)
    args = append(args, "-o")
    args = append(args, fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", v.lower, v.upper, v.workdir))
    args = append(args, v.merged)
    
    cmd := exec.Command("fuse-overlayfs", args...)
    return cmd.Run()
}

func (v *Venv) umount() error {
    args := make([]string, 0)
    args = append(args, v.merged)

    cmd := exec.Command("umount", args...)
    return cmd.Run()
}

func (v *Venv) Destroy() {
	// unmount merged
	err := v.umount()
	if err != nil {
		log.Printf("Failed to unmount merged: %s\n", err)
	}
	// remove root
	err = os.RemoveAll(v.merged)
	if err != nil {
		log.Printf("Failed to remove root: %s\n", err)
	}
}

const PYCACHE_RE = "__pycache__$"

func (v *Venv) PurgePycache() {
    re, _ := regexp.Compile(PYCACHE_RE)
    files := v.Contents("")
    for _, f := range files {
        if re.MatchString(f) {
            path := v.resolveMerged(f)
            err := os.RemoveAll(path)
            if err != nil {
                panic(err)
            }
        }
    }
}

// attempt to remove the provided path from the venv.
// path is provided relative to root.
func (v *Venv) Prune(path string) error {
    if _, err := os.Stat(v.resolveMerged(path)); os.IsNotExist(err) {
        return nil
    }
	return os.RemoveAll(v.resolveMerged(path))
}

// prints all files rooted at path
func (v *Venv) Contents(path string) []string {
    files := make([]string, 0)

    walkFunc := func(path string, info os.FileInfo, err error) error {
        if info.IsDir() {
            return nil
        }
        relative := strings.TrimPrefix(path, v.LibRoot())
        if relative != path && relative != "" {
            files = append(files, relative[1:])
        }
        return nil
    }

    filepath.Walk(v.resolveMerged(path), walkFunc)

    return files
}

// unprune re-inserts path into the venv tree. Only pruned
// paths will be unpruned, so this is simple and safe to do.
func (v *Venv) Unprune(paths ...string) error {
	err := v.umount()
	if err != nil {
		return err
	}

    for _, p := range paths {
	    err = os.Remove(v.resolveUpper(p))
        if err != nil {
            log.Printf("Failed to remove %s: %s\n", v.resolveUpper(p), err)
        }
    }

	return v.mount()
}

func (v *Venv) SizeH(p string) string {
    // TODO write a generic traverse function for a venv and use it for this and for
    // contents
    cmd := exec.Command("du", "--human-readable", "--summarize", v.resolveMerged(p))
    out, err := cmd.Output()
    if err != nil {
        panic(err)
    }
    parts := strings.Split(string(out), "\t")
    return parts[0]
}

func (v *Venv) LibRoot() string {
    return filepath.Join(v.merged, "lib", v.pythonName, "site-packages")
}

func (v *Venv) resolveMerged(path string) string {
	return filepath.Join(v.merged, "lib", v.pythonName, "site-packages", path)
}

func (v *Venv) resolveRef(path string) string {
	return filepath.Join(v.lower, "lib", v.pythonName, "site-packages", path)
}

func (v *Venv) resolveUpper(path string) string {
	return filepath.Join(v.upper, "lib", v.pythonName, "site-packages", path)
}

func (v *Venv) PythonInterpreterPath() string {
	return filepath.Join(v.merged, "bin", "python")
}
