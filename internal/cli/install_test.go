package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInstall_NoSelection(t *testing.T) {
	g := &GlobalFlags{}
	f := &installFlags{}
	err := runInstall(context.Background(), &bytes.Buffer{}, g, nil, f)
	if err == nil {
		t.Fatal("expected error when nothing is selected")
	}
	if !strings.Contains(err.Error(), "specify") {
		t.Errorf("want 'specify ...', got %v", err)
	}
	if ExitCode(err) != 2 {
		t.Errorf("want exit code 2 for pre-flight, got %d", ExitCode(err))
	}
}

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}
	if got := ExitCode(preflightErr{msg: "x", code: 2}); got != 2 {
		t.Errorf("ExitCode preflight = %d, want 2", got)
	}
	if got := ExitCode(errors.New("other")); got != 1 {
		t.Errorf("ExitCode other = %d, want 1", got)
	}
}
