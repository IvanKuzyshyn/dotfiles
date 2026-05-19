package step

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

// brewfileStep installs packages using a Brewfile. The path field is required
// and specifies the relative path to the Brewfile.
type brewfileStep struct {
	name string
	path string
}

func newBrewfile(name string, fields map[string]any) (Step, error) {
	p, _ := stringField(fields, "path")
	if p == "" {
		return nil, fmt.Errorf("brewfile step %q: 'path' field is required", name)
	}
	return &brewfileStep{name: name, path: p}, nil
}

func (s *brewfileStep) Type() string { return "brewfile" }
func (s *brewfileStep) Name() string { return s.name }

func (s *brewfileStep) resolvedPath(env Env) (string, error) {
	if env.DotfilesDir == "" {
		return "", fmt.Errorf("brewfile step %q: Env.DotfilesDir is empty", s.name)
	}
	if filepath.IsAbs(s.path) {
		return s.path, nil
	}
	return filepath.Join(env.DotfilesDir, s.path), nil
}

func (s *brewfileStep) Check(ctx context.Context, env Env) (bool, error) {
	abs, err := s.resolvedPath(env)
	if err != nil {
		return false, err
	}
	err = env.Exec.Run(ctx, "brew", []string{"bundle", "check", "--file=" + abs}, nil, nopLineSink{})
	if err == nil {
		return true, nil
	}
	var xe *xexec.Error
	if errors.As(err, &xe) {
		return false, nil
	}
	return false, fmt.Errorf("brewfile step %q check: %w", s.name, err)
}

func (s *brewfileStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	abs, err := s.resolvedPath(env)
	if err != nil {
		return err
	}
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "brew", []string{"bundle", "--file=" + abs}, nil, ls)
}

// RegisterBrewfile is the idempotent registration helper (mirrors
// RegisterBrewPackage). Tests that wipe the registry call this to restore state.
func RegisterBrewfile() {
	if !isRegistered("brewfile") {
		Register("brewfile", newBrewfile)
	}
}

func init() { RegisterBrewfile() }
