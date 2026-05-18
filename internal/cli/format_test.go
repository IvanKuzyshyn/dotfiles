package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/runner"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func mkTR(name string, state runner.State, err error) runner.ToolResult {
	return runner.ToolResult{Tool: &tool.Tool{Name: name}, State: state, Err: err}
}

func TestFormatSummary_AllSucceeded(t *testing.T) {
	r := runner.Result{Tools: []runner.ToolResult{
		mkTR("a", runner.Succeeded, nil),
		mkTR("b", runner.Succeeded, nil),
	}}
	got := FormatSummary(r, "/tmp/log.txt")
	if !strings.Contains(got, "2 succeeded, 0 skipped, 0 failed") {
		t.Errorf("missing counts line: %s", got)
	}
	if strings.Contains(got, "Failed tools") {
		t.Errorf("should not have Failed tools section: %s", got)
	}
	if strings.Contains(got, "Retry failed") {
		t.Errorf("should not have Retry section: %s", got)
	}
	if !strings.Contains(got, "/tmp/log.txt") {
		t.Errorf("missing log path: %s", got)
	}
}

func TestFormatSummary_WithFailures(t *testing.T) {
	r := runner.Result{Tools: []runner.ToolResult{
		mkTR("a", runner.Succeeded, nil),
		mkTR("bad", runner.Failed, errors.New("exit 1")),
		mkTR("dep", runner.Skipped, errors.New("dependency \"bad\" failed")),
	}}
	got := FormatSummary(r, "")
	if !strings.Contains(got, "1 succeeded, 1 skipped, 1 failed") {
		t.Errorf("missing counts: %s", got)
	}
	if !strings.Contains(got, "✗ bad: exit 1") {
		t.Errorf("missing failed tool detail: %s", got)
	}
	if !strings.Contains(got, "~ dep") {
		t.Errorf("missing skipped tool: %s", got)
	}
	if !strings.Contains(got, "Retry failed: dot install bad") {
		t.Errorf("missing retry hint: %s", got)
	}
	if strings.Contains(got, "Logs:") {
		t.Errorf("should not have Logs section when logPath empty: %s", got)
	}
}

func TestFormatSummary_TruncatesLongError(t *testing.T) {
	long := strings.Repeat("x", 500)
	r := runner.Result{Tools: []runner.ToolResult{
		mkTR("bad", runner.Failed, errors.New(long)),
	}}
	got := FormatSummary(r, "")
	if !strings.Contains(got, "...") {
		t.Errorf("expected truncation marker '...': %s", got)
	}
}
