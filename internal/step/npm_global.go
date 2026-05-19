package step

import (
	"context"
	"errors"
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

// npmGlobalStep installs an npm package globally. The package field is required
// and specifies the npm package name.
type npmGlobalStep struct {
	name string
	pkg  string
}

func newNpmGlobal(name string, fields map[string]any) (Step, error) {
	if name == "" {
		name = "npm_global"
	}
	pkg, _ := stringField(fields, "package")
	if pkg == "" {
		return nil, fmt.Errorf("npm_global step: 'package' field is required")
	}
	return &npmGlobalStep{name: name, pkg: pkg}, nil
}

func (s *npmGlobalStep) Type() string { return "npm_global" }
func (s *npmGlobalStep) Name() string { return s.name }

func (s *npmGlobalStep) Check(ctx context.Context, env Env) (bool, error) {
	err := env.Exec.Run(ctx, "npm", []string{"list", "-g", "--depth=0", s.pkg}, nil, nopLineSink{})
	if err == nil {
		return true, nil
	}
	var xe *xexec.Error
	if errors.As(err, &xe) {
		return false, nil
	}
	return false, fmt.Errorf("npm_global %q check: %w", s.name, err)
}

func (s *npmGlobalStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "npm", []string{"install", "-g", s.pkg}, nil, ls)
}

// RegisterNpmGlobal is the idempotent registration helper (mirrors
// RegisterBrewPackage). Tests that wipe the registry call this to restore state.
func RegisterNpmGlobal() {
	if !isRegistered("npm_global") {
		Register("npm_global", newNpmGlobal)
	}
}

func init() { RegisterNpmGlobal() }
