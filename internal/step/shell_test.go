package step_test

import (
	"context"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

type collectSink struct{ events []event.Event }

func (c *collectSink) Send(e event.Event) { c.events = append(c.events, e) }

func TestShell_Build_MissingInstall(t *testing.T) {
	step.RegisterShell()
	_, err := step.Build(manifest.Step{
		Type: "shell",
		Name: "x",
	})
	if err == nil {
		t.Fatal("expected error for missing install")
	}
}

func TestShell_Check_Satisfied(t *testing.T) {
	step.RegisterShell()
	s, err := step.Build(manifest.Step{
		Type:   "shell",
		Name:   "check-only",
		Fields: map[string]any{"install": "true", "check": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("sh", []string{"-c", "true"}, xexec.Result{})
	env := step.Env{Exec: fake}
	ok, err := s.Check(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected satisfied (true)")
	}
}

func TestShell_Check_Unsatisfied(t *testing.T) {
	step.RegisterShell()
	s, err := step.Build(manifest.Step{
		Type:   "shell",
		Name:   "x",
		Fields: map[string]any{"install": "true", "check": "false"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := xexec.NewFake()
	fake.Script("sh", []string{"-c", "false"}, xexec.Result{ExitCode: 1})
	env := step.Env{Exec: fake}
	ok, err := s.Check(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected unsatisfied (false)")
	}
}

func TestShell_Check_NoCheckMeansFalse(t *testing.T) {
	step.RegisterShell()
	s, _ := step.Build(manifest.Step{
		Type:   "shell",
		Name:   "x",
		Fields: map[string]any{"install": "true"},
	})
	ok, err := s.Check(context.Background(), step.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false when check is empty")
	}
}

func TestShell_Run_StreamsLines(t *testing.T) {
	step.RegisterShell()
	s, _ := step.Build(manifest.Step{
		Type:   "shell",
		Name:   "runme",
		Fields: map[string]any{"install": "echo hello && echo world"},
	})
	fake := xexec.NewFake()
	fake.Script("sh", []string{"-c", "echo hello && echo world"}, xexec.Result{
		Lines: []string{"hello", "world"},
	})
	sink := &collectSink{}
	env := step.Env{Exec: fake}
	ctx := step.WithTool(context.Background(), "mytool")
	if err := s.Run(ctx, env, sink); err != nil {
		t.Fatal(err)
	}
	// Expect two LogLine events with Tool="mytool" Step="runme"
	if len(sink.events) != 2 {
		t.Fatalf("want 2 events, got %d", len(sink.events))
	}
	for i, want := range []string{"hello", "world"} {
		e := sink.events[i]
		if e.Kind != event.LogLine {
			t.Errorf("event %d: kind = %v, want LogLine", i, e.Kind)
		}
		if e.Tool != "mytool" || e.Step != "runme" {
			t.Errorf("event %d: tool=%q step=%q", i, e.Tool, e.Step)
		}
		if e.Line != want {
			t.Errorf("event %d: line=%q want=%q", i, e.Line, want)
		}
	}
}

func TestShell_Run_PrefersUpdateWhenSatisfied(t *testing.T) {
	step.RegisterShell()
	s, _ := step.Build(manifest.Step{
		Type: "shell",
		Name: "u",
		Fields: map[string]any{
			"check":   "test -d /already",
			"install": "git clone url /already",
			"update":  "git -C /already pull",
		},
	})
	fake := xexec.NewFake()
	fake.Script("sh", []string{"-c", "test -d /already"}, xexec.Result{}) // satisfied
	fake.Script("sh", []string{"-c", "git -C /already pull"}, xexec.Result{Lines: []string{"Already up to date."}})
	sink := &collectSink{}
	env := step.Env{Exec: fake}
	if err := s.Run(context.Background(), env, sink); err != nil {
		t.Fatal(err)
	}
	// Verify the second scripted call was selected (the update branch).
	calls := fake.Calls()
	if len(calls) != 2 {
		t.Fatalf("want 2 calls (Check + update), got %d", len(calls))
	}
	if calls[1].Args[1] != "git -C /already pull" {
		t.Errorf("expected update command in calls[1], got %q", calls[1].Args[1])
	}
}
