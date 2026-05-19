package step

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
)

type gitCloneStep struct {
	name string
	url  string
	dest string
}

func newGitClone(name string, fields map[string]any) (Step, error) {
	url, _ := stringField(fields, "url")
	dest, _ := stringField(fields, "dest")
	if url == "" {
		return nil, fmt.Errorf("git_clone step %q: 'url' field is required", name)
	}
	if dest == "" {
		return nil, fmt.Errorf("git_clone step %q: 'dest' field is required", name)
	}
	return &gitCloneStep{name: name, url: url, dest: dest}, nil
}

func (s *gitCloneStep) Type() string { return "git_clone" }
func (s *gitCloneStep) Name() string { return s.name }

func (s *gitCloneStep) resolveDest(env Env) string {
	d := s.dest
	if strings.HasPrefix(d, "~/") && env.HomeDir != "" {
		d = filepath.Join(env.HomeDir, d[2:])
	} else if d == "~" && env.HomeDir != "" {
		d = env.HomeDir
	}
	return d
}

func (s *gitCloneStep) Check(_ context.Context, env Env) (bool, error) {
	if env.FS == nil {
		return false, nil
	}
	d := s.resolveDest(env)
	info, err := env.FS.Stat(d)
	if err != nil {
		return false, nil // missing is the common case; not a hard error
	}
	return info.IsDir(), nil
}

func (s *gitCloneStep) Run(ctx context.Context, env Env, sink event.Sink) error {
	d := s.resolveDest(env)
	ls := lineSinkBridge{sink: sink, tool: ctxTool(ctx), step: s.name}

	exists := false
	if env.FS != nil {
		if info, err := env.FS.Stat(d); err == nil && info.IsDir() {
			exists = true
		}
	}
	if exists {
		return env.Exec.Run(ctx, "git", []string{"-C", d, "pull", "--ff-only"}, nil, ls)
	}
	return env.Exec.Run(ctx, "git", []string{"clone", s.url, d}, nil, ls)
}

func RegisterGitClone() {
	if !isRegistered("git_clone") {
		Register("git_clone", newGitClone)
	}
}

func init() { RegisterGitClone() }
