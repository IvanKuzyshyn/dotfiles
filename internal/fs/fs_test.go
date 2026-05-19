package fs_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
)

func TestReal_CreateAndStat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := (xfs.Real{}).Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 2 {
		t.Errorf("want size 2, got %d", info.Size())
	}
}

func TestReal_SymlinkAndReadlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "link")
	if err := (xfs.Real{}).Symlink(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := (xfs.Real{}).Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Errorf("readlink got %q want %q", got, src)
	}
	// Lstat on symlink should report Mode&Symlink
	info, err := (xfs.Real{}).Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("lstat should see symlink, got mode %v", info.Mode())
	}
}

func TestReal_MkdirAllRenameRemove(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	if err := (xfs.Real{}).MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(dir, "a", "b", "d")
	if err := (xfs.Real{}).Rename(nested, renamed); err != nil {
		t.Fatal(err)
	}
	if err := (xfs.Real{}).RemoveAll(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := (xfs.Real{}).Stat(filepath.Join(dir, "a")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected NotExist, got %v", err)
	}
}

func TestReal_Walk(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"x.txt", "sub/y.txt", "sub/z.txt"} {
		full := filepath.Join(dir, p)
		_ = os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte("."), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	err := (xfs.Real{}).Walk(dir, func(path string, info xfs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			got = append(got, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"sub/y.txt", "sub/z.txt", "x.txt"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("at %d: want %q got %q", i, want[i], got[i])
		}
	}
}
