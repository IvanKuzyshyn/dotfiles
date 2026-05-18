package tool_test

import (
	"strings"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func TestNewRegistry_BuildsAndStores(t *testing.T) {
	step.RegisterShell()
	ms := []manifest.Tool{
		{Name: "a", Steps: []manifest.Step{{Type: "shell", Fields: map[string]any{"install": "true"}}}},
		{Name: "b", Steps: []manifest.Step{{Type: "shell", Fields: map[string]any{"install": "true"}}}},
	}
	r, err := tool.NewRegistry(ms)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := r.Get("a")
	if !ok || got.Name != "a" {
		t.Errorf("Get(a) = (%+v, %v)", got, ok)
	}
	all := r.All()
	if len(all) != 2 || all[0].Name != "a" || all[1].Name != "b" {
		t.Errorf("All() = %v", all)
	}
}

func TestNewRegistry_DuplicateNames(t *testing.T) {
	step.RegisterShell()
	ms := []manifest.Tool{
		{Name: "x", Steps: []manifest.Step{{Type: "shell", Fields: map[string]any{"install": "true"}}}},
		{Name: "x", Steps: []manifest.Step{{Type: "shell", Fields: map[string]any{"install": "true"}}}},
	}
	_, err := tool.NewRegistry(ms)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate error, got %v", err)
	}
}

func TestNewRegistry_StepBuildError(t *testing.T) {
	step.RegisterShell()
	ms := []manifest.Tool{
		{Name: "x", Steps: []manifest.Step{{Type: "unknown_type"}}},
	}
	_, err := tool.NewRegistry(ms)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected unknown-type error, got %v", err)
	}
}

func TestGet_Missing(t *testing.T) {
	r, _ := tool.NewRegistry(nil)
	if _, ok := r.Get("nope"); ok {
		t.Error("Get(nope) should return ok=false")
	}
}
