package cli_test

import (
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/cli"
	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

func TestParseConflictAction(t *testing.T) {
	cases := []struct {
		in   string
		want event.ConflictAction
		err  bool
	}{
		{"backup", event.ConflictBackup, false},
		{"overwrite", event.ConflictOverwrite, false},
		{"skip", event.ConflictSkip, false},
		{"abort", event.ConflictAbort, false},
		{"", event.ConflictAbort, false}, // empty defaults to abort
		{"weird", 0, true},
	}
	for _, c := range cases {
		got, err := cli.ParseConflictAction(c.in)
		if c.err && err == nil {
			t.Errorf("%q: expected error", c.in)
		}
		if !c.err && err != nil {
			t.Errorf("%q: unexpected error %v", c.in, err)
		}
		if !c.err && got != c.want {
			t.Errorf("%q: got %v want %v", c.in, got, c.want)
		}
	}
}

type captureSink struct{ events []event.Event }

func (c *captureSink) Send(e event.Event) { c.events = append(c.events, e) }

func TestFlagResolverSink_ResolvesConflicts(t *testing.T) {
	inner := &captureSink{}
	sink := cli.FlagResolverSink{Inner: inner, Action: event.ConflictBackup}
	ch := make(chan event.ConflictAction, 1)
	sink.Send(event.Event{
		Kind: event.ConflictPrompt,
		Conflict: &event.Conflict{
			TargetPath:   "/x",
			ExistingKind: "file",
			Resolver:     ch,
		},
	})
	select {
	case got := <-ch:
		if got != event.ConflictBackup {
			t.Errorf("got %v, want Backup", got)
		}
	default:
		t.Fatal("expected resolver channel to receive an action")
	}
	if len(inner.events) != 1 {
		t.Errorf("inner sink should have received the event, got %d", len(inner.events))
	}
}

func TestFlagResolverSink_PassthroughOtherEvents(t *testing.T) {
	inner := &captureSink{}
	sink := cli.FlagResolverSink{Inner: inner, Action: event.ConflictAbort}
	sink.Send(event.Event{Kind: event.LogLine, Line: "ok"})
	if len(inner.events) != 1 || inner.events[0].Line != "ok" {
		t.Errorf("passthrough failed: %+v", inner.events)
	}
}
