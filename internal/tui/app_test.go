package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func newTestApp(t *testing.T) App {
	t.Helper()
	reg, err := tool.NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	return NewApp(reg, step.Env{})
}

func TestApp_QuitOnQ(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected a quit cmd from 'q'")
	}
	if got := cmd(); got != (tea.QuitMsg{}) {
		t.Errorf("'q' should produce tea.QuitMsg, got %T", got)
	}
}

func TestApp_QuitOnCtrlC(t *testing.T) {
	a := newTestApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected a quit cmd from ctrl+c")
	}
	if got := cmd(); got != (tea.QuitMsg{}) {
		t.Errorf("ctrl+c should produce tea.QuitMsg, got %T", got)
	}
}

func TestApp_WindowSizePropagates(t *testing.T) {
	a := newTestApp(t)
	m, _ := a.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	got, ok := m.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", m)
	}
	if got.width != 100 || got.height != 40 {
		t.Errorf("width/height = %d/%d, want 100/40", got.width, got.height)
	}
}

func TestApp_TeatestQuit(t *testing.T) {
	a := newTestApp(t)
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
