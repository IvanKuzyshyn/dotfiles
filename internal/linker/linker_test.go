package linker_test

import (
	"testing"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/linker"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

// newEnv builds an Env with a fake FS and standard test paths.
func newEnv(t *testing.T) (step.Env, *xfs.Fake) {
	t.Helper()
	fs := xfs.NewFake()
	return step.Env{
		FS:          fs,
		DotfilesDir: "/dotfiles",
		HomeDir:     "/home/me",
	}, fs
}

func mkSourceFile(t *testing.T, fs *xfs.Fake, p string) {
	t.Helper()
	if err := fs.MkdirAll(parentDir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile(p, 0o644); err != nil {
		t.Fatal(err)
	}
}

func parentDir(p string) string {
	// filepath.Dir equivalent using path separators already present
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "/"
}

func TestInspect_MissingTarget(t *testing.T) {
	env, fs := newEnv(t)
	mkSourceFile(t, fs, "/dotfiles/configs/git/.gitconfig")

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "git", Target: "~"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d (%+v)", len(plan.Decisions), plan.Decisions)
	}
	if plan.Decisions[0].Kind != linker.DecideSymlink {
		t.Errorf("want DecideSymlink, got %v", plan.Decisions[0].Kind)
	}
	if plan.Decisions[0].Target != "/home/me/.gitconfig" {
		t.Errorf("target = %q", plan.Decisions[0].Target)
	}
	if plan.Decisions[0].Source != "/dotfiles/configs/git/.gitconfig" {
		t.Errorf("source = %q", plan.Decisions[0].Source)
	}
}

func TestInspect_AlreadyOk(t *testing.T) {
	env, fs := newEnv(t)
	source := "/dotfiles/configs/git/.gitconfig"
	mkSourceFile(t, fs, source)

	// Existing symlink already points at the source file.
	if err := fs.MkdirAll("/home/me", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.Symlink(source, "/home/me/.gitconfig"); err != nil {
		t.Fatal(err)
	}

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "git", Target: "~"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(plan.Decisions))
	}
	if plan.Decisions[0].Kind != linker.DecideAlreadyOk {
		t.Errorf("want DecideAlreadyOk, got %v", plan.Decisions[0].Kind)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("want 0 conflicts, got %+v", plan.Conflicts)
	}
}

func TestInspect_Conflict_SymlinkOther(t *testing.T) {
	env, fs := newEnv(t)
	source := "/dotfiles/configs/git/.gitconfig"
	mkSourceFile(t, fs, source)

	// Existing target is a symlink pointing somewhere else.
	if err := fs.MkdirAll("/home/me", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.Symlink("/somewhere/else", "/home/me/.gitconfig"); err != nil {
		t.Fatal(err)
	}

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "git", Target: "~"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 {
		t.Fatalf("decisions: %+v", plan.Decisions)
	}
	if plan.Decisions[0].Kind != linker.DecideConflict {
		t.Errorf("want DecideConflict, got %v", plan.Decisions[0].Kind)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %+v", plan.Conflicts)
	}
	if plan.Conflicts[0].ExistingKind != linker.ExistingSymlinkOther {
		t.Errorf("kind = %v", plan.Conflicts[0].ExistingKind)
	}
	if plan.Conflicts[0].Target != "/home/me/.gitconfig" {
		t.Errorf("conflict target = %q", plan.Conflicts[0].Target)
	}
}

func TestInspect_Conflict_RegularFile(t *testing.T) {
	env, fs := newEnv(t)
	source := "/dotfiles/configs/git/.gitconfig"
	mkSourceFile(t, fs, source)

	// Existing target is a regular file.
	if err := fs.MkdirAll("/home/me", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fs.CreateFile("/home/me/.gitconfig", 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "git", Target: "~"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 {
		t.Fatalf("decisions: %+v", plan.Decisions)
	}
	if plan.Decisions[0].Kind != linker.DecideConflict {
		t.Errorf("want DecideConflict, got %v", plan.Decisions[0].Kind)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %+v", plan.Conflicts)
	}
	if plan.Conflicts[0].ExistingKind != linker.ExistingFile {
		t.Errorf("kind = %v", plan.Conflicts[0].ExistingKind)
	}
}

func TestInspect_Conflict_Dir(t *testing.T) {
	env, fs := newEnv(t)
	source := "/dotfiles/configs/git/.gitconfig"
	mkSourceFile(t, fs, source)

	// Existing target is a directory.
	if err := fs.MkdirAll("/home/me/.gitconfig", 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "git", Target: "~"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 {
		t.Fatalf("decisions: %+v", plan.Decisions)
	}
	if plan.Decisions[0].Kind != linker.DecideConflict {
		t.Errorf("want DecideConflict, got %v", plan.Decisions[0].Kind)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("want 1 conflict, got %+v", plan.Conflicts)
	}
	if plan.Conflicts[0].ExistingKind != linker.ExistingDir {
		t.Errorf("kind = %v", plan.Conflicts[0].ExistingKind)
	}
}

func TestInspect_MultipleFiles_LexicalOrder(t *testing.T) {
	env, fs := newEnv(t)
	// Create two source files; Walk should return them in lexical order.
	mkSourceFile(t, fs, "/dotfiles/configs/git/.gitconfig")
	mkSourceFile(t, fs, "/dotfiles/configs/git/.gitignore_global")

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "git", Target: "~"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 2 {
		t.Fatalf("want 2 decisions, got %d (%+v)", len(plan.Decisions), plan.Decisions)
	}
	// .gitconfig comes before .gitignore_global lexically.
	if plan.Decisions[0].Target != "/home/me/.gitconfig" {
		t.Errorf("first target = %q", plan.Decisions[0].Target)
	}
	if plan.Decisions[1].Target != "/home/me/.gitignore_global" {
		t.Errorf("second target = %q", plan.Decisions[1].Target)
	}
}

func TestInspect_ExpandHome_TildeSlash(t *testing.T) {
	env, fs := newEnv(t)
	mkSourceFile(t, fs, "/dotfiles/configs/k9s/config.yaml")

	plan, err := linker.Inspect([]manifest.Config{
		{Source: "k9s", Target: "~/.config/k9s"},
	}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(plan.Decisions))
	}
	want := "/home/me/.config/k9s/config.yaml"
	if plan.Decisions[0].Target != want {
		t.Errorf("target = %q, want %q", plan.Decisions[0].Target, want)
	}
}

func TestInspect_NilFS(t *testing.T) {
	_, err := linker.Inspect(nil, step.Env{DotfilesDir: "/d"})
	if err == nil {
		t.Fatal("expected error for nil FS")
	}
}

func TestInspect_EmptyDotfilesDir(t *testing.T) {
	_, err := linker.Inspect(nil, step.Env{FS: xfs.NewFake()})
	if err == nil {
		t.Fatal("expected error for empty DotfilesDir")
	}
}

func TestInspect_EmptyConfigs(t *testing.T) {
	env, _ := newEnv(t)
	plan, err := linker.Inspect([]manifest.Config{}, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Decisions) != 0 {
		t.Errorf("want 0 decisions, got %d", len(plan.Decisions))
	}
}
