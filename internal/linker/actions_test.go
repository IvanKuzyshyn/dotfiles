package linker_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/linker"
)

func TestApply_Symlink(t *testing.T) {
	fs := xfs.NewFake()
	if err := fs.MkdirAll("/dotfiles/configs/git", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/dotfiles/configs/git/.gitconfig", 0o644); err != nil {
		t.Fatal(err)
	}
	d := linker.Decision{
		Source: "/dotfiles/configs/git/.gitconfig",
		Target: "/home/me/.gitconfig",
		Kind:   linker.DecideSymlink,
	}
	if err := linker.Apply(d, nil, linker.Skip, "/backup", fs); err != nil {
		t.Fatal(err)
	}
	link, err := fs.Readlink("/home/me/.gitconfig")
	if err != nil {
		t.Fatal(err)
	}
	if link != d.Source {
		t.Errorf("readlink = %q, want %q", link, d.Source)
	}
}

func TestApply_AlreadyOk_NoOp(t *testing.T) {
	fs := xfs.NewFake()
	d := linker.Decision{Kind: linker.DecideAlreadyOk}
	if err := linker.Apply(d, nil, linker.Skip, "/backup", fs); err != nil {
		t.Errorf("AlreadyOk should be no-op, got %v", err)
	}
}

func TestApply_Backup(t *testing.T) {
	fs := xfs.NewFake()
	// Existing file we'll back up
	if err := fs.MkdirAll("/home/me", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/home/me/.gitconfig", 0o644); err != nil {
		t.Fatal(err)
	}
	// Source
	if err := fs.MkdirAll("/dotfiles/configs/git", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/dotfiles/configs/git/.gitconfig", 0o644); err != nil {
		t.Fatal(err)
	}
	d := linker.Decision{
		Source: "/dotfiles/configs/git/.gitconfig",
		Target: "/home/me/.gitconfig",
		Kind:   linker.DecideConflict,
	}
	c := &linker.Conflict{Target: "/home/me/.gitconfig", ExistingKind: linker.ExistingFile}
	backupRoot := "/backups/run-1"
	if err := linker.Apply(d, c, linker.Backup, backupRoot, fs); err != nil {
		t.Fatal(err)
	}
	// New symlink is in place
	link, err := fs.Readlink("/home/me/.gitconfig")
	if err != nil {
		t.Fatal(err)
	}
	if link != d.Source {
		t.Errorf("symlink wrong: %q", link)
	}
	// Original file backed up under backupRoot/home/me/.gitconfig
	if _, err := fs.Stat(filepath.Join(backupRoot, "home/me/.gitconfig")); err != nil {
		t.Errorf("backup not found: %v", err)
	}
}

func TestApply_Overwrite(t *testing.T) {
	fs := xfs.NewFake()
	if err := fs.MkdirAll("/home/me", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/home/me/.x", 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/dotfiles/configs/x", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/dotfiles/configs/x/.x", 0o644); err != nil {
		t.Fatal(err)
	}
	d := linker.Decision{
		Source: "/dotfiles/configs/x/.x",
		Target: "/home/me/.x",
		Kind:   linker.DecideConflict,
	}
	c := &linker.Conflict{Target: "/home/me/.x", ExistingKind: linker.ExistingFile}
	if err := linker.Apply(d, c, linker.Overwrite, "/backup", fs); err != nil {
		t.Fatal(err)
	}
	link, err := fs.Readlink("/home/me/.x")
	if err != nil {
		t.Fatal(err)
	}
	if link != d.Source {
		t.Errorf("symlink wrong: %q", link)
	}
}

func TestApply_Skip(t *testing.T) {
	fs := xfs.NewFake()
	if err := fs.MkdirAll("/home/me", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/home/me/.x", 0o644); err != nil {
		t.Fatal(err)
	}
	d := linker.Decision{Source: "/src", Target: "/home/me/.x", Kind: linker.DecideConflict}
	if err := linker.Apply(d, &linker.Conflict{}, linker.Skip, "/backup", fs); err != nil {
		t.Fatal(err)
	}
	// Should NOT have created a symlink (target file still there).
	info, err := fs.Lstat("/home/me/.x")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("skip should not have replaced existing with symlink")
	}
}

func TestApply_Abort(t *testing.T) {
	fs := xfs.NewFake()
	d := linker.Decision{Kind: linker.DecideConflict}
	err := linker.Apply(d, &linker.Conflict{}, linker.Abort, "/backup", fs)
	if !errors.Is(err, linker.ErrAborted) {
		t.Errorf("want ErrAborted, got %v", err)
	}
}

func TestApply_OverwriteDirConflict(t *testing.T) {
	fs := xfs.NewFake()
	if err := fs.MkdirAll("/home/me/.foo/inner", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/dotfiles/configs/foo", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/dotfiles/configs/foo/inner", 0o644); err != nil {
		t.Fatal(err)
	}
	d := linker.Decision{
		Source: "/dotfiles/configs/foo/inner",
		Target: "/home/me/.foo",
		Kind:   linker.DecideConflict,
	}
	c := &linker.Conflict{ExistingKind: linker.ExistingDir}
	if err := linker.Apply(d, c, linker.Overwrite, "/backup", fs); err != nil {
		t.Fatal(err)
	}
}

// Sanity check that the Action constants in linker match event constants.
func TestActionConstants(t *testing.T) {
	if linker.Backup == linker.Overwrite || linker.Skip == linker.Abort {
		t.Error("action constants should be distinct")
	}
}
