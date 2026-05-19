package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

// ConflictResolutionMsg is emitted when the user confirms a choice in the
// modal. The App is responsible for forwarding it to the Sink so the runner's
// resolver channel unblocks. ApplyAll signals that the same choice should be
// reused for every subsequent conflict in this run.
type ConflictResolutionMsg struct {
	Action   event.ConflictAction
	ApplyAll bool
}

// ConflictModal renders an overlay prompting the user how to resolve a single
// linker conflict. Triggered by App in response to RunEventMsg{ConflictPrompt}.
// Selection is confirmed with Enter, which emits ConflictResolutionMsg. The App
// is responsible for forwarding that choice to the Sink's resolver channel.
type ConflictModal struct {
	target       string
	existingKind string
	choices      []event.ConflictAction // canonical: Backup, Overwrite, Skip, Abort
	cursor       int                    // index into choices
	applyAll     bool                   // "Apply to remaining" toggle

	width, height int
}

// NewConflictModal builds a modal for one conflict prompt. The Resolver
// channel on the event.Conflict is intentionally not retained — the App
// holds that wiring via the Sink so the modal stays pure UI.
func NewConflictModal(c event.Conflict) ConflictModal {
	return ConflictModal{
		target:       c.TargetPath,
		existingKind: c.ExistingKind,
		choices: []event.ConflictAction{
			event.ConflictBackup,
			event.ConflictOverwrite,
			event.ConflictSkip,
			event.ConflictAbort,
		},
	}
}

// Update advances the modal in response to a Bubble Tea message. Enter or Esc
// returns a Cmd carrying a ConflictResolutionMsg; the App is expected to
// forward that to the Sink and clear its modal pointer.
func (c ConflictModal) Update(msg tea.Msg) (ConflictModal, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = m.Width
		c.height = m.Height
		return c, nil

	case tea.KeyMsg:
		switch m.String() {
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
			}
			return c, nil
		case "down", "j":
			if c.cursor < len(c.choices)-1 {
				c.cursor++
			}
			return c, nil
		case " ", "space":
			c.applyAll = !c.applyAll
			return c, nil
		case "enter":
			action := c.choices[c.cursor]
			applyAll := c.applyAll
			return c, func() tea.Msg {
				return ConflictResolutionMsg{Action: action, ApplyAll: applyAll}
			}
		case "esc":
			// Esc is an explicit shortcut for Abort; ApplyAll is forced false
			// because aborting once already terminates the whole run.
			return c, func() tea.Msg {
				return ConflictResolutionMsg{Action: event.ConflictAbort, ApplyAll: false}
			}
		}
	}
	return c, nil
}

// View renders the modal body. When width/height are unknown it returns the
// bare text without box styling so tests can assert on plain content. When
// sized, the bordered card is centered within the available viewport so it
// reads as an overlay rather than a header.
func (c ConflictModal) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", conflictTitleStyle.Render("Conflict"))
	fmt.Fprintf(&b, "%s %s\n", conflictLabelStyle.Render("at:"), c.target)
	fmt.Fprintf(&b, "%s %s\n\n", conflictLabelStyle.Render("existing:"), c.existingKind)

	for i, action := range c.choices {
		mark := "  "
		label := conflictActionLabel(action)
		if i == c.cursor {
			mark = conflictCursorStyle.Render("▸ ")
			label = conflictCursorStyle.Render(label)
		}
		fmt.Fprintf(&b, "%s%s\n", mark, label)
	}

	b.WriteByte('\n')
	applyMark := "[ ]"
	if c.applyAll {
		applyMark = conflictApplyOnStyle.Render("[x]")
	}
	fmt.Fprintf(&b, "%s Apply to remaining   %s\n",
		applyMark, conflictHintStyle.Render("(space to toggle)"))

	b.WriteByte('\n')
	b.WriteString(conflictHintStyle.Render("↑/↓ move · enter confirm · esc abort"))

	body := b.String()
	if c.width == 0 || c.height == 0 {
		return body
	}
	card := conflictBoxStyle.Render(body)
	return lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, card)
}

// conflictActionLabel renders an action's human-readable name.
func conflictActionLabel(a event.ConflictAction) string {
	switch a {
	case event.ConflictBackup:
		return "Backup"
	case event.ConflictOverwrite:
		return "Overwrite"
	case event.ConflictSkip:
		return "Skip"
	case event.ConflictAbort:
		return "Abort"
	}
	return "?"
}

var (
	conflictCursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	conflictTitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	conflictLabelStyle   = lipgloss.NewStyle().Faint(true)
	conflictHintStyle    = lipgloss.NewStyle().Faint(true)
	conflictApplyOnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	conflictBoxStyle     = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("11")).
				Padding(1, 3)
)
