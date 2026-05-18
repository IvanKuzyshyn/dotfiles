package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/runner"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

type fakeStep struct {
	name       string
	check      bool
	checkErr   error
	runErr     error
	runPanic   bool
	checkPanic bool
}

func (f *fakeStep) Type() string { return "fake" }
func (f *fakeStep) Name() string { return f.name }
func (f *fakeStep) Check(_ context.Context, _ step.Env) (bool, error) {
	if f.checkPanic {
		panic("boom-check")
	}
	return f.check, f.checkErr
}
func (f *fakeStep) Run(_ context.Context, _ step.Env, _ event.Sink) error {
	if f.runPanic {
		panic("boom-run")
	}
	return f.runErr
}

type collectSink struct{ events []event.Event }

func (c *collectSink) Send(e event.Event) { c.events = append(c.events, e) }

// autoResolveSink auto-resolves ConflictPrompt events with the configured action.
type autoResolveSink struct {
	events []event.Event
	action event.ConflictAction
}

func (s *autoResolveSink) Send(e event.Event) {
	s.events = append(s.events, e)
	if e.Kind == event.ConflictPrompt && e.Conflict != nil && e.Conflict.Resolver != nil {
		e.Conflict.Resolver <- s.action
	}
}

func mkTool(name string, deps []string, steps ...step.Step) *tool.Tool {
	return &tool.Tool{Name: name, DependsOn: deps, Steps: steps, Configs: []manifest.Config{}}
}

func TestRun_AllSucceed(t *testing.T) {
	t1 := mkTool("a", nil, &fakeStep{name: "s1"})
	t2 := mkTool("b", nil, &fakeStep{name: "s2"})
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1, t2}}, step.Env{}, &collectSink{})
	s, sk, f := r.Counts()
	if s != 2 || sk != 0 || f != 0 {
		t.Errorf("counts = %d/%d/%d", s, sk, f)
	}
	if r.AnyFailed() {
		t.Error("AnyFailed should be false")
	}
}

func TestRun_OneFailsOthersContinue(t *testing.T) {
	good := mkTool("good", nil, &fakeStep{name: "s"})
	bad := mkTool("bad", nil, &fakeStep{name: "s", runErr: errors.New("boom")})
	other := mkTool("other", nil, &fakeStep{name: "s"})
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{good, bad, other}}, step.Env{}, &collectSink{})
	s, sk, f := r.Counts()
	if s != 2 || sk != 0 || f != 1 {
		t.Errorf("counts = %d/%d/%d", s, sk, f)
	}
}

func TestRun_DependencyFailureSkipsDependents(t *testing.T) {
	bad := mkTool("a", nil, &fakeStep{name: "s", runErr: errors.New("boom")})
	dependent := mkTool("b", []string{"a"}, &fakeStep{name: "s"})
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{bad, dependent}}, step.Env{}, &collectSink{})
	s, sk, f := r.Counts()
	if s != 0 || sk != 1 || f != 1 {
		t.Errorf("counts = %d/%d/%d", s, sk, f)
	}
	if r.Tools[1].State != runner.Skipped {
		t.Errorf("dependent should be Skipped, got %v", r.Tools[1].State)
	}
}

func TestRun_CheckSatisfiedSkipsStep(t *testing.T) {
	t1 := mkTool("a", nil, &fakeStep{name: "s", check: true})
	sink := &collectSink{}
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, step.Env{}, sink)
	if r.Tools[0].State != runner.Succeeded {
		t.Errorf("want succeeded, got %v", r.Tools[0].State)
	}
	var sawSkipped bool
	for _, e := range sink.events {
		if e.Kind == event.StepSkipped {
			sawSkipped = true
		}
	}
	if !sawSkipped {
		t.Error("expected StepSkipped event")
	}
}

func TestRun_PanicInRunBecomesFailure(t *testing.T) {
	t1 := mkTool("a", nil, &fakeStep{name: "s", runPanic: true})
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, step.Env{}, &collectSink{})
	if r.Tools[0].State != runner.Failed {
		t.Errorf("want failed, got %v", r.Tools[0].State)
	}
	if r.Tools[0].Err == nil {
		t.Error("expected non-nil error")
	}
}

func TestRun_PanicInCheckBecomesFailure(t *testing.T) {
	t1 := mkTool("a", nil, &fakeStep{name: "s", checkPanic: true})
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, step.Env{}, &collectSink{})
	if r.Tools[0].State != runner.Failed {
		t.Errorf("want failed, got %v", r.Tools[0].State)
	}
}

