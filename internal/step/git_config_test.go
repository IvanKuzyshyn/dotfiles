package step_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestGitConfig_Build_Validation(t *testing.T) {
	step.RegisterGitConfig()
	_, err := step.Build(manifest.Step{Type: "git_config", Name: "x"})
	if err == nil {
		t.Fatal("expected error for missing key/value")
	}
	_, err = step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "k", "value": "v", "scope": "weird"},
	})
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
}

func TestGitConfig_Check_Repo_Matches(t *testing.T) {
	step.RegisterGitConfig()
	s, _ := step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "core.hooksPath", "value": ".githooks"},
	})
	fake := xexec.NewFake()
	fake.Script("git",
		[]string{"-C", "/dotfiles", "config", "--get", "core.hooksPath"},
		xexec.Result{Lines: []string{".githooks"}, ExitCode: 0})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake, DotfilesDir: "/dotfiles"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected satisfied=true")
	}
}

func TestGitConfig_Check_Repo_Mismatch(t *testing.T) {
	step.RegisterGitConfig()
	s, _ := step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "core.hooksPath", "value": ".githooks"},
	})
	fake := xexec.NewFake()
	fake.Script("git",
		[]string{"-C", "/dotfiles", "config", "--get", "core.hooksPath"},
		xexec.Result{Lines: []string{"otherval"}})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake, DotfilesDir: "/dotfiles"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected satisfied=false on value mismatch")
	}
}

func TestGitConfig_Check_Unset(t *testing.T) {
	step.RegisterGitConfig()
	s, _ := step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "core.hooksPath", "value": ".githooks"},
	})
	fake := xexec.NewFake()
	fake.Script("git",
		[]string{"-C", "/dotfiles", "config", "--get", "core.hooksPath"},
		xexec.Result{ExitCode: 1}) // exit 1 = key not set
	ok, err := s.Check(context.Background(), step.Env{Exec: fake, DotfilesDir: "/dotfiles"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected satisfied=false when key is unset")
	}
}

func TestGitConfig_Run_Repo(t *testing.T) {
	step.RegisterGitConfig()
	s, _ := step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "core.hooksPath", "value": ".githooks"},
	})
	fake := xexec.NewFake()
	fake.Script("git",
		[]string{"-C", "/dotfiles", "config", "core.hooksPath", ".githooks"},
		xexec.Result{})
	if err := s.Run(context.Background(), step.Env{Exec: fake, DotfilesDir: "/dotfiles"}, &collectSink{}); err != nil {
		t.Fatal(err)
	}
}

func TestGitConfig_Run_Global(t *testing.T) {
	step.RegisterGitConfig()
	s, _ := step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "user.email", "value": "a@b.com", "scope": "global"},
	})
	fake := xexec.NewFake()
	fake.Script("git",
		[]string{"config", "--global", "user.email", "a@b.com"},
		xexec.Result{})
	if err := s.Run(context.Background(), step.Env{Exec: fake}, &collectSink{}); err != nil {
		t.Fatal(err)
	}
}

func TestGitConfig_Repo_RequiresDotfilesDir(t *testing.T) {
	step.RegisterGitConfig()
	s, _ := step.Build(manifest.Step{
		Type: "git_config", Name: "x",
		Fields: map[string]any{"key": "k", "value": "v"},
	})
	_, err := s.Check(context.Background(), step.Env{Exec: xexec.NewFake()})
	if err == nil {
		t.Fatal("expected error for empty DotfilesDir")
	}
}
