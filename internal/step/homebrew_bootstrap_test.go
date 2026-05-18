package step_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestHomebrewBootstrap_CheckInstalled(t *testing.T) {
	step.RegisterHomebrewBootstrap()
	s, err := step.Build(manifest.Step{Type: "homebrew_bootstrap"})
	if err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("sh", []string{"-c", "command -v brew >/dev/null 2>&1"}, xexec.Result{ExitCode: 0})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected installed=true")
	}
}

func TestHomebrewBootstrap_CheckMissing(t *testing.T) {
	step.RegisterHomebrewBootstrap()
	s, _ := step.Build(manifest.Step{Type: "homebrew_bootstrap"})
	fake := xexec.NewFake()
	fake.Script("sh", []string{"-c", "command -v brew >/dev/null 2>&1"}, xexec.Result{ExitCode: 1})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected installed=false")
	}
}

func TestHomebrewBootstrap_Run(t *testing.T) {
	step.RegisterHomebrewBootstrap()
	s, _ := step.Build(manifest.Step{Type: "homebrew_bootstrap", Name: "bootstrap"})
	fake := xexec.NewFake()
	const installScript = `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
	fake.Script("sh", []string{"-c", installScript}, xexec.Result{Lines: []string{"==> Installing Homebrew"}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 1 || sink.events[0].Line != "==> Installing Homebrew" {
		t.Errorf("unexpected events: %+v", sink.events)
	}
}
