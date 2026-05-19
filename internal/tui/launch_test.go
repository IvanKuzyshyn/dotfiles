package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// newTestLaunchApp builds a launchApp whose sink uses a fakeProg recorder so
// tests can assert on forwarded messages without spinning up Bubble Tea.
func newTestLaunchApp(t *testing.T) (launchApp, *fakeProg) {
	t.Helper()
	return newTestLaunchAppWithRegistry(t, nil)
}

// newTestLaunchAppWithRegistry is the same as newTestLaunchApp but lets the
// caller seed manifests so dependency expansion can be exercised.
func newTestLaunchAppWithRegistry(t *testing.T, manifests []manifest.Tool) (launchApp, *fakeProg) {
	t.Helper()
	reg, err := tool.NewRegistry(manifests)
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

// TestLaunchApp_StartRunExpandsDependencies ensures the TUI run respects
// depends_on like the CLI does. Picking only `claude` should also drag in
// `claude-code` and `nvm` so the user doesn't have to know the chain.
func TestLaunchApp_StartRunExpandsDependencies(t *testing.T) {
	manifests := []manifest.Tool{
		{Name: "nvm", Configs: []manifest.Config{{Source: "nvm", Target: "~"}}},
		{Name: "claude-code", DependsOn: []string{"nvm"}, Configs: []manifest.Config{{Source: "cc", Target: "~"}}},
		{Name: "claude", DependsOn: []string{"claude-code"}, Configs: []manifest.Config{{Source: "claude", Target: "~"}}},
	}
	la, _ := newTestLaunchAppWithRegistry(t, manifests)
	claude, ok := la.inner.reg.Get("claude")
	if !ok {
		t.Fatalf("registry missing claude")
	}

	got, _ := la.Update(StartRunMsg{Tools: []*tool.Tool{claude}})

	gotLA := got.(launchApp)
	names := make([]string, 0, len(gotLA.inner.runner.tools))
	for _, t := range gotLA.inner.runner.tools {
		names = append(names, t.Name)
	}
	want := map[string]bool{"claude": true, "claude-code": true, "nvm": true}
	if len(names) != len(want) {
		t.Fatalf("runner.tools = %v, want all three (claude + deps)", names)
	}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected tool %q in expanded run", n)
		}
	}
}
