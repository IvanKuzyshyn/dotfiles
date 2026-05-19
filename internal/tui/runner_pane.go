package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// logBufferCap bounds the rolling log buffer at ~200 lines per the spec.
const logBufferCap = 200

// RunEventMsg wraps a runner event for Bubble Tea delivery. The Sink (Task
// 43) sends one of these per event.Event it receives from the runner.
type RunEventMsg struct {
	Event event.Event
}

// RetryToolMsg is emitted when the user presses `r` on the summary screen
// while a failed tool is focused. Task 44 will handle it by restarting the
// runner for that single tool.
type RetryToolMsg struct {
	Tool *tool.Tool
}

// ToolStatus is the per-tool lifecycle state shown in the left pane.
type ToolStatus int

const (
	StatusPending ToolStatus = iota
	StatusRunning
	StatusSucceeded
	StatusFailed
	StatusSkipped
)

// RunnerPane is the split-pane runner screen. The left side lists tools with
// status icons; the right side shows a rolling log for the focused tool. The
// zero value is safe to use (View returns empty, Update is a no-op) so the
// App can hold an unpopulated pane until a run starts.
type RunnerPane struct {
	tools     []*tool.Tool
	status    map[string]ToolStatus
	logs      map[string]*ringBuffer
	focused   int
	onSummary bool

	// userFocused flips true once the user presses Tab. Until then the
	// focused tool auto-follows whichever tool last emitted ToolStarted, so
	// the user sees the active tool's log without having to chase it
	// manually.
	userFocused bool

	width, height int

	spinner     spinner.Model
	viewport    viewport.Model
	progress    progress.Model
	fullLogOpen bool
	tickStarted bool
}

// NewRunnerPane initializes a pane for the given tools list. Status starts at
// StatusPending for every tool.
func NewRunnerPane(tools []*tool.Tool) RunnerPane {
	status := make(map[string]ToolStatus, len(tools))
	logs := make(map[string]*ringBuffer, len(tools))
	for _, t := range tools {
		status[t.Name] = StatusPending
		logs[t.Name] = newRingBuffer(logBufferCap)
	}
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = runnerSpinnerStyle
	pb := progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	return RunnerPane{
		tools:    tools,
		status:   status,
		logs:     logs,
		spinner:  sp,
		viewport: viewport.New(0, 0),
		progress: pb,
	}
}

// Update advances the pane in response to a Bubble Tea message. RunEventMsg
// drives status and log updates; key messages handle focus/log/retry; window
// resize messages propagate to the viewport.
func (r RunnerPane) Update(msg tea.Msg) (RunnerPane, tea.Cmd) {
	switch m := msg.(type) {
	case RunEventMsg:
		cmd := r.applyEvent(m.Event)
		return r, cmd

	case tea.WindowSizeMsg:
		r.width = m.Width
		r.height = m.Height
		// Leave a header row above the runner; the App renders it.
		r.viewport.Width = m.Width
		r.viewport.Height = max(0, m.Height-2)
		// Cap the progress bar to something readable on wide terminals.
		barWidth := m.Width - 4
		if barWidth > 60 {
			barWidth = 60
		}
		if barWidth < 10 {
			barWidth = 10
		}
		r.progress.Width = barWidth
		if r.fullLogOpen {
			r.viewport.SetContent(r.focusedLogString())
		}
		return r, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		r.spinner, cmd = r.spinner.Update(msg)
		return r, cmd

	case progress.FrameMsg:
		newProg, cmd := r.progress.Update(msg)
		r.progress = newProg.(progress.Model)
		return r, cmd

	case tea.KeyMsg:
		if len(r.tools) == 0 {
			return r, nil
		}
		switch m.String() {
		case "tab":
			r.userFocused = true
			r.focused = (r.focused + 1) % len(r.tools)
			if r.fullLogOpen {
				r.viewport.SetContent(r.focusedLogString())
				r.viewport.GotoBottom()
			}
			return r, nil
		case "l":
			if r.fullLogOpen {
				r.fullLogOpen = false
				return r, nil
			}
			if r.isFocusedTerminal() {
				r.fullLogOpen = true
				r.viewport.SetContent(r.focusedLogString())
				r.viewport.GotoBottom()
			}
			return r, nil
		case "esc":
			// Only consume esc when the full-log viewport is open; otherwise
			// fall through so the app-level handler (Task 44) can route it.
			if r.fullLogOpen {
				r.fullLogOpen = false
				return r, nil
			}
		case "r":
			if r.onSummary && r.focusedStatus() == StatusFailed {
				t := r.tools[r.focused]
				return r, func() tea.Msg { return RetryToolMsg{Tool: t} }
			}
			return r, nil
		}

		// In full-log mode forward unhandled keys (arrows, pgup/pgdn) to the
		// viewport so the user can scroll the log.
		if r.fullLogOpen {
			var cmd tea.Cmd
			r.viewport, cmd = r.viewport.Update(msg)
			return r, cmd
		}
		return r, nil
	}

	return r, nil
}

