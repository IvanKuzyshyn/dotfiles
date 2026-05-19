package step_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestBrewPackage_CheckInstalled(t *testing.T) {
	step.RegisterBrewPackage()
	s, err := step.Build(manifest.Step{Type: "brew_package", Fields: map[string]any{"package": "jq"}})
	if err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("brew", []string{"list", "--formula", "jq"}, xexec.Result{ExitCode: 0})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected installed=true")
	}
}

func TestBrewPackage_CheckMissing(t *testing.T) {
	step.RegisterBrewPackage()
	s, _ := step.Build(manifest.Step{Type: "brew_package", Fields: map[string]any{"package": "jq"}})
	fake := xexec.NewFake()
	fake.Script("brew", []string{"list", "--formula", "jq"}, xexec.Result{ExitCode: 1})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected installed=false")
	}
}

func TestBrewPackage_Run(t *testing.T) {
	step.RegisterBrewPackage()
	s, _ := step.Build(manifest.Step{Type: "brew_package", Name: "install-jq", Fields: map[string]any{"package": "jq"}})
	fake := xexec.NewFake()
	fake.Script("brew", []string{"install", "jq"}, xexec.Result{Lines: []string{"==> Installing jq"}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 1 || sink.events[0].Line != "==> Installing jq" {
		t.Errorf("unexpected events: %+v", sink.events)
	}
}

func TestBrewPackage_MissingPackageField(t *testing.T) {
	step.RegisterBrewPackage()
	_, err := step.Build(manifest.Step{Type: "brew_package"})
	if err == nil {
		t.Error("expected error for missing 'package' field")
	}
}
