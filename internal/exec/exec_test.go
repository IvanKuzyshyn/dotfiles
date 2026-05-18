package exec_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

type capture struct{ lines []string }

func (c *capture) Line(s string) { c.lines = append(c.lines, s) }

func TestRealExec_Echo(t *testing.T) {
	c := &capture{}
	err := xexec.Real{}.Run(context.Background(), "sh", []string{"-c", "echo hello"}, nil, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.lines) == 0 || !strings.Contains(c.lines[0], "hello") {
		t.Errorf("expected hello in output, got %v", c.lines)
	}
}

func TestRealExec_NonZero(t *testing.T) {
	err := xexec.Real{}.Run(context.Background(), "sh", []string{"-c", "exit 7"}, nil, &capture{})
	if err == nil {
		t.Fatal("expected error")
	}
	var xe *xexec.Error
	if !errors.As(err, &xe) || xe.ExitCode != 7 {
		t.Fatalf("expected ExitCode 7, got %v", err)
	}
}
