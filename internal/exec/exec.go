// Package exec provides interfaces and implementations for executing external commands.
package exec

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// LineSink receives lines of output from a command.
type LineSink interface {
	Line(s string)
}

// Exec executes an external command.
type Exec interface {
	Run(ctx context.Context, cmd string, args []string, env []string, sink LineSink) error
}

// Error wraps command execution errors with exit code information.
type Error struct {
	ExitCode int
	Stderr   string
	err      error // underlying *exec.ExitError
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("exit %d", e.ExitCode)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *Error) Unwrap() error {
	return e.err
}

// Real is the real implementation of Exec.
type Real struct{}

// Run executes the command with the given arguments and environment,
// streaming lines to the sink. On non-zero exit, returns an Error.
func (Real) Run(ctx context.Context, cmd string, args []string, env []string, sink LineSink) error {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = append(os.Environ(), env...)

	// Create a pipe to capture both stdout and stderr merged together.
	out, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	c.Stderr = c.Stdout

	// Start the command.
	if err := c.Start(); err != nil {
		return err
	}

	// Stream output line-by-line to the sink.
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		sink.Line(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Wait for the command to complete.
	if err := c.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return &Error{ExitCode: ee.ExitCode(), err: ee}
		}
		return err
	}

	return nil
}
