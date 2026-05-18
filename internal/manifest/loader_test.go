package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/platform"
)

func writeYAML(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadFile_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "git.yaml")
	writeYAML(t, p, `
name: git
description: VCS
steps:
  - type: shell
    install: echo hi
`)
	got, err := manifest.LoadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "git" || got.Description != "VCS" {
		t.Errorf("unexpected: %+v", got)
	}
	if got.Source == "" {
		t.Errorf("expected non-empty Source")
	}
}

func TestLoadDir_PartialFailure(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "ok.yaml"), `
name: ok
steps:
  - type: shell
    install: echo
`)
	writeYAML(t, filepath.Join(dir, "broken.yaml"), `name: [this: is: not: valid`)

	tools, err := manifest.LoadDir(dir)
	if err == nil {
		t.Fatal("expected joined error for broken file")
	}
	if len(tools) != 1 || tools[0].Name != "ok" {
		t.Errorf("expected 1 good tool, got %v", tools)
	}
}

func TestLoadDir_MissingDir(t *testing.T) {
	tools, err := manifest.LoadDir(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Errorf("missing dir should be nil error, got %v", err)
	}
	if tools != nil {
		t.Errorf("missing dir should return nil tools")
	}
}

func TestMerge_OverlayWinsByName(t *testing.T) {
	embedded := []manifest.Tool{{Name: "git", Description: "embedded"}}
	overlay := []manifest.Tool{{Name: "git", Description: "overlay"}}
	got := manifest.Merge(embedded, overlay)
	if len(got) != 1 || got[0].Description != "overlay" {
		t.Errorf("overlay should win: %+v", got)
	}
}

func TestMerge_PreservesNonConflicting(t *testing.T) {
	embedded := []manifest.Tool{{Name: "a"}, {Name: "b"}}
	overlay := []manifest.Tool{{Name: "c"}}
	got := manifest.Merge(embedded, overlay)
	if len(got) != 3 {
		t.Errorf("want 3 tools, got %d", len(got))
	}
	// Ordered alphabetically
	if got[0].Name != "a" || got[1].Name != "b" || got[2].Name != "c" {
		t.Errorf("not sorted: %v", got)
	}
}

func TestFilterPlatform(t *testing.T) {
	tools := []manifest.Tool{
		{Name: "any"}, // no platforms → kept
		{Name: "darwin-only", Platforms: []string{"darwin"}},
		{Name: "linux-only", Platforms: []string{"linux"}},
	}
	got := manifest.FilterPlatform(tools, platform.Platform{OS: "darwin"})
	names := []string{}
	for _, t := range got {
		names = append(names, t.Name)
	}
	if len(names) != 2 || names[0] != "any" || names[1] != "darwin-only" {
		t.Errorf("expected [any, darwin-only], got %v", names)
	}
}
