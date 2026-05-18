// Package tui implements the Bubble Tea front-end shown when `dot` is run
// without a subcommand. The App owns three sub-screens (picker, runner,
// summary) and an optional modal overlay; it routes key and window events
// to whichever screen is active.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// screen identifies which sub-model is currently rendered.
type screen int

const (
	screenPicker screen = iota
	screenRunner
	screenSummary
)

// App is the top-level Bubble Tea model. It holds the active sub-screen,
// an optional conflict-modal overlay, and the registry/env that downstream
// screens need to start a run.
type App struct {
	screen screen
	picker Picker
	runner RunnerPane
	modal  *ConflictModal // nil when not shown

	reg *tool.Registry
	env step.Env

	width, height int
}

// NewApp constructs an App with empty placeholder sub-screens. Real sub-screens
// will replace these placeholders in subsequent TUI tasks.
func NewApp(reg *tool.Registry, env step.Env) App {
	return App{
		screen: screenPicker,
		picker: NewPicker(reg.All()),
		runner: RunnerPane{},
		reg:    reg,
		env:    env,
	}
}

// Init satisfies tea.Model. The shell has no startup commands; sub-screens
// will return their own Cmds from their Update methods once implemented.
func (a App) Init() tea.Cmd { return nil }

// Update routes messages to the active sub-screen. Global keys (q, ctrl+c)
// quit regardless of which screen is showing. WindowSizeMsg is recorded on
// the App and forwarded to every sub-model so they can resize together.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Forward to every sub-model so they can size themselves
		// regardless of which one is currently visible.
		var cmds []tea.Cmd
		var c tea.Cmd
		a.picker, c = a.picker.Update(msg)
		cmds = append(cmds, c)
		a.runner, c = a.runner.Update(msg)
		cmds = append(cmds, c)
		if a.modal != nil {
			var updated ConflictModal
			updated, c = a.modal.Update(msg)
			*a.modal = updated
			cmds = append(cmds, c)
		}
		return a, tea.Batch(cmds...)
	}

	// Modal, when visible, swallows everything else.
	if a.modal != nil {
		updated, cmd := a.modal.Update(msg)
		*a.modal = updated
		return a, cmd
	}

	switch a.screen {
	case screenPicker:
		var cmd tea.Cmd
		a.picker, cmd = a.picker.Update(msg)
		return a, cmd
	case screenRunner, screenSummary:
		var cmd tea.Cmd
		a.runner, cmd = a.runner.Update(msg)
		return a, cmd
	}
	return a, nil
}

// View renders the active sub-screen with a minimal header. Modal overlay
// rendering is left to later tasks; for now we just delegate to the modal's
// View when it's visible.
func (a App) View() string {
	header := headerStyle.Render("dot")
	var body string
	switch a.screen {
	case screenPicker:
		body = a.picker.View()
	case screenRunner, screenSummary:
		body = a.runner.View()
	}
	if a.modal != nil {
		body = a.modal.View()
	}
	return header + "\n" + body
}

var headerStyle = lipgloss.NewStyle().Bold(true)

// --- Placeholder sub-models ---------------------------------------------
//
// The real RunnerPane and ConflictModal types arrive in Tasks 41 and 42.
// These stubs exist only so the App compiles and the shell's routing can
// be tested in isolation. Each stub mirrors the Bubble Tea Update/View
// shape but returns its own concrete type so the App can store updated
// values without type assertions.

// RunnerPane is replaced by a real split-pane view in Task 41.
type RunnerPane struct{}

// Update is a no-op pending the real implementation.
func (r RunnerPane) Update(_ tea.Msg) (RunnerPane, tea.Cmd) { return r, nil }

// View returns a placeholder string.
func (r RunnerPane) View() string { return "runner placeholder" }

// ConflictModal is replaced by a real overlay in Task 42.
type ConflictModal struct{}

// Update is a no-op pending the real implementation.
func (c ConflictModal) Update(_ tea.Msg) (ConflictModal, tea.Cmd) { return c, nil }

// View returns a placeholder string.
func (c ConflictModal) View() string { return "conflict modal placeholder" }
