package step_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestBrewCask_CheckInstalled(t *testing.T) {
	step.RegisterBrewCask()
	s, err := step.Build(manifest.Step{Type: "brew_cask", Fields: map[string]any{"package": "ghostty"}})
	if err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("brew", []string{"list", "--cask", "ghostty"}, xexec.Result{ExitCode: 0})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected installed=true")
	}
}

func TestBrewCask_CheckMissing(t *testing.T) {
	step.RegisterBrewCask()
	s, _ := step.Build(manifest.Step{Type: "brew_cask", Fields: map[string]any{"package": "ghostty"}})
	fake := xexec.NewFake()
	fake.Script("brew", []string{"list", "--cask", "ghostty"}, xexec.Result{ExitCode: 1})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected installed=false")
	}
}

func TestBrewCask_Run(t *testing.T) {
	step.RegisterBrewCask()
	s, _ := step.Build(manifest.Step{Type: "brew_cask", Name: "install-ghostty", Fields: map[string]any{"package": "ghostty"}})
	fake := xexec.NewFake()
	fake.Script("brew", []string{"install", "--cask", "ghostty"}, xexec.Result{Lines: []string{"==> Installing Cask ghostty"}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 1 || sink.events[0].Line != "==> Installing Cask ghostty" {
		t.Errorf("unexpected events: %+v", sink.events)
	}
}

func TestBrewCask_MissingPackageField(t *testing.T) {
	step.RegisterBrewCask()
	_, err := step.Build(manifest.Step{Type: "brew_cask"})
	if err == nil {
		t.Error("expected error for missing 'package' field")
	}
}
