package runner

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// State summarizes the runner outcome for one tool.
type State int

const (
	Succeeded State = iota
	Skipped         // dep failed or run cancelled
	Failed
)

// ToolResult records what happened to a single tool during Run.
type ToolResult struct {
	Tool  *tool.Tool
	State State
	Err   error // nil for Succeeded; set for Failed; reason for Skipped
}

// Result is the aggregated outcome of running a Plan.
type Result struct {
	Tools []ToolResult
}

// Counts returns (succeeded, skipped, failed).
func (r Result) Counts() (int, int, int) {
	var s, sk, f int
	for _, t := range r.Tools {
		switch t.State {
		case Succeeded:
			s++
		case Skipped:
			sk++
		case Failed:
			f++
		}
	}
	return s, sk, f
}

// AnyFailed reports whether any tool ended in Failed state.
func (r Result) AnyFailed() bool {
	_, _, f := r.Counts()
	return f > 0
}

// Run executes the plan tool-by-tool. Tools run sequentially. A failure
// in one tool fails that tool and skips any later tool that depends on
// it (transitively), but does not abort the run.
//
// Within a tool, steps run sequentially. If a step's Check reports the
// effect is already satisfied, the step is skipped and a StepSkipped
// event is emitted. Otherwise Run is called. A failure in any step fails
// the tool; remaining steps do not run.
//
// Panics in step code are recovered and reported as StepFailed.
//
// Context cancellation is honored before each tool starts: remaining
// tools emit ToolSkipped with a "cancelled" reason.
func Run(ctx context.Context, plan Plan, env step.Env, sink event.Sink) Result {
	var res Result
	failed := make(map[string]struct{})
	for _, t := range plan.Tools {
		// Context cancellation: skip remaining tools.
		select {
		case <-ctx.Done():
			sink.Send(event.Event{Kind: event.ToolSkipped, Tool: t.Name, Err: ctx.Err()})
			res.Tools = append(res.Tools, ToolResult{Tool: t, State: Skipped, Err: ctx.Err()})
			continue
		default:
		}
		// Dependency failure: skip.
		if dep, ok := firstFailedDep(t, failed); ok {
			err := fmt.Errorf("dependency %q failed", dep)
			sink.Send(event.Event{Kind: event.ToolSkipped, Tool: t.Name, Err: err})
			res.Tools = append(res.Tools, ToolResult{Tool: t, State: Skipped, Err: err})
			continue
		}
		sink.Send(event.Event{Kind: event.ToolStarted, Tool: t.Name})
		if err := runTool(ctx, t, env, sink); err != nil {
			failed[t.Name] = struct{}{}
			sink.Send(event.Event{Kind: event.ToolFailed, Tool: t.Name, Err: err})
			res.Tools = append(res.Tools, ToolResult{Tool: t, State: Failed, Err: err})
			continue
		}
		sink.Send(event.Event{Kind: event.ToolFinished, Tool: t.Name})
		res.Tools = append(res.Tools, ToolResult{Tool: t, State: Succeeded})
	}
	return res
}

// runTool executes one tool's steps. Returns the first step error.
func runTool(ctx context.Context, t *tool.Tool, env step.Env, sink event.Sink) error {
	stepCtx := step.WithTool(ctx, t.Name)
	for _, s := range t.Steps {
		stepName := s.Name()
		if stepName == "" {
			stepName = s.Type()
		}
		sink.Send(event.Event{Kind: event.StepStarted, Tool: t.Name, Step: stepName})

		// Check first — may skip.
		ok, err := checkSafe(stepCtx, s, env)
		if err != nil {
			sink.Send(event.Event{Kind: event.StepFailed, Tool: t.Name, Step: stepName, Err: err})
			return err
		}
		if ok {
			sink.Send(event.Event{Kind: event.StepSkipped, Tool: t.Name, Step: stepName})
			continue
		}

		if err := runSafe(stepCtx, s, env, sink); err != nil {
			sink.Send(event.Event{Kind: event.StepFailed, Tool: t.Name, Step: stepName, Err: err})
			return err
		}
		sink.Send(event.Event{Kind: event.StepFinished, Tool: t.Name, Step: stepName})
	}
	return nil
}

// firstFailedDep returns the name of the first DependsOn entry that's in
// the failed set, or empty/false if none.
func firstFailedDep(t *tool.Tool, failed map[string]struct{}) (string, bool) {
	for _, dep := range t.DependsOn {
		if _, bad := failed[dep]; bad {
			return dep, true
		}
	}
	return "", false
}

// checkSafe wraps Step.Check with panic recovery.
func checkSafe(ctx context.Context, s step.Step, env step.Env) (ok bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("step %q check panicked: %v\n%s", s.Name(), r, debug.Stack())
		}
	}()
	return s.Check(ctx, env)
}

// runSafe wraps Step.Run with panic recovery.
func runSafe(ctx context.Context, s step.Step, env step.Env, sink event.Sink) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("step %q run panicked: %v\n%s", s.Name(), r, debug.Stack())
		}
	}()
	return s.Run(ctx, env, sink)
}

// Ensure that ToolResult.Err can hold standard error wrapping.
var _ error = errors.New("")