// View renders the split pane (or the full-log viewport, when open). For a
// zero-value pane (no run yet) it returns an empty string so the App can
// safely call View on every screen. When the run is complete a summary
// banner appears above the pane; a help footer is always rendered below.
func (r RunnerPane) View() string {
	if len(r.tools) == 0 {
		return ""
	}
	var body string
	if r.fullLogOpen {
		body = r.viewport.View()
	} else {
		left := r.renderToolList()
		right := r.renderFocusedLog()
		// Fall back to a vertical stack when the terminal is too narrow for a
		// useful side-by-side layout.
		if r.width > 0 && r.width < 50 {
			body = left + "\n" + right
		} else {
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		}
	}
	var b strings.Builder
	if r.onSummary {
		b.WriteString(r.summaryBanner())
		b.WriteByte('\n')
	} else if len(r.tools) > 1 {
		// Live progress bar while the run is in flight. For single-tool runs
		// the bar is just noise; the spinner already conveys "I'm working".
		done := 0
		for _, t := range r.tools {
			switch r.status[t.Name] {
			case StatusSucceeded, StatusFailed, StatusSkipped:
				done++
			}
		}
		label := fmt.Sprintf(" %d/%d tools", done, len(r.tools))
		b.WriteString(r.progress.View() + runnerHelpStyle.Render(label))
		b.WriteByte('\n')
	}
	b.WriteString(body)
	b.WriteByte('\n')
	b.WriteString(r.helpFooter())
	return b.String()
}

// summaryBanner renders the end-of-run line showing tally + retry hint. The
// banner color reflects the worst outcome: red if anything failed, yellow if
// only skipped, green for all-success. Only invoked when r.onSummary is true.
func (r RunnerPane) summaryBanner() string {
	var s, f, sk int
	for _, st := range r.status {
		switch st {
		case StatusSucceeded:
			s++
		case StatusFailed:
			f++
		case StatusSkipped:
			sk++
		}
	}
	msg := fmt.Sprintf("Run complete: %d succeeded · %d failed · %d skipped", s, f, sk)
	style := runnerSummarySuccessStyle
	switch {
	case f > 0:
		style = runnerSummaryFailStyle
	case sk > 0:
		style = runnerSummarySkipStyle
	}
	return style.Render(msg)
}

// helpFooter renders a one-line key-hint footer. The available actions depend
// on whether the run is over and whether anything failed.
func (r RunnerPane) helpFooter() string {
	hints := []string{"tab focus", "l logs"}
	if r.onSummary {
		for _, st := range r.status {
			if st == StatusFailed {
				hints = append(hints, "r retry")
				break
			}
		}
	}
	hints = append(hints, "q quit")
	return runnerHelpStyle.Render(strings.Join(hints, " · "))
}

