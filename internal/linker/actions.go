package linker

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
)

// ErrAborted indicates the user chose to abort linking via the resolver.
var ErrAborted = errors.New("linker: aborted by resolver")

// Apply executes one Decision against the filesystem. For Conflict
// decisions, the resolver is consulted to determine which Action to take.
// backupRoot is the directory under which backups are placed (typically
// per-run, e.g. ~/.dotfiles_backup_<UTC-ts>).
func Apply(d Decision, c *Conflict, action Action, backupRoot string, fs xfs.FS) error {
	switch d.Kind {
	case DecideSymlink:
		return createSymlink(d.Source, d.Target, fs)
	case DecideAlreadyOk:
		return nil
	case DecideConflict:
		switch action {
		case Backup:
			if err := backupTarget(d.Target, backupRoot, fs); err != nil {
				return err
			}
			return createSymlink(d.Source, d.Target, fs)
		case Overwrite:
			if err := removeTarget(d.Target, fs, c); err != nil {
				return err
			}
			return createSymlink(d.Source, d.Target, fs)
		case Skip:
			return nil
		case Abort:
			return ErrAborted
		default:
			return fmt.Errorf("linker.Apply: unknown action %d", action)
		}
	default:
		return fmt.Errorf("linker.Apply: unknown decision kind %d", d.Kind)
	}
}

func createSymlink(source, target string, fs xfs.FS) error {
	parent := filepath.Dir(target)
	if parent != "" && parent != "/" && parent != "." {
		if err := fs.MkdirAll(parent, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", parent, err)
		}
	}
	return fs.Symlink(source, target)
}

func backupTarget(target, backupRoot string, fs xfs.FS) error {
	dest := backupPath(backupRoot, target)
	parent := filepath.Dir(dest)
	if parent != "" && parent != "/" && parent != "." {
		if err := fs.MkdirAll(parent, 0o755); err != nil {
			return fmt.Errorf("create backup parent %s: %w", parent, err)
		}
	}
	if err := fs.Rename(target, dest); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", target, dest, err)
	}
	return nil
}

// backupPath nests target under backupRoot, stripping the leading "/" so
// /home/me/.gitconfig → backupRoot/home/me/.gitconfig.
func backupPath(backupRoot, target string) string {
	rel := strings.TrimPrefix(target, "/")
	return filepath.Join(backupRoot, rel)
}

// removeTarget deletes the existing item at target, dispatching on kind.
func removeTarget(target string, fs xfs.FS, c *Conflict) error {
	if c != nil && c.ExistingKind == ExistingDir {
		return fs.RemoveAll(target)
	}
	return fs.Remove(target)
}
