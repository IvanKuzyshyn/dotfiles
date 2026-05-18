package tui

import (
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func mkTools() []*tool.Tool {
	return []*tool.Tool{
		{Name: "alpha", Description: "first", Tags: []string{"cli"}},
		{Name: "beta", Description: "second", Tags: []string{"cli", "lang"}},
		{Name: "gamma", Description: "third", Tags: []string{"lang"}},
	}
}

func TestPicker_Construction(t *testing.T) {
	tools := mkTools()
	p := NewPicker(tools)

	if got, want := len(p.list.Items()), len(tools); got != want {
		t.Errorf("list items = %d, want %d", got, want)
	}
	if len(p.selected) != 0 {
		t.Errorf("expected empty selection, got %d entries", len(p.selected))
	}
	if p.tagFilter != "" {
		t.Errorf("tagFilter = %q, want empty", p.tagFilter)
	}
	wantTags := []string{"cli", "lang"}
	if !reflect.DeepEqual(p.tags, wantTags) {
		t.Errorf("tags = %v, want %v", p.tags, wantTags)
	}
}

func TestPicker_ToggleSelection(t *testing.T) {
	p := NewPicker(mkTools())

	// Cursor starts at index 0 (alpha). Space toggles selection on.
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeySpace})
	if _, on := p.selected["alpha"]; !on {
		t.Fatalf("alpha not selected after first space")
	}

	// Second space deselects.
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeySpace})
	if _, on := p.selected["alpha"]; on {
		t.Fatalf("alpha still selected after second space")
	}
}

func TestPicker_SelectAll(t *testing.T) {
	tools := mkTools()
	p := NewPicker(tools)

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if got, want := len(p.selected), len(tools); got != want {
		t.Fatalf("after first 'a': selected=%d, want %d", got, want)
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if got := len(p.selected); got != 0 {
		t.Fatalf("after second 'a': selected=%d, want 0", got)
	}
}

func TestPicker_TagCycle(t *testing.T) {
	p := NewPicker(mkTools()) // tags: ["cli", "lang"]

	if p.tagFilter != "" {
		t.Fatalf("initial tagFilter = %q, want empty", p.tagFilter)
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.tagFilter != "cli" {
		t.Fatalf("after first 't': tagFilter = %q, want cli", p.tagFilter)
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.tagFilter != "lang" {
		t.Fatalf("after second 't': tagFilter = %q, want lang", p.tagFilter)
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if p.tagFilter != "" {
		t.Fatalf("after third 't': tagFilter = %q, want empty (wrapped)", p.tagFilter)
	}
}

func TestPicker_TagFilterShrinksItems(t *testing.T) {
	p := NewPicker(mkTools())
	// Cycle to "cli" — only alpha and beta have it.
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if got, want := len(p.list.Items()), 2; got != want {
		t.Fatalf("with tag=cli: items=%d, want %d", got, want)
	}
}

func TestPicker_EnterEmitsStartRunMsg(t *testing.T) {
	tools := mkTools()
	p := NewPicker(tools)

	// down, space, down, space, enter — select items at indices 1 and 2 (beta, gamma).
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeySpace})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeySpace})
	p, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("enter with selection produced no Cmd")
	}
	msg := cmd()
	startRun, ok := msg.(StartRunMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want StartRunMsg", msg)
	}
	if len(startRun.Tools) != 2 {
		t.Fatalf("StartRunMsg.Tools len=%d, want 2", len(startRun.Tools))
	}
	if startRun.Tools[0].Name != "beta" || startRun.Tools[1].Name != "gamma" {
		t.Errorf("selected = [%s, %s], want [beta, gamma]",
			startRun.Tools[0].Name, startRun.Tools[1].Name)
	}
}

func TestPicker_EnterWithEmptySelectionIsNoop(t *testing.T) {
	p := NewPicker(mkTools())
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("enter with empty selection should be a no-op, got cmd %v", cmd)
	}
}

// TestPicker_TeatestSequence runs the spec's key sequence through teatest to
// confirm the picker drives the full Bubble Tea loop without panicking and
// reaches the expected selection state.
func TestPicker_TeatestSequence(t *testing.T) {
	reg, err := tool.NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	app := NewApp(reg, step.Env{})
	// Replace the empty picker with one backed by our three test tools so
	// the on-screen list has content.
	app.picker = NewPicker(mkTools())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	// Quit so the test model terminates cleanly.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
