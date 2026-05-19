package manifest_test

import (
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"gopkg.in/yaml.v3"
)

func TestTool_MinimalUnmarshal(t *testing.T) {
	raw := `
name: git
steps:
  - type: shell
    install: echo hi
`
	var got manifest.Tool
	if err := yaml.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "git" {
		t.Errorf("name = %q, want %q", got.Name, "git")
	}
	if len(got.Steps) != 1 {
		t.Fatalf("want 1 step, got %d", len(got.Steps))
	}
	if got.Steps[0].Type != "shell" {
		t.Errorf("step type = %q, want shell", got.Steps[0].Type)
	}
	if got.Steps[0].Fields["install"] != "echo hi" {
		t.Errorf("step install field = %v, want 'echo hi'", got.Steps[0].Fields["install"])
	}
}

func TestTool_FullUnmarshal(t *testing.T) {
	raw := `
name: ghostty
description: Terminal emulator
platforms: [darwin]
tags: [gui, terminal]
depends_on: [homebrew]
steps:
  - type: brew_cask
    name: install ghostty
    package: ghostty
configs:
  - source: ghostty
    target: ~/.config/ghostty
`
	var got manifest.Tool
	if err := yaml.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "ghostty" || got.Description != "Terminal emulator" {
		t.Errorf("unexpected: %+v", got)
	}
	if len(got.Platforms) != 1 || got.Platforms[0] != "darwin" {
		t.Errorf("platforms = %v", got.Platforms)
	}
	if len(got.DependsOn) != 1 || got.DependsOn[0] != "homebrew" {
		t.Errorf("depends_on = %v", got.DependsOn)
	}
	if got.Steps[0].Fields["package"] != "ghostty" {
		t.Errorf("step package = %v", got.Steps[0].Fields["package"])
	}
	if len(got.Configs) != 1 || got.Configs[0].Source != "ghostty" {
		t.Errorf("configs = %+v", got.Configs)
	}
}

func TestTool_RoundTrip(t *testing.T) {
	in := manifest.Tool{
		Name:        "x",
		Description: "d",
		Platforms:   []string{"darwin"},
		Steps: []manifest.Step{{
			Type:   "shell",
			Name:   "do",
			Fields: map[string]any{"install": "echo"},
		}},
	}
	data, err := yaml.Marshal(&in)
	if err != nil {
		t.Fatal(err)
	}
	var out manifest.Tool
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || out.Description != in.Description {
		t.Errorf("round-trip mismatch: in=%+v out=%+v", in, out)
	}
}