func TestRun_ContextCancellationSkipsRemaining(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	t1 := mkTool("a", nil, &fakeStep{name: "s"})
	t2 := mkTool("b", nil, &fakeStep{name: "s"})
	r := runner.Run(ctx, runner.Plan{Tools: []*tool.Tool{t1, t2}}, step.Env{}, &collectSink{})
	s, sk, f := r.Counts()
	if s != 0 || sk != 2 || f != 0 {
		t.Errorf("counts = %d/%d/%d", s, sk, f)
	}
}

func TestRun_EmitsExpectedEvents(t *testing.T) {
	t1 := mkTool("a", nil, &fakeStep{name: "s1"})
	sink := &collectSink{}
	runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, step.Env{}, sink)
	var kinds []event.Kind
	for _, e := range sink.events {
		kinds = append(kinds, e.Kind)
	}
	want := []event.Kind{
		event.ToolStarted,
		event.StepStarted,
		event.StepFinished,
		event.ToolFinished,
	}
	if len(kinds) != len(want) {
		t.Fatalf("want %v, got %v", want, kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("at %d: want %v, got %v", i, want[i], kinds[i])
		}
	}
}

func TestRun_DeploysConfigs_NoConflict(t *testing.T) {
	fs := xfs.NewFake()
	fs.MkdirAll("/dotfiles/configs/git", 0o755)
	fs.CreateFile("/dotfiles/configs/git/.gitconfig", 0o644)
	env := step.Env{
		Exec:        xexec.NewFake(),
		FS:          fs,
		DotfilesDir: "/dotfiles",
		HomeDir:     "/home/me",
	}
	t1 := &tool.Tool{
		Name:    "git",
		Steps:   []step.Step{&fakeStep{name: "noop"}},
		Configs: []manifest.Config{{Source: "git", Target: "/home/me"}},
	}
	sink := &autoResolveSink{action: event.ConflictBackup}
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, env, sink)
	if r.Tools[0].State != runner.Succeeded {
		t.Errorf("want succeeded, got %v err=%v", r.Tools[0].State, r.Tools[0].Err)
	}
	if _, err := fs.Readlink("/home/me/.gitconfig"); err != nil {
		t.Errorf("expected symlink at /home/me/.gitconfig: %v", err)
	}
}

func TestRun_DeploysConfigs_BackupOnConflict(t *testing.T) {
	fs := xfs.NewFake()
	fs.MkdirAll("/dotfiles/configs/git", 0o755)
	fs.CreateFile("/dotfiles/configs/git/.gitconfig", 0o644)
	fs.MkdirAll("/home/me", 0o755)
	fs.CreateFile("/home/me/.gitconfig", 0o644) // existing file, will conflict
	env := step.Env{
		Exec:        xexec.NewFake(),
		FS:          fs,
		DotfilesDir: "/dotfiles",
		HomeDir:     "/home/me",
	}
	t1 := &tool.Tool{
		Name:    "git",
		Steps:   []step.Step{&fakeStep{name: "noop"}},
		Configs: []manifest.Config{{Source: "git", Target: "/home/me"}},
	}
	sink := &autoResolveSink{action: event.ConflictBackup}
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, env, sink)
	if r.Tools[0].State != runner.Succeeded {
		t.Errorf("want succeeded, got %v err=%v", r.Tools[0].State, r.Tools[0].Err)
	}
	if _, err := fs.Readlink("/home/me/.gitconfig"); err != nil {
		t.Errorf("symlink missing: %v", err)
	}
	sawPrompt := false
	for _, e := range sink.events {
		if e.Kind == event.ConflictPrompt {
			sawPrompt = true
		}
	}
	if !sawPrompt {
		t.Error("expected ConflictPrompt event")
	}
}

func TestRun_DeploysConfigs_AbortFailsTool(t *testing.T) {
	fs := xfs.NewFake()
	fs.MkdirAll("/dotfiles/configs/git", 0o755)
	fs.CreateFile("/dotfiles/configs/git/.gitconfig", 0o644)
	fs.MkdirAll("/home/me", 0o755)
	fs.CreateFile("/home/me/.gitconfig", 0o644)
	env := step.Env{
		Exec:        xexec.NewFake(),
		FS:          fs,
		DotfilesDir: "/dotfiles",
		HomeDir:     "/home/me",
	}
	t1 := &tool.Tool{
		Name:    "git",
		Steps:   []step.Step{&fakeStep{name: "noop"}},
		Configs: []manifest.Config{{Source: "git", Target: "/home/me"}},
	}
	sink := &autoResolveSink{action: event.ConflictAbort}
	r := runner.Run(context.Background(), runner.Plan{Tools: []*tool.Tool{t1}}, env, sink)
	if r.Tools[0].State != runner.Failed {
		t.Errorf("want failed (abort), got %v", r.Tools[0].State)
	}
}
