package venv

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// grow the "fringe" outwards

type Venv struct {
	lower      string
	merged     string
	upper      string
	workdir    string
	pythonName string
}

// NB may need runtime.LockOSThread()

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
    args = append(args, "-o")
    args = append(args, "allow_")
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

// attempt to remove the provided path from the venv.
// path is provided relative to root.
func (v *Venv) Prune(path string) error {
	return os.RemoveAll(v.resolveLower(path))
}

// unprune re-inserts path into the venv tree. Only pruned
// paths will be unpruned, so this is simple and safe to do.
func (v *Venv) Unprune(path string) error {
	err := v.umount()
	if err != nil {
		return err
	}

	err = os.Remove(v.resolveUpper(path))
	if err != nil {
		return err
	}

	return v.mount()
}

func (v *Venv) ReferencePath() string {
	return v.resolveLower("")
}

func (v *Venv) resolveLower(path string) string {
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
