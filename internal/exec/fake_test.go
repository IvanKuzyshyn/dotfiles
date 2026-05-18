package exec_test

import (
	"context"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

func TestFake_MatchByCmdAndArgs(t *testing.T) {
	f := xexec.NewFake()
	f.Script("brew", []string{"install", "jq"}, xexec.Result{Lines: []string{"==> Installing jq"}, ExitCode: 0})
	c := &capture{}
	if err := f.Run(context.Background(), "brew", []string{"install", "jq"}, nil, c); err != nil {
		t.Fatal(err)
	}
	if len(c.lines) != 1 || c.lines[0] != "==> Installing jq" {
		t.Fatalf("got %v", c.lines)
	}
}

func TestFake_NoMatch(t *testing.T) {
	f := xexec.NewFake()
	if err := f.Run(context.Background(), "brew", []string{"install", "jq"}, nil, &capture{}); err == nil {
		t.Error("expected unscripted-command error")
	}
}

func TestFake_NonZeroExit(t *testing.T) {
	f := xexec.NewFake()
	f.Script("brew", []string{"install", "bad"}, xexec.Result{ExitCode: 1})
	err := f.Run(context.Background(), "brew", []string{"install", "bad"}, nil, &capture{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFake_RecordsCalls(t *testing.T) {
	f := xexec.NewFake()
	f.Script("git", []string{"status"}, xexec.Result{ExitCode: 0})
	_ = f.Run(context.Background(), "git", []string{"status"}, nil, &capture{})
	calls := f.Calls()
	if len(calls) != 1 || calls[0].Cmd != "git" || calls[0].Args[0] != "status" {
		t.Fatalf("got %+v", calls)
	}
}
