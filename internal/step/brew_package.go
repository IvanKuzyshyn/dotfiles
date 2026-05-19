package step

import (
	"context"
	"errors"
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

// brewPackageStep installs a brew formula. The package field is required
// and specifies the formula name.
type brewPackageStep struct {
	name string
	pkg  string
}

func newBrewPackage(name string, fields map[string]any) (Step, error) {
	if name == "" {
		name = "brew_package"
	}
	pkg, _ := stringField(fields, "package")
	if pkg == "" {
		return nil, fmt.Errorf("brew_package step: 'package' field is required")
	}
	return &brewPackageStep{name: name, pkg: pkg}, nil
}

func (s *brewPackageStep) Type() string { return "brew_package" }
func (s *brewPackageStep) Name() string { return s.name }

func (s *brewPackageStep) Check(ctx context.Context, env Env) (bool, error) {
	err := env.Exec.Run(ctx, "brew", []string{"list", "--formula", s.pkg}, nil, nopLineSink{})
	if err == nil {
		return true, nil
	}
	var xe *xexec.Error
	if errors.As(err, &xe) {
		return false, nil
	}
	return false, fmt.Errorf("brew_package %q check: %w", s.name, err)
}

func (s *brewPackageStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "brew", []string{"install", s.pkg}, nil, ls)
}

// RegisterBrewPackage is the idempotent registration helper (mirrors
// RegisterShell). Tests that wipe the registry call this to restore state.
func RegisterBrewPackage() {
	if !isRegistered("brew_package") {
		Register("brew_package", newBrewPackage)
	}
}

func init() { RegisterBrewPackage() }
