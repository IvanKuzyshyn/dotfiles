package fs

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

// Fake is an in-memory filesystem implementation for testing.
type Fake struct {
	nodes map[string]*node
}

type node struct {
	isDir  bool
	target string // for symlinks
	mode   os.FileMode
	size   int64
}

// fakeInfo implements os.FileInfo for the fake filesystem.
type fakeInfo struct {
	name string
	mode os.FileMode
	size int64
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return f.size }
func (f fakeInfo) Mode() os.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeInfo) Sys() any           { return nil }

// NewFake creates an empty fake filesystem.
func NewFake() *Fake {
	return &Fake{
		nodes: make(map[string]*node),
	}
}

// Stat returns info about the node at path, following symlinks.
func (f *Fake) Stat(path string) (FileInfo, error) {
	path = pathClean(path)
	n, err := f.getNode(path)
	if err != nil {
		return nil, err
	}
	// Follow symlinks
	if n.target != "" {
		return f.Stat(n.target)
	}
	return fakeInfo{
		name: pathBase(path),
		mode: n.mode,
		size: n.size,
	}, nil
}

// Lstat returns info about the node at path without following symlinks.
func (f *Fake) Lstat(path string) (FileInfo, error) {
	path = pathClean(path)
	n, err := f.getNode(path)
	if err != nil {
		return nil, err
	}
	return fakeInfo{
		name: pathBase(path),
		mode: n.mode,
		size: n.size,
	}, nil
}

// Symlink creates a symlink at target pointing to source.
func (f *Fake) Symlink(source, target string) error {
	target = pathClean(target)
	// Check if target already exists
	if _, exists := f.nodes[target]; exists {
		return &os.LinkError{Op: "symlink", Old: source, New: target, Err: os.ErrExist}
	}
	// Check if parent exists
	parent := pathDir(target)
	if parent != "/" && parent != "" {
		if _, err := f.getNode(parent); err != nil {
			return &os.LinkError{Op: "symlink", Old: source, New: target, Err: os.ErrNotExist}
		}
	}
	f.nodes[target] = &node{
		target: source,
		mode:   os.ModeSymlink | 0o644,
	}
	return nil
}

// Readlink returns the target of the symlink at path.
func (f *Fake) Readlink(path string) (string, error) {
	path = pathClean(path)
	n, err := f.getNode(path)
	if err != nil {
		return "", err
	}
	if n.target == "" {
		return "", &os.PathError{Op: "readlink", Path: path, Err: errors.New("not a symlink")}
	}
	return n.target, nil
}

// Mkdir creates a directory at path. Parent must exist.
func (f *Fake) Mkdir(path string, perm os.FileMode) error {
	path = pathClean(path)
	if path == "/" {
		return &os.PathError{Op: "mkdir", Path: path, Err: os.ErrExist}
	}
	// Check if already exists
	if _, exists := f.nodes[path]; exists {
		return &os.PathError{Op: "mkdir", Path: path, Err: os.ErrExist}
	}
	// Check if parent exists
	parent := pathDir(path)
	if parent != "/" {
		if _, err := f.getNode(parent); err != nil {
			return &os.PathError{Op: "mkdir", Path: path, Err: os.ErrNotExist}
		}
	}
	f.nodes[path] = &node{
		isDir: true,
		mode:  os.ModeDir | perm,
	}
	return nil
}

// MkdirAll creates a directory and all parent directories.
func (f *Fake) MkdirAll(path string, perm os.FileMode) error {
	path = pathClean(path)
	if path == "/" {
		return nil
	}
	// Check if already exists
	if n, exists := f.nodes[path]; exists {
		if n.isDir {
			return nil
		}
		return &os.PathError{Op: "mkdir", Path: path, Err: os.ErrExist}
	}
	// Create parent first
	parent := pathDir(path)
	if parent != "/" && parent != "" {
		if err := f.MkdirAll(parent, perm); err != nil {
			return err
		}
	}
	f.nodes[path] = &node{
		isDir: true,
		mode:  os.ModeDir | perm,
	}
	return nil
}

