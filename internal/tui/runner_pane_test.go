package tui

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func mkRunnerTools() []*tool.Tool {
	return []*tool.Tool{
		{Name: "alpha", Description: "first"},
		{Name: "beta", Description: "second"},
		{Name: "gamma", Description: "third"},
	}
}

func TestRunnerPane_Construction(t *testing.T) {
	tools := mkRunnerTools()
	r := NewRunnerPane(tools)

	if got, want := len(r.tools), len(tools); got != want {
		t.Fatalf("tools len=%d, want %d", got, want)
	}
	for _, tl := range tools {
		if got := r.status[tl.Name]; got != StatusPending {
			t.Errorf("status[%s]=%v, want StatusPending", tl.Name, got)
		}
		buf, ok := r.logs[tl.Name]
		if !ok {
			t.Errorf("logs missing entry for %s", tl.Name)
			continue
		}
		if got := len(buf.Lines()); got != 0 {
			t.Errorf("logs[%s] len=%d, want 0", tl.Name, got)
		}
	}
	if r.onSummary {
		t.Error("onSummary should be false on construction")
	}
}

func TestRunnerPane_ZeroValueIsSafe(t *testing.T) {
	var r RunnerPane
	if got := r.View(); got != "" {
		t.Errorf("zero View()=%q, want empty", got)
	}
	updated, cmd := r.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Errorf("zero Update returned cmd, want nil")
	}
	if updated.View() != "" {
		t.Error("zero pane View() after Update should still be empty")
	}
}

func TestRunnerPane_ToolStartedUpdatesStatus(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "alpha"}})
	if got := r.status["alpha"]; got != StatusRunning {
		t.Errorf("status[alpha]=%v, want StatusRunning", got)
	}
}

func TestRunnerPane_LogLineAppendsInOrder(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	for _, line := range []string{"one", "two", "three"} {
		r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.LogLine, Tool: "alpha", Line: line}})
	}
	got := r.logs["alpha"].Lines()
	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("logs[alpha] len=%d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("logs[alpha][%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestRunnerPane_RingBufferCapsAt200(t *testing.T) {
	buf := newRingBuffer(logBufferCap)
	for i := 0; i < 250; i++ {
		buf.Push(strconv.Itoa(i))
	}
	lines := buf.Lines()
	if len(lines) != logBufferCap {
		t.Fatalf("len(Lines)=%d, want %d", len(lines), logBufferCap)
	}
	// First retained line should be the 50th push (250 - 200 = 50).
	if lines[0] != "50" {
		t.Errorf("first line=%q, want %q", lines[0], "50")
	}
	if lines[len(lines)-1] != "249" {
		t.Errorf("last line=%q, want %q", lines[len(lines)-1], "249")
	}
}

func TestRunnerPane_ToolFinishedMarksSucceeded(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "alpha"}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "alpha"}})
	if got := r.status["alpha"]; got != StatusSucceeded {
		t.Errorf("status[alpha]=%v, want StatusSucceeded", got)
	}
}

func TestRunnerPane_AllTerminalFlipsOnSummary(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "alpha"}})
	if r.onSummary {
		t.Fatal("onSummary should still be false after 1/3 terminal")
	}
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFailed, Tool: "beta", Err: errors.New("boom")}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolSkipped, Tool: "gamma"}})
	if !r.onSummary {
		t.Fatal("onSummary should be true after all tools terminal")
	}
}

func TestRunnerPane_TabCyclesFocus(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	if r.focused != 0 {
		t.Fatalf("initial focus=%d, want 0", r.focused)
	}
	for i := 0; i < 4; i++ {
		r, _ = r.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	if r.focused != 1 {
		t.Errorf("focus after 4 tabs=%d, want 1", r.focused)
	}
}

func TestRunnerPane_RetryEmitsMsgOnlyWhenFailedOnSummary(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	// Not on summary yet: r should be a no-op.
	_, cmd := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Fatal("r before summary should be no-op")
	}

	// Drive all tools to terminal: alpha failed, others succeeded.
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFailed, Tool: "alpha", Err: errors.New("nope")}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "beta"}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "gamma"}})
	if !r.onSummary {
		t.Fatal("expected onSummary=true precondition")
	}

	// Focus alpha (already index 0) and press 'r' → RetryToolMsg.
	r, cmd = r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("r on failed focused tool should emit RetryToolMsg")
	}
	msg := cmd()
	retry, ok := msg.(RetryToolMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want RetryToolMsg", msg)
	}
	if retry.Tool == nil || retry.Tool.Name != "alpha" {
		t.Errorf("RetryToolMsg.Tool=%v, want alpha", retry.Tool)
	}

	// Tab to beta (succeeded) — r should be a no-op.
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, cmd = r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Error("r on succeeded tool should be no-op")
	}
}

func TestRunnerPane_LOpensFullLogForTerminalFocus(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	// alpha is still pending; 'l' should not open the full log.
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if r.fullLogOpen {
		t.Fatal("l should not open full log while focused tool is pending")
	}

	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.LogLine, Tool: "alpha", Line: "hi"}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "alpha"}})

	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !r.fullLogOpen {
		t.Fatal("l should open full log for terminal focused tool")
	}

	// Esc closes the viewport.
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if r.fullLogOpen {
		t.Fatal("esc should close full log view")
	}
}

