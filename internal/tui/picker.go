package tui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// StartRunMsg is emitted by the Picker when the user confirms a selection
// with Enter. The TUI app forwards it to the runner goroutine (Task 44).
type StartRunMsg struct {
	Tools []*tool.Tool
}

// pickerItem adapts a *tool.Tool for use as a bubbles list.Item. FilterValue
// joins the name and tags so substring filter (`/`) matches both.
type pickerItem struct {
	tool *tool.Tool
}

func (p pickerItem) FilterValue() string {
	return p.tool.Name + " " + strings.Join(p.tool.Tags, " ")
}

// Picker is the tool-selection screen. It wraps bubbles list.Model with a
// custom delegate that draws a checkbox prefix and tracks which tools the
// user has toggled on.
type Picker struct {
	list     list.Model
	tools    []*tool.Tool        // canonical order, never mutated
	selected map[string]struct{} // set of tool names

	tagFilter string   // "" means "all tags"
	tags      []string // distinct tags across tools, sorted

	width, height int
}

// NewPicker constructs a Picker for the given tools. The tools slice is
// retained as the canonical order; the bubbles list is initialized with one
// item per tool.
func NewPicker(tools []*tool.Tool) Picker {
	selected := make(map[string]struct{})

	items := make([]list.Item, 0, len(tools))
	for _, t := range tools {
		items = append(items, pickerItem{tool: t})
	}

	// The delegate captures the selected map (a reference type) directly
	// rather than a back-pointer to the Picker, so it keeps working when
	// callers store the Picker by value and reassign it on every Update.
	l := list.New(items, pickerDelegate{selected: selected}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)

	return Picker{
		list:     l,
		tools:    tools,
		selected: selected,
		tags:     distinctTags(tools),
	}
}

// Update advances the picker state in response to a Bubble Tea message. The
// returned Cmd is non-nil only when Enter triggers a StartRunMsg.
func (p Picker) Update(msg tea.Msg) (Picker, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = m.Width
		p.height = m.Height
		// Leave a couple of rows for the status header.
		p.list.SetSize(m.Width, max(0, m.Height-2))
		return p, nil

	case tea.KeyMsg:
		// While the user is typing into the filter input, forward all keys
		// to the list so the input behaves normally.
		if p.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			p.list, cmd = p.list.Update(msg)
			return p, cmd
		}

		switch m.String() {
		case " ", "space":
			p.toggleCurrent()
			return p, nil
		case "a":
			p.toggleAllVisible()
			return p, nil
		case "t":
			return p, p.cycleTag()
		case "enter":
			if len(p.selected) == 0 {
				return p, nil
			}
			return p, p.emitStartRun()
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View renders a status header, the bubbles list, and a key-hint footer.
func (p Picker) View() string {
	tag := p.tagFilter
	if tag == "" {
		tag = "all tags"
	} else {
		tag = "tag=" + tag
	}
	// Highlight the selected-count when anything is picked so the user can
	// tell the action is "armed".
	count := fmt.Sprintf("%d/%d", len(p.selected), len(p.tools))
	if len(p.selected) > 0 {
		count = pickerCountActiveStyle.Render(count)
	}
	status := pickerStatusStyle.Render("[") + count + pickerStatusStyle.Render(" selected · "+tag+"]")
	footer := pickerHelpStyle.Render("space toggle · a all · t cycle tag · / filter · enter run · q quit")
	return status + "\n" + p.list.View() + "\n" + footer
}

// toggleCurrent flips the selection state of the item under the cursor.
func (p *Picker) toggleCurrent() {
	item, ok := p.list.SelectedItem().(pickerItem)
	if !ok {
		return
	}
	name := item.tool.Name
	if _, on := p.selected[name]; on {
		delete(p.selected, name)
	} else {
		p.selected[name] = struct{}{}
	}
}

// toggleAllVisible selects every currently-visible item, or deselects them
// all if they are already fully selected.
func (p *Picker) toggleAllVisible() {
	visible := p.list.VisibleItems()
	if len(visible) == 0 {
		return
	}
	allOn := true
	for _, it := range visible {
		pi, ok := it.(pickerItem)
		if !ok {
			continue
		}
		if _, on := p.selected[pi.tool.Name]; !on {
			allOn = false
			break
		}
	}
	for _, it := range visible {
		pi, ok := it.(pickerItem)
		if !ok {
			continue
		}
		if allOn {
			delete(p.selected, pi.tool.Name)
		} else {
			p.selected[pi.tool.Name] = struct{}{}
		}
	}
}

// cycleTag advances tagFilter to the next tag in p.tags, wrapping back to
// "" (all tags) after the last one. Re-populates list items to match. The
// returned Cmd carries the re-filter command from list.SetItems, which is
// non-nil while the bubbles list is in FilterApplied state.
func (p *Picker) cycleTag() tea.Cmd {
	if len(p.tags) == 0 {
		return nil
	}
	// States are "" then each tag in order, length = len(tags)+1.
	cur := -1
	for i, t := range p.tags {
		if t == p.tagFilter {
			cur = i
			break
		}
	}
	next := cur + 1
	if next >= len(p.tags) {
		p.tagFilter = ""
	} else {
		p.tagFilter = p.tags[next]
	}
	return p.list.SetItems(p.filteredItems())
}

// filteredItems returns the canonical tool slice filtered by tagFilter.
func (p *Picker) filteredItems() []list.Item {
	out := make([]list.Item, 0, len(p.tools))
	for _, t := range p.tools {
		if p.tagFilter != "" && !hasTag(t.Tags, p.tagFilter) {
			continue
		}
		out = append(out, pickerItem{tool: t})
	}
	return out
}

// emitStartRun returns a Cmd that yields StartRunMsg with the currently
// selected tools, in canonical (registry) order.
func (p Picker) emitStartRun() tea.Cmd {
	picked := make([]*tool.Tool, 0, len(p.selected))
	for _, t := range p.tools {
		if _, on := p.selected[t.Name]; on {
			picked = append(picked, t)
		}
	}
	return func() tea.Msg { return StartRunMsg{Tools: picked} }
}

// distinctTags returns the union of all tools' Tags, sorted ascending.
func distinctTags(tools []*tool.Tool) []string {
	set := make(map[string]struct{})
	for _, t := range tools {
		for _, tag := range t.Tags {
			set[tag] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// pickerDelegate renders each row as `[x] name — description` with the
// cursor row inverse-styled. It holds the shared selection set (a map, so
// the reference survives Picker value-copies between Update calls).
type pickerDelegate struct {
	selected map[string]struct{}
}

func (d pickerDelegate) Height() int                             { return 1 }
func (d pickerDelegate) Spacing() int                            { return 0 }
func (d pickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(pickerItem)
	if !ok {
		return
	}
	_, on := d.selected[pi.tool.Name]
	box := "[ ]"
	if on {
		box = "[x]"
	}
	line := fmt.Sprintf("%s %s", box, pi.tool.Name)
	if pi.tool.Description != "" {
		line += " — " + pi.tool.Description
	}
	if index == m.Index() {
		line = pickerCursorStyle.Render(line)
	} else if on {
		line = pickerSelectedStyle.Render(line)
	}
	fmt.Fprint(w, line) //nolint:errcheck
}

var (
	pickerStatusStyle      = lipgloss.NewStyle().Faint(true)
	pickerHelpStyle        = lipgloss.NewStyle().Faint(true)
	pickerCountActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	pickerCursorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	pickerSelectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)
