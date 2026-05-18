package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// newTestLaunchApp builds a launchApp whose sink uses a fakeProg recorder so
// tests can assert on forwarded messages without spinning up Bubble Tea.
func newTestLaunchApp(t *testing.T) (launchApp, *fakeProg) {
	t.Helper()
	reg, err := tool.NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	fp := &fakeProg{}
	sink := &Sink{prog: fp}
	return launchApp{
		inner: NewApp(reg, step.Env{}),
		sink:  sink,
	}, fp
}

func TestLaunchApp_StartRunMsgSwitchesToRunnerScreen(t *testing.T) {
	la, _ := newTestLaunchApp(t)
	tool1 := &tool.Tool{Name: "foo"}

	got, cmd := la.Update(StartRunMsg{Tools: []*tool.Tool{tool1}})

	gotLA, ok := got.(launchApp)
	if !ok {
		t.Fatalf("Update returned %T, want launchApp", got)
	}
	if gotLA.inner.screen != screenRunner {
		t.Errorf("screen = %v, want screenRunner", gotLA.inner.screen)
	}
	if len(gotLA.inner.runner.tools) != 1 || gotLA.inner.runner.tools[0].Name != "foo" {
		t.Errorf("runner.tools = %+v, want [foo]", gotLA.inner.runner.tools)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from StartRunMsg (runner goroutine)")
	}
}

func TestLaunchApp_ConflictPromptOpensModal(t *testing.T) {
	la, _ := newTestLaunchApp(t)
	c := event.Conflict{
		TargetPath:   "/home/user/.gitconfig",
		ExistingKind: "file",
	}
	ev := event.Event{Kind: event.ConflictPrompt, Tool: "git", Conflict: &c}

	got, _ := la.Update(RunEventMsg{Event: ev})

	gotLA := got.(launchApp)
	if gotLA.inner.modal == nil {
		t.Fatal("expected modal to be set after ConflictPrompt")
	}
	if gotLA.inner.modal.target != "/home/user/.gitconfig" {
		t.Errorf("modal.target = %q, want /home/user/.gitconfig", gotLA.inner.modal.target)
	}
}

func TestLaunchApp_ConflictResolutionForwardsToSink(t *testing.T) {
	la, _ := newTestLaunchApp(t)
	resolverCh := make(chan event.ConflictAction, 1)
	c := event.Conflict{
		TargetPath:   "/x/y",
		ExistingKind: "file",
		Resolver:     resolverCh,
	}
	// Sink.Send is what registers the resolver in production (called from
	// the runner goroutine). Call it directly so the sink can later route
	// the user's choice into resolverCh.
	la.sink.Send(event.Event{Kind: event.ConflictPrompt, Tool: "git", Conflict: &c})
	// Then deliver the corresponding RunEventMsg through Update so the
	// launchApp opens its modal.
	model, _ := la.Update(RunEventMsg{Event: event.Event{Kind: event.ConflictPrompt, Tool: "git", Conflict: &c}})
	la = model.(launchApp)

	// Then resolve.
	model, _ = la.Update(ConflictResolutionMsg{Action: event.ConflictSkip})
	la = model.(launchApp)

	if la.inner.modal != nil {
		t.Errorf("modal should be cleared after ConflictResolutionMsg")
	}
	select {
	case got := <-resolverCh:
		if got != event.ConflictSkip {
			t.Errorf("resolver received %v, want ConflictSkip", got)
		}
	default:
		t.Fatal("resolver channel received nothing")
	}
}

func TestLaunchApp_RunCompletedSwitchesToSummary(t *testing.T) {
	la, _ := newTestLaunchApp(t)

	got, cmd := la.Update(RunCompletedMsg{})

	gotLA := got.(launchApp)
	if gotLA.inner.screen != screenSummary {
		t.Errorf("screen = %v, want screenSummary", gotLA.inner.screen)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd from RunCompletedMsg, got non-nil")
	}
}

func TestLaunchApp_RetryStartsRunForOneTool(t *testing.T) {
	la, _ := newTestLaunchApp(t)
	tool1 := &tool.Tool{Name: "foo"}

	got, cmd := la.Update(RetryToolMsg{Tool: tool1})

	gotLA := got.(launchApp)
	if gotLA.inner.screen != screenRunner {
		t.Errorf("screen = %v, want screenRunner after retry", gotLA.inner.screen)
	}
	if len(gotLA.inner.runner.tools) != 1 || gotLA.inner.runner.tools[0].Name != "foo" {
		t.Errorf("runner.tools = %+v, want [foo]", gotLA.inner.runner.tools)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from RetryToolMsg")
	}
}

func TestLaunchApp_RetryNilToolIsNoOp(t *testing.T) {
	la, _ := newTestLaunchApp(t)

	got, cmd := la.Update(RetryToolMsg{Tool: nil})

	gotLA := got.(launchApp)
	if gotLA.inner.screen != screenPicker {
		t.Errorf("screen changed unexpectedly on nil retry: %v", gotLA.inner.screen)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd for nil-tool retry")
	}
}

// TestLaunchApp_ForwardsUnhandledMsgsToInner sanity-checks that messages the
// launcher doesn't intercept (here a quit key) still reach the inner App.
func TestLaunchApp_ForwardsUnhandledMsgsToInner(t *testing.T) {
	la, _ := newTestLaunchApp(t)
	_, cmd := la.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit cmd to be forwarded from inner App")
	}
	if got := cmd(); got != (tea.QuitMsg{}) {
		t.Errorf("expected tea.QuitMsg, got %T", got)
	}
}
