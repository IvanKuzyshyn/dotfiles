package step

import (
	"context"
	"errors"
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

// brewCaskStep installs a brew cask. The package field is required
// and specifies the cask name.
type brewCaskStep struct {
	name string
	pkg  string
}

func newBrewCask(name string, fields map[string]any) (Step, error) {
	if name == "" {
		name = "brew_cask"
	}
	pkg, _ := stringField(fields, "package")
	if pkg == "" {
		return nil, fmt.Errorf("brew_cask step: 'package' field is required")
	}
	return &brewCaskStep{name: name, pkg: pkg}, nil
}

func (s *brewCaskStep) Type() string { return "brew_cask" }
func (s *brewCaskStep) Name() string { return s.name }

func (s *brewCaskStep) Check(ctx context.Context, env Env) (bool, error) {
	err := env.Exec.Run(ctx, "brew", []string{"list", "--cask", s.pkg}, nil, nopLineSink{})
	if err == nil {
		return true, nil
	}
	var xe *xexec.Error
	if errors.As(err, &xe) {
		return false, nil
	}
	return false, fmt.Errorf("brew_cask %q check: %w", s.name, err)
}

func (s *brewCaskStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "brew", []string{"install", "--cask", s.pkg}, nil, ls)
}

// RegisterBrewCask is the idempotent registration helper (mirrors
// RegisterShell). Tests that wipe the registry call this to restore state.
func RegisterBrewCask() {
	if !isRegistered("brew_cask") {
		Register("brew_cask", newBrewCask)
	}
}

func init() { RegisterBrewCask() }
