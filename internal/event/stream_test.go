package event_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

func TestStreamSink_FormatsKinds(t *testing.T) {
	var buf bytes.Buffer
	s := event.StreamSink{W: &buf}
	s.Send(event.Event{Kind: event.ToolStarted, Tool: "git"})
	s.Send(event.Event{Kind: event.StepStarted, Tool: "git", Step: "configure"})
	s.Send(event.Event{Kind: event.LogLine, Tool: "git", Step: "configure", Line: "ok"})
	s.Send(event.Event{Kind: event.StepFinished, Tool: "git", Step: "configure"})
	s.Send(event.Event{Kind: event.StepFailed, Tool: "git", Step: "x", Err: errors.New("boom")})
	s.Send(event.Event{Kind: event.ToolFinished, Tool: "git"})

	out := buf.String()
	for _, want := range []string{"git", "configure", "ok", "boom", "✓ git"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}
