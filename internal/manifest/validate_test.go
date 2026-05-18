package manifest_test

import (
	"strings"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
)

var knownTypes = []string{"shell", "brew_package"}

func TestValidate_MissingName(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Steps: []manifest.Step{{Type: "shell"}}},
	}, knownTypes)
	if err == nil || !strings.Contains(err.Error(), "missing name") {
		t.Errorf("expected missing name error, got %v", err)
	}
}

func TestValidate_NoSteps(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "x"},
	}, knownTypes)
	if err == nil || !strings.Contains(err.Error(), "no steps") {
		t.Errorf("expected no-steps error, got %v", err)
	}
}

func TestValidate_UnknownStepType(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "x", Steps: []manifest.Step{{Type: "made_up"}}},
	}, knownTypes)
	if err == nil || !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("expected unknown-type error, got %v", err)
	}
}

func TestValidate_UnknownDependsOn(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "x", DependsOn: []string{"y"}, Steps: []manifest.Step{{Type: "shell"}}},
	}, knownTypes)
	if err == nil || !strings.Contains(err.Error(), `unknown tool "y"`) {
		t.Errorf("expected unknown-dep error, got %v", err)
	}
}

func TestValidate_Cycle(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "a", DependsOn: []string{"b"}, Steps: []manifest.Step{{Type: "shell"}}},
		{Name: "b", DependsOn: []string{"c"}, Steps: []manifest.Step{{Type: "shell"}}},
		{Name: "c", DependsOn: []string{"a"}, Steps: []manifest.Step{{Type: "shell"}}},
	}, knownTypes)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle error, got %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "", Steps: []manifest.Step{{Type: "shell"}}},
		{Name: "y", Steps: []manifest.Step{{Type: "bogus"}}},
	}, knownTypes)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "missing name") || !strings.Contains(msg, "unknown type") {
		t.Errorf("expected both errors joined, got %v", err)
	}
}

func TestValidate_OK(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "a", DependsOn: []string{"b"}, Steps: []manifest.Step{{Type: "shell"}}},
		{Name: "b", Steps: []manifest.Step{{Type: "shell"}}},
	}, knownTypes)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_DuplicateName(t *testing.T) {
	err := manifest.Validate([]manifest.Tool{
		{Name: "a", Steps: []manifest.Step{{Type: "shell"}}},
		{Name: "a", Steps: []manifest.Step{{Type: "shell"}}},
	}, knownTypes)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate-name error, got %v", err)
	}
}
