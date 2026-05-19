package tool_test

import (
	"strings"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func newReg(t *testing.T, mts []manifest.Tool) *tool.Registry {
	t.Helper()
	step.RegisterShell()
	r, err := tool.NewRegistry(mts)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func shell(install string) manifest.Step {
	return manifest.Step{Type: "shell", Fields: map[string]any{"install": install}}
}

func TestSelect_All(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "a", Steps: []manifest.Step{shell("a")}},
		{Name: "b", Steps: []manifest.Step{shell("b")}},
	})
	got, err := tool.Select(r, nil, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 tools, got %d", len(got))
	}
}

func TestSelect_Tag(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "a", Tags: []string{"gui"}, Steps: []manifest.Step{shell("a")}},
		{Name: "b", Tags: []string{"cli"}, Steps: []manifest.Step{shell("b")}},
		{Name: "c", Tags: []string{"gui", "cli"}, Steps: []manifest.Step{shell("c")}},
	})
	got, err := tool.Select(r, nil, false, "gui")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "c" {
		t.Errorf("want [a,c], got %v", got)
	}
}

func TestSelect_Names(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "git", Steps: []manifest.Step{shell("a")}},
		{Name: "vim", Steps: []manifest.Step{shell("b")}},
	})
	got, err := tool.Select(r, []string{"git"}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "git" {
		t.Errorf("got %v", got)
	}
}

func TestSelect_UnknownNameWithSuggestion(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "git", Steps: []manifest.Step{shell("a")}},
		{Name: "github-cli", Steps: []manifest.Step{shell("b")}},
	})
	_, err := tool.Select(r, []string{"git-no-such"}, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("want unknown-tool, got %v", err)
	}
}

func TestExpandDeps_Transitive(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "a", DependsOn: []string{"b"}, Steps: []manifest.Step{shell("a")}},
		{Name: "b", DependsOn: []string{"c"}, Steps: []manifest.Step{shell("b")}},
		{Name: "c", Steps: []manifest.Step{shell("c")}},
		{Name: "d", Steps: []manifest.Step{shell("d")}}, // unrelated
	})
	a, _ := r.Get("a")
	got, err := tool.ExpandDeps(r, []*tool.Tool{a}, false)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, t := range got {
		names = append(names, t.Name)
	}
	want := []string{"a", "b", "c"}
	if len(names) != 3 || names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Errorf("want %v, got %v", want, names)
	}
}

func TestExpandDeps_NoDepsErrorsOnMissing(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "a", DependsOn: []string{"b"}, Steps: []manifest.Step{shell("a")}},
	})
	a, _ := r.Get("a")
	_, err := tool.ExpandDeps(r, []*tool.Tool{a}, true)
	if err == nil || !strings.Contains(err.Error(), "depends on") {
		t.Errorf("want depends-on error, got %v", err)
	}
}

func TestSort_TopologicalDeterministic(t *testing.T) {
	r := newReg(t, []manifest.Tool{
		{Name: "a", DependsOn: []string{"b", "c"}, Steps: []manifest.Step{shell("a")}},
		{Name: "b", DependsOn: []string{"d"}, Steps: []manifest.Step{shell("b")}},
		{Name: "c", DependsOn: []string{"d"}, Steps: []manifest.Step{shell("c")}},
		{Name: "d", Steps: []manifest.Step{shell("d")}},
	})
	in := r.All()
	out := tool.Sort(in)
	names := []string{}
	for _, t := range out {
		names = append(names, t.Name)
	}
	// d must come before b, c. b and c before a. Ties broken alphabetically.
	// Expected: [d, b, c, a]
	want := []string{"d", "b", "c", "a"}
	if len(names) != len(want) {
		t.Fatalf("want len %d, got %v", len(want), names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("at %d: want %q got %q (full: %v)", i, want[i], names[i], names)
		}
	}
}
