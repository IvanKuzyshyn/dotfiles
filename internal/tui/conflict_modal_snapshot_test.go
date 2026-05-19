//go:build teatest

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestConflictModal_Snapshot captures the modal's View() with a known
// conflict and a cursor positioned on the second choice (Overwrite).
func TestConflictModal_Snapshot(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	teatest.RequireEqualOutput(t, []byte(m.View()))
}
