// Package event defines the structured events emitted by the runner.
package event

// Level indicates the severity of a LogLine.
type Level int

const (
	LevelInfo Level = iota
	LevelWarn
	LevelError
)

// Kind discriminates event types.
type Kind int

const (
	ToolStarted Kind = iota
	StepStarted
	LogLine
	StepSkipped
	StepFinished
	StepFailed
	ToolFinished
	ToolFailed
	ToolSkipped
	ConflictPrompt
	ConflictResolved
)

// ConflictAction is the user's choice for a single linker conflict.
type ConflictAction int

const (
	ConflictBackup ConflictAction = iota
	ConflictOverwrite
	ConflictSkip
	ConflictAbort
)

// Conflict describes a linker conflict surfaced via a ConflictPrompt event.
// Resolver is a write channel; the consumer (CLI flag handler or TUI modal)
// sends the chosen action; the linker waits to receive it.
type Conflict struct {
	TargetPath   string
	ExistingKind string // "file" | "dir" | "symlink-other"
	Resolver     chan<- ConflictAction
}

// Event is the structured event emitted by the runner to a Sink.
type Event struct {
	Kind     Kind
	Tool     string
	Step     string
	Line     string    // for LogLine
	Level    Level     // for LogLine
	Err      error     // for *Failed
	Conflict *Conflict // non-nil only for ConflictPrompt
}

// Sink consumes events from the runner. Implementations: StreamSink (CLI),
// LogFileSink (per-run log file), Tee (fan-out), and TUISink (Phase 3).
type Sink interface {
	Send(Event)
}
