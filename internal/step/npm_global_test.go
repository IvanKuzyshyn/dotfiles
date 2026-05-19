package step_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

func TestNpmGlobal_CheckInstalled(t *testing.T) {
	step.RegisterNpmGlobal()
	s, err := step.Build(manifest.Step{Type: "npm_global", Fields: map[string]any{"package": "claude-code"}})
	if err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("npm", []string{"list", "-g", "--depth=0", "claude-code"}, xexec.Result{ExitCode: 0})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected installed=true")
	}
}

func TestNpmGlobal_CheckMissing(t *testing.T) {
	step.RegisterNpmGlobal()
	s, _ := step.Build(manifest.Step{Type: "npm_global", Fields: map[string]any{"package": "claude-code"}})
	fake := xexec.NewFake()
	fake.Script("npm", []string{"list", "-g", "--depth=0", "claude-code"}, xexec.Result{ExitCode: 1})
	ok, err := s.Check(context.Background(), step.Env{Exec: fake})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected installed=false")
	}
}

func TestNpmGlobal_Run(t *testing.T) {
	step.RegisterNpmGlobal()
	s, _ := step.Build(manifest.Step{Type: "npm_global", Name: "install-claude-code", Fields: map[string]any{"package": "claude-code"}})
	fake := xexec.NewFake()
	fake.Script("npm", []string{"install", "-g", "claude-code"}, xexec.Result{Lines: []string{"added 42 packages"}})
	sink := &collectSink{}
	err := s.Run(context.Background(), step.Env{Exec: fake}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 1 || sink.events[0].Line != "added 42 packages" {
		t.Errorf("unexpected events: %+v", sink.events)
	}
}

func TestNpmGlobal_MissingPackageField(t *testing.T) {
	step.RegisterNpmGlobal()
	_, err := step.Build(manifest.Step{Type: "npm_global"})
	if err == nil {
		t.Error("expected error for missing 'package' field")
	}
}
