package cli

import (
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

// ParseConflictAction maps the --on-conflict flag value to the event constant.
// Returns a friendly error on unknown values.
func ParseConflictAction(s string) (event.ConflictAction, error) {
	switch s {
	case "backup":
		return event.ConflictBackup, nil
	case "overwrite":
		return event.ConflictOverwrite, nil
	case "skip":
		return event.ConflictSkip, nil
	case "abort", "":
		return event.ConflictAbort, nil
	default:
		return 0, fmt.Errorf("invalid --on-conflict value %q (want backup|overwrite|skip|abort)", s)
	}
}

// FlagResolverSink wraps an underlying sink and auto-resolves any
// ConflictPrompt events with a preconfigured action. Other events pass
// through to the inner sink unchanged.
type FlagResolverSink struct {
	Inner  event.Sink
	Action event.ConflictAction
}

// Send forwards to Inner, then if the event is a ConflictPrompt with a
// resolver channel, sends the configured action to it.
func (s FlagResolverSink) Send(e event.Event) {
	s.Inner.Send(e)
	if e.Kind == event.ConflictPrompt && e.Conflict != nil && e.Conflict.Resolver != nil {
		// Non-blocking would lose the choice; we expect the resolver
		// channel to have capacity 1 (the runner allocates it as such).
		e.Conflict.Resolver <- s.Action
	}
}
