package step

import (
	"context"
	"errors"
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

const homebrewInstallScript = `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`

// homebrewBootstrapStep installs Homebrew if it's missing. It's the dependency
// every brew_package / brew_cask / brewfile step implicitly requires.
type homebrewBootstrapStep struct {
	name string
}

func newHomebrewBootstrap(name string, _ map[string]any) (Step, error) {
	if name == "" {
		name = "homebrew_bootstrap"
	}
	return &homebrewBootstrapStep{name: name}, nil
}

func (s *homebrewBootstrapStep) Type() string { return "homebrew_bootstrap" }
func (s *homebrewBootstrapStep) Name() string { return s.name }

func (s *homebrewBootstrapStep) Check(ctx context.Context, env Env) (bool, error) {
	err := env.Exec.Run(ctx, "sh", []string{"-c", "command -v brew >/dev/null 2>&1"}, nil, nopLineSink{})
	if err == nil {
		return true, nil
	}
	var xe *xexec.Error
	if errors.As(err, &xe) {
		return false, nil
	}
	return false, fmt.Errorf("homebrew_bootstrap %q check: %w", s.name, err)
}

func (s *homebrewBootstrapStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "sh", []string{"-c", homebrewInstallScript}, nil, ls)
}

// RegisterHomebrewBootstrap is the idempotent registration helper (mirrors
// RegisterShell). Tests that wipe the registry call this to restore state.
func RegisterHomebrewBootstrap() {
	if !isRegistered("homebrew_bootstrap") {
		Register("homebrew_bootstrap", newHomebrewBootstrap)
	}
}

func init() { RegisterHomebrewBootstrap() }
