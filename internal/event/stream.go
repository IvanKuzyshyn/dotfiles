package event

import (
	"fmt"
	"io"
)

// StreamSink writes formatted events to an io.Writer.
type StreamSink struct {
	W io.Writer
}

func (s StreamSink) Send(e Event) {
	switch e.Kind {
	case ToolStarted:
		fmt.Fprintf(s.W, "▸ %s\n", e.Tool)
	case StepStarted:
		fmt.Fprintf(s.W, "  → %s\n", e.Step)
	case LogLine:
		fmt.Fprintf(s.W, "    %s\n", e.Line)
	case StepSkipped:
		fmt.Fprintf(s.W, "  ~ %s (skipped)\n", e.Step)
	case StepFinished:
		fmt.Fprintf(s.W, "  ✓ %s\n", e.Step)
	case StepFailed:
		fmt.Fprintf(s.W, "  ✗ %s: %v\n", e.Step, e.Err)
	case ToolFinished:
		fmt.Fprintf(s.W, "✓ %s\n", e.Tool)
	case ToolFailed:
		if e.Err != nil {
			fmt.Fprintf(s.W, "✗ %s: %v\n", e.Tool, e.Err)
		} else {
			fmt.Fprintf(s.W, "✗ %s\n", e.Tool)
		}
	case ToolSkipped:
		fmt.Fprintf(s.W, "~ %s (skipped)\n", e.Tool)
	case ConflictPrompt:
		if e.Conflict != nil {
			fmt.Fprintf(s.W, "⚠ conflict at %s (%s)\n", e.Conflict.TargetPath, e.Conflict.ExistingKind)
		}
	case ConflictResolved:
		// nothing to print
	}
}
