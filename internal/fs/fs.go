package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileInfo aliases os.FileInfo for clarity in callers.
type FileInfo = fs.FileInfo

// WalkFn matches filepath.WalkFunc signature.
type WalkFn func(path string, info FileInfo, err error) error

// FS is the filesystem seam used by linker, steps, and CLI helpers.
// Real implementation delegates to os/filepath; tests substitute Fake.
type FS interface {
	Stat(path string) (FileInfo, error)
	Lstat(path string) (FileInfo, error)
	Symlink(source, target string) error
	Readlink(path string) (string, error)
	Mkdir(path string, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Rename(old, new string) error
	Remove(path string) error
	RemoveAll(path string) error
	Walk(root string, fn WalkFn) error
}

// Real delegates all calls to the host OS.
type Real struct{}

func (Real) Stat(path string) (FileInfo, error)           { return os.Stat(path) }
func (Real) Lstat(path string) (FileInfo, error)          { return os.Lstat(path) }
func (Real) Symlink(source, target string) error          { return os.Symlink(source, target) }
func (Real) Readlink(path string) (string, error)         { return os.Readlink(path) }
func (Real) Mkdir(path string, perm os.FileMode) error    { return os.Mkdir(path, perm) }
func (Real) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (Real) Rename(old, new string) error                 { return os.Rename(old, new) }
func (Real) Remove(path string) error                     { return os.Remove(path) }
func (Real) RemoveAll(path string) error                  { return os.RemoveAll(path) }
func (Real) Walk(root string, fn WalkFn) error {
	return filepath.Walk(root, func(path string, info FileInfo, err error) error {
		return fn(path, info, err)
	})
}
