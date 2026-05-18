package step_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestGitClone_MissingFields(t *testing.T) {
	step.RegisterGitClone()
	_, err := step.Build(manifest.Step{Type: "git_clone", Name: "x"})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	_, err = step.Build(manifest.Step{
		Type: "git_clone", Name: "x",
		Fields: map[string]any{"url": "https://example.com/repo.git"},
	})
	if err == nil {
		t.Fatal("expected error for missing dest")
	}
}

func TestGitClone_Check_MissingDest(t *testing.T) {
	step.RegisterGitClone()
	s, _ := step.Build(manifest.Step{
		Type: "git_clone", Name: "g",
		Fields: map[string]any{"url": "u", "dest": "/d"},
	})
	fs := xfs.NewFake()
	ok, err := s.Check(context.Background(), step.Env{FS: fs})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected satisfied=false for missing dest")
	}
}

func TestGitClone_Check_DestExists(t *testing.T) {
	step.RegisterGitClone()
	s, _ := step.Build(manifest.Step{
		Type: "git_clone", Name: "g",
		Fields: map[string]any{"url": "u", "dest": "/d"},
	})
	fs := xfs.NewFake()
	if err := fs.MkdirAll("/d", 0o755); err != nil {
		t.Fatal(err)
	}
	ok, err := s.Check(context.Background(), step.Env{FS: fs})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected satisfied=true when dest is a directory")
	}
}

func TestGitClone_Run_Clones_WhenMissing(t *testing.T) {
	step.RegisterGitClone()
	s, _ := step.Build(manifest.Step{
		Type: "git_clone", Name: "g",
		Fields: map[string]any{"url": "https://example.com/r.git", "dest": "/d"},
	})
	fs := xfs.NewFake() // dest missing
	fake := xexec.NewFake()
	fake.Script("git", []string{"clone", "https://example.com/r.git", "/d"}, xexec.Result{Lines: []string{"Cloning..."}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake, FS: fs}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(sink.events))
	}
}

func TestGitClone_Run_Pulls_WhenPresent(t *testing.T) {
	step.RegisterGitClone()
	s, _ := step.Build(manifest.Step{
		Type: "git_clone", Name: "g",
		Fields: map[string]any{"url": "u", "dest": "/d"},
	})
	fs := xfs.NewFake()
	if err := fs.MkdirAll("/d", 0o755); err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("git", []string{"-C", "/d", "pull", "--ff-only"}, xexec.Result{Lines: []string{"Already up to date."}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake, FS: fs}, sink)
	if err != nil {
		t.Fatal(err)
	}
	calls := fake.Calls()
	if len(calls) != 1 || calls[0].Args[0] != "-C" {
		t.Errorf("expected pull, got %+v", calls)
	}
}

func TestGitClone_HomeExpansion(t *testing.T) {
	step.RegisterGitClone()
	s, _ := step.Build(manifest.Step{
		Type: "git_clone", Name: "g",
		Fields: map[string]any{"url": "u", "dest": "~/.foo"},
	})
	fs := xfs.NewFake()
	fake := xexec.NewFake()
	fake.Script("git", []string{"clone", "u", "/home/me/.foo"}, xexec.Result{})
	err := s.Run(context.Background(), step.Env{Exec: fake, FS: fs, HomeDir: "/home/me"}, &collectSink{})
	if err != nil {
		t.Fatal(err)
	}
}
