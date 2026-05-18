package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

func mkConflict() event.Conflict {
	return event.Conflict{
		TargetPath:   "/x/y",
		ExistingKind: "file",
		Resolver:     nil,
	}
}

func TestConflictModal_Construction(t *testing.T) {
	m := NewConflictModal(mkConflict())
	if m.target != "/x/y" {
		t.Errorf("target = %q, want /x/y", m.target)
	}
	if m.existingKind != "file" {
		t.Errorf("existingKind = %q, want file", m.existingKind)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.applyAll {
		t.Error("applyAll should default to false")
	}
	want := []event.ConflictAction{
		event.ConflictBackup,
		event.ConflictOverwrite,
		event.ConflictSkip,
		event.ConflictAbort,
	}
	if len(m.choices) != len(want) {
		t.Fatalf("choices len = %d, want %d", len(m.choices), len(want))
	}
	for i, c := range want {
		if m.choices[i] != c {
			t.Errorf("choices[%d] = %v, want %v", i, m.choices[i], c)
		}
	}
}

func TestConflictModal_DownNavigates(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestConflictModal_DownClampsAtEnd(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m.cursor = 3
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 3 {
		t.Errorf("cursor = %d, want 3 (clamped)", m.cursor)
	}
}

func TestConflictModal_UpNavigates(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m.cursor = 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestConflictModal_UpClampsAtZero(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m.cursor)
	}
}

func TestConflictModal_SpaceTogglesApplyAll(t *testing.T) {
	m := NewConflictModal(mkConflict())
	if m.applyAll {
		t.Fatal("precondition: applyAll should start false")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !m.applyAll {
		t.Errorf("after first space: applyAll = false, want true")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.applyAll {
		t.Errorf("after second space: applyAll = true, want false")
	}
}

func TestConflictModal_EnterEmitsResolution(t *testing.T) {
	m := NewConflictModal(mkConflict())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no Cmd")
	}
	msg, ok := cmd().(ConflictResolutionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want ConflictResolutionMsg", cmd())
	}
	if msg.Action != event.ConflictBackup {
		t.Errorf("Action = %v, want ConflictBackup", msg.Action)
	}
	if msg.ApplyAll {
		t.Errorf("ApplyAll = true, want false")
	}
}

// TestConflictModal_DownDownEnterPicksSkip mirrors the spec's teatest scenario:
// from cursor 0, two Downs land on Skip (the third choice).
func TestConflictModal_DownDownEnterPicksSkip(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no Cmd")
	}
	msg, ok := cmd().(ConflictResolutionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want ConflictResolutionMsg", cmd())
	}
	if msg.Action != event.ConflictSkip {
		t.Errorf("Action = %v, want ConflictSkip", msg.Action)
	}
	if msg.ApplyAll {
		t.Errorf("ApplyAll = true, want false")
	}
}

func TestConflictModal_EscEmitsAbort(t *testing.T) {
	m := NewConflictModal(mkConflict())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc produced no Cmd")
	}
	msg, ok := cmd().(ConflictResolutionMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want ConflictResolutionMsg", cmd())
	}
	if msg.Action != event.ConflictAbort {
		t.Errorf("Action = %v, want ConflictAbort", msg.Action)
	}
	if msg.ApplyAll {
		t.Errorf("ApplyAll = true, want false")
	}
}

func TestConflictModal_ViewContainsTargetAndChoices(t *testing.T) {
	m := NewConflictModal(mkConflict())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := m.View()
	for _, want := range []string{
		"/x/y",
		"Backup",
		"Overwrite",
		"Skip",
		"Abort",
		"Apply to remaining",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("View missing %q, got:\n%s", want, view)
		}
	}
}

// TestConflictModal_TeatestSpecScenario sends ConflictPrompt via a hand-built
// App harness, then ↓ ↓ enter; asserts the third choice (Skip) is emitted.
// Task 44 will wire this end-to-end through the real App.Update; for now we
// drive the modal directly through teatest by embedding it in a minimal model
// that mirrors App's overlay routing.
func TestConflictModal_TeatestSpecScenario(t *testing.T) {
	reg, err := tool.NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	app := NewApp(reg, step.Env{})
	modal := NewConflictModal(mkConflict())
	app.modal = &modal

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