// Rename moves a node from old to new.
func (f *Fake) Rename(old, new string) error {
	old = pathClean(old)
	new = pathClean(new)

	// Check if old exists
	n, err := f.getNode(old)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: old, New: new, Err: os.ErrNotExist}
	}
	// Check if new parent exists
	newParent := pathDir(new)
	if newParent != "/" {
		if _, err := f.getNode(newParent); err != nil {
			return &os.LinkError{Op: "rename", Old: old, New: new, Err: os.ErrNotExist}
		}
	}
	// Move the node
	f.nodes[new] = n
	delete(f.nodes, old)
	return nil
}

// Remove deletes the node at path. For directories, errors if non-empty.
func (f *Fake) Remove(path string) error {
	path = pathClean(path)
	if _, err := f.getNode(path); err != nil {
		return &os.PathError{Op: "remove", Path: path, Err: os.ErrNotExist}
	}
	// Check if directory is empty
	if f.isDirectory(path) {
		for p := range f.nodes {
			if pathDir(p) == path {
				return &os.PathError{Op: "remove", Path: path, Err: os.ErrPermission}
			}
		}
	}
	delete(f.nodes, path)
	return nil
}

// RemoveAll recursively removes path and all children. Idempotent.
func (f *Fake) RemoveAll(path string) error {
	path = pathClean(path)
	// If path doesn't exist, that's fine (idempotent)
	if _, err := f.getNode(path); err != nil {
		return nil
	}
	// Collect all nodes to delete (path and children)
	toDelete := []string{path}
	for p := range f.nodes {
		if isUnder(p, path) {
			toDelete = append(toDelete, p)
		}
	}
	// Delete all
	for _, p := range toDelete {
		delete(f.nodes, p)
	}
	return nil
}

// Walk visits root and all descendants in lexical order.
func (f *Fake) Walk(root string, fn WalkFn) error {
	root = pathClean(root)
	if _, err := f.getNode(root); err != nil {
		return err
	}

	// Collect all paths under root in lexical order
	var paths []string
	paths = append(paths, root)
	for p := range f.nodes {
		if p != root && isUnder(p, root) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)

	for _, p := range paths {
		n := f.nodes[p]
		info := fakeInfo{
			name: pathBase(p),
			mode: n.mode,
			size: n.size,
		}
		err := fn(p, info, nil)
		if err != nil {
			if err == ErrSkipDir {
				// Skip children of this directory
				// Continue to next path but don't descend
				continue
			}
			return err
		}
	}
	return nil
}

// getNode returns the node at path, or ErrNotExist if not found.
func (f *Fake) getNode(path string) (*node, error) {
	if path == "/" {
		// Root always exists (implicit)
		return &node{isDir: true, mode: os.ModeDir | 0o755}, nil
	}
	if n, exists := f.nodes[path]; exists {
		return n, nil
	}
	return nil, os.ErrNotExist
}

// isDirectory returns true if path is a directory.
func (f *Fake) isDirectory(path string) bool {
	n, err := f.getNode(path)
	return err == nil && n.isDir
}

// pathClean normalizes a path using forward slashes.
func pathClean(p string) string {
	return path.Clean(p)
}

// pathDir returns the directory part of a path.
func pathDir(p string) string {
	return path.Dir(p)
}

// pathBase returns the base name of a path.
func pathBase(p string) string {
	return path.Base(p)
}

// isUnder returns true if p is under root (strict subset, not equal).
func isUnder(p, root string) bool {
	if root == "/" {
		return p != "/"
	}
	if len(p) <= len(root) {
		return false
	}
	return p[:len(root)] == root && p[len(root)] == '/'
}

// ErrSkipDir is returned by WalkFn to skip descending into a directory.
var ErrSkipDir = filepath.SkipDir
