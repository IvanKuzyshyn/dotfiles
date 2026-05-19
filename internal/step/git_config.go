package step

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
)

const (
	scopeRepo   = "repo"
	scopeGlobal = "global"
)

type gitConfigStep struct {
	name  string
	key   string
	value string
	scope string
}

func newGitConfig(name string, fields map[string]any) (Step, error) {
	key, _ := stringField(fields, "key")
	value, _ := stringField(fields, "value")
	scope, _ := stringField(fields, "scope")
	if scope == "" {
		scope = scopeRepo
	}
	if key == "" {
		return nil, fmt.Errorf("git_config step %q: 'key' field is required", name)
	}
	if value == "" {
		return nil, fmt.Errorf("git_config step %q: 'value' field is required", name)
	}
	if scope != scopeRepo && scope != scopeGlobal {
		return nil, fmt.Errorf("git_config step %q: invalid scope %q (want %q or %q)",
			name, scope, scopeRepo, scopeGlobal)
	}
	return &gitConfigStep{name: name, key: key, value: value, scope: scope}, nil
}

func (s *gitConfigStep) Type() string { return "git_config" }
func (s *gitConfigStep) Name() string { return s.name }

func (s *gitConfigStep) checkArgs(env Env) ([]string, error) {
	switch s.scope {
	case scopeGlobal:
		return []string{"config", "--global", "--get", s.key}, nil
	default: // repo
		if env.DotfilesDir == "" {
			return nil, fmt.Errorf("git_config step %q: repo scope requires Env.DotfilesDir", s.name)
		}
		return []string{"-C", env.DotfilesDir, "config", "--get", s.key}, nil
	}
}

func (s *gitConfigStep) runArgs(env Env) ([]string, error) {
	switch s.scope {
	case scopeGlobal:
		return []string{"config", "--global", s.key, s.value}, nil
	default: // repo
		if env.DotfilesDir == "" {
			return nil, fmt.Errorf("git_config step %q: repo scope requires Env.DotfilesDir", s.name)
		}
		return []string{"-C", env.DotfilesDir, "config", s.key, s.value}, nil
	}
}

func (s *gitConfigStep) Check(ctx context.Context, env Env) (bool, error) {
	args, err := s.checkArgs(env)
	if err != nil {
		return false, err
	}
	cap := &captureLines{}
	err = env.Exec.Run(ctx, "git", args, nil, cap)
	if err != nil {
		var xe *xexec.Error
		if errors.As(err, &xe) {
			return false, nil
		}
		return false, fmt.Errorf("git_config step %q check: %w", s.name, err)
	}
	if len(cap.lines) == 0 {
		return false, nil
	}
	got := strings.TrimSpace(cap.lines[0])
	return got == s.value, nil
}

func (s *gitConfigStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	args, err := s.runArgs(env)
	if err != nil {
		return err
	}
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}
	return env.Exec.Run(ctx, "git", args, nil, ls)
}

type captureLines struct{ lines []string }

func (c *captureLines) Line(s string) { c.lines = append(c.lines, s) }

func RegisterGitConfig() {
	if !isRegistered("git_config") {
		Register("git_config", newGitConfig)
	}
}

func init() { RegisterGitConfig() }
