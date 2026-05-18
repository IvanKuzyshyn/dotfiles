package step

import (
	"context"
	"errors"
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

// shellStep is the manifest's escape-hatch step type. It runs arbitrary
// sh -c commands for install/update with optional check predicates.
type shellStep struct {
	name    string
	check   string
	install string
	update  string
}

func newShell(name string, fields map[string]any) (Step, error) {
	install, _ := stringField(fields, "install")
	if install == "" {
		return nil, fmt.Errorf("shell step %q: 'install' field is required", name)
	}
	check, _ := stringField(fields, "check")
	update, _ := stringField(fields, "update")
	return &shellStep{name: name, check: check, install: install, update: update}, nil
}

func (s *shellStep) Type() string { return "shell" }
func (s *shellStep) Name() string { return s.name }

// Check returns true if the check predicate exits zero. If no check is
// configured, returns false (so the step always runs).
func (s *shellStep) Check(ctx context.Context, env Env) (bool, error) {
	if s.check == "" {
		return false, nil
	}
	err := env.Exec.Run(ctx, "sh", []string{"-c", s.check}, nil, nopLineSink{})
	if err == nil {
		return true, nil
	}
	var xe *xexec.Error
	if errors.As(err, &xe) {
		return false, nil // non-zero exit = unsatisfied, not an error
	}
	return false, fmt.Errorf("shell step %q check: %w", s.name, err)
}

// Run executes install (or update if Check is satisfied and update is set).
func (s *shellStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	cmd := s.install
	if s.update != "" {
		ok, err := s.Check(ctx, env)
		if err != nil {
			return err
		}
		if ok {
			cmd = s.update
		}
	}
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "sh", []string{"-c", cmd}, nil, ls)
}

// stringField extracts a string field from the inline-decoded map. Returns
// ("", false) if absent or not a string.
func stringField(fields map[string]any, key string) (string, bool) {
	v, ok := fields[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// nopLineSink discards lines (used during Check where output isn't surfaced).
type nopLineSink struct{}

func (nopLineSink) Line(string) {}

// lineSinkBridge adapts xexec.LineSink output into event.LogLine events.
type lineSinkBridge struct {
	sink event.Sink
	tool string
	step string
}

func (b lineSinkBridge) Line(s string) {
	b.sink.Send(event.Event{
		Kind:  event.LogLine,
		Tool:  b.tool,
		Step:  b.step,
		Line:  s,
		Level: event.LevelInfo,
	})
}

// ctxTool returns the tool name from context, or "" if unset. The runner
// stores the current tool name via a private context key; outside the
// runner (e.g. step unit tests) it returns "".
type ctxKey int

const ctxKeyTool ctxKey = 0

func ctxTool(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyTool).(string)
	return v
}

// WithTool returns a derived context carrying the tool name. Called by the
// runner before invoking each step so log events carry the right Tool field.
func WithTool(parent context.Context, tool string) context.Context {
	return context.WithValue(parent, ctxKeyTool, tool)
}

// RegisterShell registers the shell step type with the package registry.
// Safe to call multiple times; no-ops if already registered. Exported so
// tests that reset the registry can restore this type.
func RegisterShell() {
	for _, t := range RegisteredTypes() {
		if t == "shell" {
			return
		}
	}
	Register("shell", newShell)
}

func init() { RegisterShell() }
