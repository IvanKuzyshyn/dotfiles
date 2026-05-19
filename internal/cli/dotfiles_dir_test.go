package cli_test

import (
	"errors"
	"os"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/cli"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
)

func mkDir(t *testing.T, fs xfs.FS, path string) {
	t.Helper()
	if err := fs.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func staticEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolve_FlagWins(t *testing.T) {
	fs := xfs.NewFake()
	mkDir(t, fs, "/dotfiles/configs")
	r := cli.Resolver{FS: fs, Home: "/home/me", Cwd: "/somewhere", Env: staticEnv(nil)}
	got, err := r.Resolve("/dotfiles")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/dotfiles" {
		t.Errorf("want /dotfiles, got %q", got)
	}
}

func TestResolve_EnvVar(t *testing.T) {
	fs := xfs.NewFake()
	mkDir(t, fs, "/abc/configs")
	r := cli.Resolver{FS: fs, Home: "/home", Env: staticEnv(map[string]string{"DOTFILES_DIR": "/abc"})}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/abc" {
		t.Errorf("want /abc, got %q", got)
	}
}

func TestResolve_CwdWithConfigs(t *testing.T) {
	fs := xfs.NewFake()
	mkDir(t, fs, "/here/configs")
	r := cli.Resolver{FS: fs, Home: "/home", Cwd: "/here", Env: staticEnv(nil)}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/here" {
		t.Errorf("want /here, got %q", got)
	}
}

func TestResolve_HomeDotfiles(t *testing.T) {
	fs := xfs.NewFake()
	mkDir(t, fs, "/home/me/dotfiles/configs")
	r := cli.Resolver{FS: fs, Home: "/home/me", Cwd: "/elsewhere", Env: staticEnv(nil)}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/me/dotfiles" {
		t.Errorf("want /home/me/dotfiles, got %q", got)
	}
}

func TestResolve_HomeHiddenDotfiles(t *testing.T) {
	fs := xfs.NewFake()
	mkDir(t, fs, "/home/me/.dotfiles/configs")
	r := cli.Resolver{FS: fs, Home: "/home/me", Cwd: "/elsewhere", Env: staticEnv(nil)}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/home/me/.dotfiles" {
		t.Errorf("want /home/me/.dotfiles, got %q", got)
	}
}

func TestResolve_NoneFound(t *testing.T) {
	fs := xfs.NewFake()
	r := cli.Resolver{FS: fs, Home: "/home", Cwd: "/elsewhere", Env: staticEnv(nil)}
	_, err := r.Resolve("")
	if err == nil || !errors.Is(err, cli.ErrDotfilesDirNotFound) {
		t.Fatalf("want ErrDotfilesDirNotFound, got %v", err)
	}
}

// Use os.Setenv-style helper for compatibility (not strictly required since
// we use Env injection, but proves the API is sensible).
var _ = os.Setenv
