//go:build teatest

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestPicker_Snapshot captures the picker's View() with a known set of tools,
// the cursor moved one row down, and the second tool toggled on. The output
// is compared against testdata/<TestName>.golden; regenerate with
// `make snapshot`.
func TestPicker_Snapshot(t *testing.T) {
	p := NewPicker(mkTools())
	p, _ = p.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeySpace})
	teatest.RequireEqualOutput(t, []byte(p.View()))
}
