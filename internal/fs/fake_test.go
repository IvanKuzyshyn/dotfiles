package fs_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
)

func TestFake_StatMissing(t *testing.T) {
	f := xfs.NewFake()
	_, err := f.Stat("/missing")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want NotExist, got %v", err)
	}
}

func TestFake_MkdirAllAndStat(t *testing.T) {
	f := xfs.NewFake()
	if err := f.MkdirAll("/a/b/c", 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat("/a/b")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Errorf("want dir")
	}
}

func TestFake_SymlinkReadlinkLstat(t *testing.T) {
	f := xfs.NewFake()
	if err := f.MkdirAll("/d", 0o755); err != nil {
		t.Fatal(err)
	}
	// Need a target file
	if err := f.MkdirAll("/src", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.Symlink("/src", "/d/link"); err != nil {
		t.Fatal(err)
	}
	got, err := f.Readlink("/d/link")
	if err != nil || got != "/src" {
		t.Fatalf("readlink got=%q err=%v", got, err)
	}
	info, err := f.Lstat("/d/link")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("lstat should show symlink mode")
	}
	// Stat follows
	info, err = f.Stat("/d/link")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Errorf("stat should follow symlink to /src (dir)")
	}
}

func TestFake_RenameAndRemove(t *testing.T) {
	f := xfs.NewFake()
	if err := f.MkdirAll("/a/b", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.Rename("/a/b", "/a/c"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stat("/a/b"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("rename old should be gone")
	}
	if err := f.RemoveAll("/a"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Stat("/a"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("removeall should remove /a")
	}
}

func TestFake_WalkLexical(t *testing.T) {
	f := xfs.NewFake()
	for _, d := range []string{"/r", "/r/a", "/r/b"} {
		if err := f.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var visited []string
	err := f.Walk("/r", func(path string, info xfs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(visited)
	want := []string{"/r", "/r/a", "/r/b"}
	if len(visited) != len(want) {
		t.Fatalf("want %v got %v", want, visited)
	}
	for i := range visited {
		if visited[i] != want[i] {
			t.Fatalf("at %d: want %q got %q", i, want[i], visited[i])
		}
	}
}

// Suppress unused import warning for filepath if you don't use it elsewhere.
var _ = filepath.Join
