package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
)

// ErrDotfilesDirNotFound indicates no candidate path resolved to a usable
// dotfiles directory.
var ErrDotfilesDirNotFound = errors.New("dotfiles directory not found")

// Resolver finds the dotfiles directory by checking flag / env var / cwd
// / default home locations in order. All filesystem access goes through
// FS for testability.
type Resolver struct {
	FS   xfs.FS
	Home string
	Cwd  string
	Env  func(string) string // os.Getenv by default; injected for tests
}

// Resolve returns the absolute path to the dotfiles directory, or an
// error wrapping ErrDotfilesDirNotFound. The order is:
//  1. flag (if non-empty)
//  2. $DOTFILES_DIR (if set and non-empty)
//  3. cwd if it contains configs/
//  4. ~/dotfiles if it contains configs/
//  5. ~/.dotfiles if it contains configs/
func (r Resolver) Resolve(flag string) (string, error) {
	if flag != "" {
		return r.abs(flag)
	}
	if env := r.env("DOTFILES_DIR"); env != "" {
		return r.abs(env)
	}
	if r.hasConfigs(r.Cwd) {
		return r.abs(r.Cwd)
	}
	if r.Home != "" {
		for _, sub := range []string{"dotfiles", ".dotfiles"} {
			candidate := filepath.Join(r.Home, sub)
			if r.hasConfigs(candidate) {
				return r.abs(candidate)
			}
		}
	}
	return "", fmt.Errorf("%w: tried --dotfiles-dir, DOTFILES_DIR, cwd, ~/dotfiles, ~/.dotfiles",
		ErrDotfilesDirNotFound)
}

func (r Resolver) env(key string) string {
	if r.Env != nil {
		return r.Env(key)
	}
	return os.Getenv(key)
}

func (r Resolver) hasConfigs(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := r.FS.Stat(filepath.Join(dir, "configs"))
	return err == nil && info.IsDir()
}

func (r Resolver) abs(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	if r.Cwd == "" {
		return filepath.Abs(p)
	}
	return filepath.Join(r.Cwd, p), nil
}