// TestRunnerPane_SpecScenarioViewContainsState feeds the spec's event stream
// (ToolStarted, LogLine ×3, ToolFinished) and asserts the rendered View
// surfaces the tool name, a succeeded marker, and a recent log line.
func TestRunnerPane_SpecScenarioViewContainsState(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(tea.WindowSizeMsg{Width: 120, Height: 20})

	feed := []event.Event{
		{Kind: event.ToolStarted, Tool: "alpha"},
		{Kind: event.LogLine, Tool: "alpha", Line: "compiling"},
		{Kind: event.LogLine, Tool: "alpha", Line: "linking"},
		{Kind: event.LogLine, Tool: "alpha", Line: "all done"},
		{Kind: event.ToolFinished, Tool: "alpha"},
	}
	for _, e := range feed {
		r, _ = r.Update(RunEventMsg{Event: e})
	}

	view := r.View()
	if !strings.Contains(view, "alpha") {
		t.Errorf("View should contain tool name 'alpha', got:\n%s", view)
	}
	if !strings.Contains(view, "✓") {
		t.Errorf("View should contain succeeded marker '✓', got:\n%s", view)
	}
	if !strings.Contains(view, "all done") {
		t.Errorf("View should contain last log line 'all done', got:\n%s", view)
	}
}

// TestRunnerPane_SpinnerTickEmittedOnFirstRunning asserts the first transition
// into Running returns a non-nil tea.Cmd so the spinner animation can start.
func TestRunnerPane_SpinnerTickEmittedOnFirstRunning(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	_, cmd := r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "alpha"}})
	if cmd == nil {
		t.Fatal("first ToolStarted should return a non-nil cmd to kick off the spinner")
	}
}

// TestRunnerPane_SpinnerTickEmittedOnlyOnce asserts subsequent ToolStarted
// events do not re-emit a fresh tick (the spinner's own TickMsg loop keeps
// the animation going).
func TestRunnerPane_SpinnerTickEmittedOnlyOnce(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, cmd := r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "alpha"}})
	if cmd == nil {
		t.Fatal("precondition: first ToolStarted should emit a tick cmd")
	}
	_, cmd = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "beta"}})
	if cmd != nil {
		t.Fatal("second ToolStarted should not emit another tick cmd")
	}
}

// TestRunnerPane_EscFallsThroughWhenFullLogClosed asserts esc with no full log
// open is a no-op (no cmd, no state mutation) so the app-level handler can
// own the key in Task 44.
func TestRunnerPane_EscFallsThroughWhenFullLogClosed(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	if r.fullLogOpen {
		t.Fatal("precondition: fullLogOpen should be false on a fresh pane")
	}
	beforeFocus := r.focused
	updated, cmd := r.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Errorf("esc with no full log open should not emit a cmd, got %v", cmd)
	}
	if updated.fullLogOpen {
		t.Error("esc should not mutate fullLogOpen when it is already false")
	}
	if updated.focused != beforeFocus {
		t.Errorf("esc should not mutate focus, got %d want %d", updated.focused, beforeFocus)
	}
}

// TestRunnerPane_EscClosesFullLog asserts that when the full-log viewport is
// open, esc closes it.
func TestRunnerPane_EscClosesFullLog(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	// Bring alpha to a terminal state so 'l' is permitted to open the log.
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.LogLine, Tool: "alpha", Line: "hi"}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "alpha"}})
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !r.fullLogOpen {
		t.Fatal("precondition: 'l' should have opened the full log")
	}
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if r.fullLogOpen {
		t.Error("esc should close the full log viewport")
	}
}

func TestRunnerPane_FocusAutoFollowsRunningTool(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	if r.focused != 0 {
		t.Fatalf("initial focus=%d, want 0", r.focused)
	}
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "beta"}})
	if r.focused != 1 {
		t.Errorf("focus after ToolStarted beta=%d, want 1", r.focused)
	}
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "gamma"}})
	if r.focused != 2 {
		t.Errorf("focus after ToolStarted gamma=%d, want 2", r.focused)
	}
}

func TestRunnerPane_ManualTabStopsAutoFollow(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyTab})
	if r.focused != 1 {
		t.Fatalf("focus after tab=%d, want 1", r.focused)
	}
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolStarted, Tool: "gamma"}})
	if r.focused != 1 {
		t.Errorf("focus after manual+ToolStarted gamma=%d, want 1 (manual override)", r.focused)
	}
}

func TestRunnerPane_SummaryBannerWhenComplete(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFinished, Tool: "alpha"}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolFailed, Tool: "beta", Err: errors.New("nope")}})
	r, _ = r.Update(RunEventMsg{Event: event.Event{Kind: event.ToolSkipped, Tool: "gamma"}})
	if !r.onSummary {
		t.Fatal("expected onSummary=true")
	}
	view := r.View()
	if !strings.Contains(view, "1 succeeded") {
		t.Errorf("View missing '1 succeeded': %q", view)
	}
	if !strings.Contains(view, "1 failed") {
		t.Errorf("View missing '1 failed': %q", view)
	}
	if !strings.Contains(view, "1 skipped") {
		t.Errorf("View missing '1 skipped': %q", view)
	}
}

func TestRunnerPane_HelpFooterPresentInView(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := r.View()
	if !strings.Contains(view, "q") {
		t.Errorf("View missing 'q' help hint: %q", view)
	}
}