// applyEvent mutates pane state from a single runner event. Step events
// also push a synthetic log line so users see step boundaries inline. The
// returned tea.Cmd is non-nil only on the first transition into Running, to
// kick off the spinner tick; subsequent ticks are propagated by the spinner
// itself via spinner.TickMsg handling in Update.
func (r *RunnerPane) applyEvent(e event.Event) tea.Cmd {
	if e.Tool == "" {
		return nil
	}
	var cmd tea.Cmd
	switch e.Kind {
	case event.ToolStarted:
		r.status[e.Tool] = StatusRunning
		if !r.userFocused {
			for i, t := range r.tools {
				if t.Name == e.Tool {
					r.focused = i
					break
				}
			}
		}
		if !r.tickStarted {
			r.tickStarted = true
			cmd = r.spinner.Tick
		}
	case event.ToolFinished:
		r.status[e.Tool] = StatusSucceeded
		cmd = r.progress.SetPercent(r.completionRatio())
	case event.ToolFailed:
		r.status[e.Tool] = StatusFailed
		cmd = r.progress.SetPercent(r.completionRatio())
	case event.ToolSkipped:
		r.status[e.Tool] = StatusSkipped
		cmd = r.progress.SetPercent(r.completionRatio())
	case event.LogLine:
		r.pushLog(e.Tool, e.Line)
	case event.StepStarted:
		r.pushLog(e.Tool, "> step: "+e.Step)
	case event.StepFinished:
		r.pushLog(e.Tool, "< step: "+e.Step+" ok")
	case event.StepSkipped:
		r.pushLog(e.Tool, "~ step: "+e.Step+" skipped")
	case event.StepFailed:
		msg := "x step: " + e.Step + " failed"
		if e.Err != nil {
			msg += " (" + e.Err.Error() + ")"
		}
		r.pushLog(e.Tool, msg)
	case event.ConflictPrompt, event.ConflictResolved:
		// Modal handles conflicts; nothing to do here.
	}
	r.refreshSummaryFlag()
	return cmd
}

// completionRatio returns the fraction of tools in a terminal state. Used to
// drive the progress bar.
func (r RunnerPane) completionRatio() float64 {
	if len(r.tools) == 0 {
		return 0
	}
	done := 0
	for _, t := range r.tools {
		switch r.status[t.Name] {
		case StatusSucceeded, StatusFailed, StatusSkipped:
			done++
		}
	}
	return float64(done) / float64(len(r.tools))
}

// refreshSummaryFlag flips onSummary to true when every tool has reached a
// terminal status. It never flips back to false — once the run is over the
// summary screen owns the pane.
func (r *RunnerPane) refreshSummaryFlag() {
	for _, t := range r.tools {
		switch r.status[t.Name] {
		case StatusSucceeded, StatusFailed, StatusSkipped:
			continue
		default:
			return
		}
	}
	r.onSummary = true
}

func (r *RunnerPane) pushLog(name, line string) {
	buf, ok := r.logs[name]
	if !ok {
		buf = newRingBuffer(logBufferCap)
		r.logs[name] = buf
	}
	buf.Push(line)
	if r.fullLogOpen && r.focusedToolName() == name {
		r.viewport.SetContent(r.focusedLogString())
		r.viewport.GotoBottom()
	}
}

func (r RunnerPane) focusedToolName() string {
	if len(r.tools) == 0 {
		return ""
	}
	return r.tools[r.focused].Name
}

func (r RunnerPane) focusedStatus() ToolStatus {
	if len(r.tools) == 0 {
		return StatusPending
	}
	return r.status[r.tools[r.focused].Name]
}

func (r RunnerPane) isFocusedTerminal() bool {
	switch r.focusedStatus() {
	case StatusSucceeded, StatusFailed, StatusSkipped:
		return true
	}
	return false
}

func (r RunnerPane) focusedLogString() string {
	buf := r.logs[r.focusedToolName()]
	if buf == nil {
		return ""
	}
	return strings.Join(buf.Lines(), "\n")
}

