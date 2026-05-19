//go:build teatest

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

// TestRunnerPane_Snapshot captures the split-pane View() after a synthetic
// event sequence: alpha starts and finishes, beta starts and emits one log
// line. The spinner is deliberately not advanced past its initial frame, so
// the snapshot is deterministic.
func TestRunnerPane_Snapshot(t *testing.T) {
	r := NewRunnerPane(mkRunnerTools())
	r, _ = r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	feed := []event.Event{
		{Kind: event.ToolStarted, Tool: "alpha"},
		{Kind: event.LogLine, Tool: "alpha", Line: "alpha line 1"},
		{Kind: event.ToolFinished, Tool: "alpha"},
		{Kind: event.ToolStarted, Tool: "beta"},
		{Kind: event.LogLine, Tool: "beta", Line: "beta line 1"},
	}
	for _, e := range feed {
		r, _ = r.Update(RunEventMsg{Event: e})
	}

	teatest.RequireEqualOutput(t, []byte(r.View()))
}
