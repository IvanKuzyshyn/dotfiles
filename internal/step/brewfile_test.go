package step_test

import (
	"context"
	"path/filepath"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestBrewfile_BuildMissingPath(t *testing.T) {
	step.RegisterBrewfile()
	_, err := step.Build(manifest.Step{Type: "brewfile", Name: "x"})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestBrewfile_CheckInstalled(t *testing.T) {
	step.RegisterBrewfile()
	s, _ := step.Build(manifest.Step{
		Type:   "brewfile",
		Name:   "bf",
		Fields: map[string]any{"path": "Brewfile"},
	})
	dotfiles := "/dotfiles"
	abs := filepath.Join(dotfiles, "Brewfile")
	fake := xexec.NewFake()
	fake.Script("brew", []string{"bundle", "check", "--file=" + abs}, xexec.Result{ExitCode: 0})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake, DotfilesDir: dotfiles})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected satisfied=true")
	}
}

func TestBrewfile_CheckMissing(t *testing.T) {
	step.RegisterBrewfile()
	s, _ := step.Build(manifest.Step{
		Type: "brewfile", Name: "bf",
		Fields: map[string]any{"path": "Brewfile"},
	})
	fake := xexec.NewFake()
	fake.Script("brew", []string{"bundle", "check", "--file=/d/Brewfile"}, xexec.Result{ExitCode: 1})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake, DotfilesDir: "/d"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected satisfied=false")
	}
}

func TestBrewfile_Run(t *testing.T) {
	step.RegisterBrewfile()
	s, _ := step.Build(manifest.Step{
		Type: "brewfile", Name: "bf",
		Fields: map[string]any{"path": "Brewfile"},
	})
	fake := xexec.NewFake()
	fake.Script("brew", []string{"bundle", "--file=/d/Brewfile"}, xexec.Result{Lines: []string{"==> Installing"}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake, DotfilesDir: "/d"}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 1 || sink.events[0].Line != "==> Installing" {
		t.Errorf("unexpected events: %+v", sink.events)
	}
}

func TestBrewfile_NoDotfilesDir(t *testing.T) {
	step.RegisterBrewfile()
	s, _ := step.Build(manifest.Step{
		Type: "brewfile", Name: "bf",
		Fields: map[string]any{"path": "Brewfile"},
	})
	_, err := s.Check(context.Background(), step.Env{Exec: xexec.NewFake()})
	if err == nil {
		t.Fatal("expected error for empty DotfilesDir")
	}
}
