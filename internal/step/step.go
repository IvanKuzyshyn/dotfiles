// Package step defines the Step interface every install action implements,
// along with the runtime Env handed to each step.
package step

import (
	"context"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/platform"
)

// Env is the runtime context handed to each Step. Steps use Env.Exec for
// subprocess execution and Env.FS for filesystem access so the runner can
// substitute fakes in tests.
type Env struct {
	Exec        xexec.Exec
	FS          xfs.FS
	Platform    platform.Platform
	HomeDir     string
	DotfilesDir string
}

// Step is one declarative install/check action. Implementations live in
// sibling files (shell.go, brew_package.go, etc.) and register themselves
// with the package registry in their init().
type Step interface {
	// Type returns the manifest "type" string this step implements.
	Type() string
	// Name returns a human-readable identifier shown in events and logs.
	Name() string
	// Check reports whether the step's effect is already satisfied. When
	// true, the runner skips Run.
	Check(ctx context.Context, env Env) (bool, error)
	// Run performs the step. Streams log lines to sink as event.LogLine.
	Run(ctx context.Context, env Env, sink event.Sink) error
}
