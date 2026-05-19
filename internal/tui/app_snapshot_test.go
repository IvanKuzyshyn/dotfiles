//go:build teatest

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// TestApp_Snapshot captures the top-level View() on the picker screen with a
// known three-tool registry. This exercises the App header + picker delegate
// rendering path end-to-end.
func TestApp_Snapshot(t *testing.T) {
	reg, err := tool.NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	app := NewApp(reg, step.Env{})
	// Replace the empty picker so the screen has deterministic content.
	app.picker = NewPicker(mkTools())

	m, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	rendered := m.(App).View()
	teatest.RequireEqualOutput(t, []byte(rendered))
}
