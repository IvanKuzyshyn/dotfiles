package exec

import (
	"context"
	"fmt"
	"reflect"
)

// Result is the scripted outcome of a fake Run call.
type Result struct {
	Lines    []string
	ExitCode int
	Err      error // optional non-Exit error
}

// Call records a single invocation of Fake.Run.
type Call struct {
	Cmd  string
	Args []string
	Env  []string
}

type scripted struct {
	cmd    string
	args   []string
	result Result
}

// Fake is a scripted Exec implementation for tests.
type Fake struct {
	scripts []scripted
	calls   []Call
}

// NewFake returns an empty Fake.
func NewFake() *Fake { return &Fake{} }

// Script registers a response for a (cmd, args) pair. First match wins.
func (f *Fake) Script(cmd string, args []string, result Result) {
	f.scripts = append(f.scripts, scripted{cmd: cmd, args: args, result: result})
}

// Calls returns a snapshot of every Run invocation in order.
func (f *Fake) Calls() []Call { return append([]Call(nil), f.calls...) }

// Run satisfies Exec.
func (f *Fake) Run(ctx context.Context, cmd string, args []string, env []string, sink LineSink) error {
	f.calls = append(f.calls, Call{Cmd: cmd, Args: append([]string(nil), args...), Env: append([]string(nil), env...)})
	for _, s := range f.scripts {
		if s.cmd == cmd && reflect.DeepEqual(s.args, args) {
			for _, line := range s.result.Lines {
				sink.Line(line)
			}
			if s.result.Err != nil {
				return s.result.Err
			}
			if s.result.ExitCode != 0 {
				return &Error{ExitCode: s.result.ExitCode}
			}
			return nil
		}
	}
	return fmt.Errorf("exec: unscripted command %q %v (loud failure: scripts %d, calls %d)", cmd, args, len(f.scripts), len(f.calls))
}
