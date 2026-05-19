package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList_EmbeddedRegistry(t *testing.T) {
	g := &GlobalFlags{
		ConfigDir:   filepath.Join(t.TempDir(), "no-overlay"),
		DotfilesDir: "",
	}
	cmd := NewListCmd(g)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	// tools/example.yaml is embedded in the binary; it must appear in the list.
	for _, want := range []string{"NAME", "example", "embedded"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output:\n%s", want, got)
		}
	}
}

func TestList_WithOverlay(t *testing.T) {
	dir := t.TempDir()
	tools := filepath.Join(dir, "tools")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `
name: example
description: A test tool
platforms: [darwin]
tags: [cli, dev]
steps:
  - type: shell
    install: echo hi
`
	if err := os.WriteFile(filepath.Join(tools, "example.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	g := &GlobalFlags{ConfigDir: dir}
	cmd := NewListCmd(g)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, want := range []string{"NAME", "SOURCE", "PLATFORMS", "TAGS", "example", "overlay", "darwin", "cli,dev"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output:\n%s", want, got)
		}
	}
}

func TestSourceLabel(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "-"},
		{"embedded:foo.yaml", "embedded"},
		{"/abs/path/foo.yaml", "overlay"},
	}
	for _, c := range cases {
		if got := sourceLabel(c.in); got != c.want {
			t.Errorf("sourceLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
