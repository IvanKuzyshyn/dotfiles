package tui

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

// progSender is the subset of *tea.Program the Sink relies on. Defining it
// here lets tests substitute a recorder without spinning up a real Bubble Tea
// program. *tea.Program satisfies this interface natively.
type progSender interface {
	Send(msg tea.Msg)
}

// Sink bridges the runner's event stream to a tea.Program. It converts every
// event into a RunEventMsg and forwards it to the program. For ConflictPrompt
// events it additionally stashes the resolver channel so the App's modal
// handler can later unblock the runner via Resolve.
type Sink struct {
	prog      progSender
	resolvers sync.Map // targetPath (string) → chan<- event.ConflictAction
}

// NewSink wraps a tea.Program in a Sink ready to receive runner events.
func NewSink(prog *tea.Program) *Sink {
	return &Sink{prog: prog}
}

// Send implements event.Sink. ConflictPrompt events capture the resolver
// channel so Resolve can answer them later; every event is then forwarded to
// the program as a RunEventMsg.
func (s *Sink) Send(e event.Event) {
	if e.Kind == event.ConflictPrompt && e.Conflict != nil {
		s.resolvers.Store(e.Conflict.TargetPath, e.Conflict.Resolver)
	}
	s.prog.Send(RunEventMsg{Event: e})
}

// Resolve sends the user's choice into the channel registered by the most
// recent ConflictPrompt for targetPath. If no resolver is registered (or it
// was already answered), this is a no-op so callers don't have to track
// lifecycle.
func (s *Sink) Resolve(targetPath string, action event.ConflictAction) {
	if ch, ok := s.resolvers.LoadAndDelete(targetPath); ok {
		ch.(chan<- event.ConflictAction) <- action
	}
}
