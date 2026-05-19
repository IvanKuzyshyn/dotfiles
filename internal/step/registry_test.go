package step

import (
	"context"
	"errors"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
)

type fakeStep struct {
	name string
}

func (f *fakeStep) Type() string                                     { return "fake" }
func (f *fakeStep) Name() string                                     { return f.name }
func (f *fakeStep) Check(_ context.Context, _ Env) (bool, error)     { return false, nil }
func (f *fakeStep) Run(_ context.Context, _ Env, _ event.Sink) error { return nil }

func TestRegister_AndBuild(t *testing.T) {
	resetRegistryForTest()
	Register("fake", func(name string, fields map[string]any) (Step, error) {
		return &fakeStep{name: name}, nil
	})
	got, err := Build(manifest.Step{Type: "fake", Name: "f1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type() != "fake" || got.Name() != "f1" {
		t.Errorf("unexpected step: type=%q name=%q", got.Type(), got.Name())
	}
}

func TestBuild_UnknownType(t *testing.T) {
	resetRegistryForTest()
	_, err := Build(manifest.Step{Type: "nope"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	resetRegistryForTest()
	Register("fake", func(name string, _ map[string]any) (Step, error) {
		return &fakeStep{name: name}, nil
	})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()
	Register("fake", func(name string, _ map[string]any) (Step, error) {
		return &fakeStep{name: name}, nil
	})
}

func TestConstructorRejectsBadFields(t *testing.T) {
	resetRegistryForTest()
	Register("bad", func(_ string, _ map[string]any) (Step, error) {
		return nil, errors.New("bad fields")
	})
	_, err := Build(manifest.Step{Type: "bad"})
	if err == nil || err.Error() != "bad fields" {
		t.Errorf("expected 'bad fields', got %v", err)
	}
}

func TestRegisteredTypes_Sorted(t *testing.T) {
	resetRegistryForTest()
	noop := func(_ string, _ map[string]any) (Step, error) { return nil, nil }
	Register("zebra", noop)
	Register("apple", noop)
	Register("mango", noop)
	got := RegisteredTypes()
	want := []string{"apple", "mango", "zebra"}
	if len(got) != len(want) {
		t.Fatalf("len got=%d want=%d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("at %d: got %q want %q", i, got[i], want[i])
		}
	}
}