// statusIcon returns the leading glyph for a tool row. For Running tools it
// returns the live spinner frame; for terminal states a static colored
// character.
func (r RunnerPane) statusIcon(s ToolStatus) string {
	switch s {
	case StatusRunning:
		return r.spinner.View()
	case StatusSucceeded:
		return runnerSucceededStyle.Render("✓")
	case StatusFailed:
		return runnerFailedStyle.Render("✗")
	case StatusSkipped:
		return runnerSkippedStyle.Render("~")
	default:
		return runnerPendingStyle.Render("·")
	}
}

func (r RunnerPane) renderToolList() string {
	var b strings.Builder
	for i, t := range r.tools {
		icon := r.statusIcon(r.status[t.Name])
		row := fmt.Sprintf("%s %s", icon, t.Name)
		if i == r.focused {
			row = runnerFocusedRowStyle.Render("> " + row)
		} else {
			row = "  " + row
		}
		b.WriteString(row)
		b.WriteByte('\n')
	}
	width := r.leftWidth()
	if width <= 0 {
		return b.String()
	}
	return lipgloss.NewStyle().Width(width).Render(b.String())
}

func (r RunnerPane) renderFocusedLog() string {
	buf := r.logs[r.focusedToolName()]
	if buf == nil {
		return ""
	}
	lines := buf.Lines()
	// Cap visible lines to the available height so we don't blow past the
	// terminal. height-2 accounts for the App's header.
	if r.height > 2 {
		visibleLines := r.height - 2
		if len(lines) > visibleLines {
			lines = lines[len(lines)-visibleLines:]
		}
	}
	content := strings.Join(lines, "\n")
	width := r.rightWidth()
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Render(content)
}

// leftWidth gives ~30% of total width to the tool list; the remainder goes to
// the log pane. Returns 0 when width is unknown so callers skip styling.
func (r RunnerPane) leftWidth() int {
	if r.width <= 0 {
		return 0
	}
	w := r.width * 3 / 10
	if w < 16 {
		w = 16
	}
	return w
}

func (r RunnerPane) rightWidth() int {
	if r.width <= 0 {
		return 0
	}
	return max(0, r.width-r.leftWidth())
}

// ANSI 16-color palette used so styling adapts to the user's terminal theme
// instead of locking to a fixed truecolor value. `10` is bright green, `9` is
// bright red, `11` is bright yellow, `14` is bright cyan.
var (
	runnerFocusedRowStyle     = lipgloss.NewStyle().Bold(true)
	runnerHelpStyle           = lipgloss.NewStyle().Faint(true)
	runnerSucceededStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	runnerFailedStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	runnerSkippedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	runnerSpinnerStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	runnerPendingStyle        = lipgloss.NewStyle().Faint(true)
	runnerSummarySuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	runnerSummaryFailStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	runnerSummarySkipStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
)

// ringBuffer is a fixed-capacity FIFO of strings used for per-tool log
// retention. Push wraps once capacity is reached; Lines returns entries in
// insertion order, oldest first.
type ringBuffer struct {
	buf      []string
	head     int // index of the oldest entry (when size == capacity)
	size     int
	capacity int
}

func newRingBuffer(capacity int) *ringBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &ringBuffer{capacity: capacity, buf: make([]string, 0, capacity)}
}

func (r *ringBuffer) Push(line string) {
	if r.size < r.capacity {
		r.buf = append(r.buf, line)
		r.size++
		return
	}
	r.buf[r.head] = line
	r.head = (r.head + 1) % r.capacity
}

func (r *ringBuffer) Lines() []string {
	out := make([]string, 0, r.size)
	if r.size < r.capacity {
		out = append(out, r.buf...)
		return out
	}
	for i := 0; i < r.size; i++ {
		out = append(out, r.buf[(r.head+i)%r.capacity])
	}
	return out
}
