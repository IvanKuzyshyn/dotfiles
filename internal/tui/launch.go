package tui

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ivankuzyshyn/dotfiles/internal/cli"
	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/platform"
	"github.com/ivankuzyshyn/dotfiles/internal/runner"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// RunCompletedMsg is sent by the runner goroutine when runner.Run returns. The
// launchApp uses it to switch the screen into the summary state. It is
// defined here rather than in app.go because only the launcher cares about
// run lifecycle messages.
type RunCompletedMsg struct {
	Result runner.Result
}

// Launch builds the dotfiles registry, constructs the Bubble Tea program, and
// blocks until the user quits. It is invoked from cmd/dot/main.go when the
// user runs `dot` with no subcommand on a TTY. Returns a non-nil error when
// any tool in the run failed so main can set a non-zero exit code.
func Launch(g *cli.GlobalFlags) error {
	// 1. Load manifests via the same path the CLI uses so the TUI sees the
	// same set of tools (embedded + overlays).
	manifests, err := cli.LoadAllManifests(g)
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		return errors.New("no tools available")
	}

	// 2. Validate against the registered step type set.
	if err := manifest.Validate(manifests, step.RegisteredTypes()); err != nil {
		return err
	}

	// 3. Build registry.
	reg, err := tool.NewRegistry(manifests)
	if err != nil {
		return err
	}

	// 4. Build env. Failure to resolve a dotfiles dir is non-fatal for the
	// TUI: the user may pick tools with no configs. Linker apply will fail
	// loudly later if the path is required but missing.
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	dotfilesDir, _ := (cli.Resolver{FS: xfs.Real{}, Home: home, Cwd: cwd}).Resolve(g.DotfilesDir)
	env := step.Env{
		Exec:        xexec.Real{},
		FS:          xfs.Real{},
		Platform:    platform.Detect(),
		HomeDir:     home,
		DotfilesDir: dotfilesDir,
	}

	// 5. Open log-file sink so a run-log is persisted regardless of TUI
	// rendering. Failure to open is non-fatal (matches CLI install).
	var logSink *event.LogFileSink
	if l, lerr := event.NewLogFileSink(cli.StateDir(home), 10); lerr == nil {
		logSink = l
		defer logSink.Close()
	}

	// 6. Build sink + program. The sink references the program for Send
	// forwarding; the program holds the launchApp model, which holds the
	// sink. We break the cycle by constructing the sink without a program
	// reference and assigning it after tea.NewProgram returns.
	sink := &Sink{}
	la := launchApp{
		inner:   NewApp(reg, env),
		env:     env,
		sink:    sink,
		logSink: logSink,
	}
	prog := tea.NewProgram(la, tea.WithAltScreen())
	sink.prog = prog

	// 7. Run. Errors here are program errors (terminal IO etc.), not run
	// failures, and bubble up directly.
	finalModel, err := prog.Run()
	if err != nil {
		return err
	}

	// 8. Translate run result to an exit-code-bearing error so the caller
	// can match the CLI's `dot install` exit behavior.
	if final, ok := finalModel.(launchApp); ok && final.result.AnyFailed() {
		return fmt.Errorf("one or more tools failed")
	}
	return nil
}

// launchApp wraps the inner App and intercepts run-control messages so the
// inner App stays unaware of runner/sink/goroutine wiring.
type launchApp struct {
	inner   App
	env     step.Env
	sink    *Sink
	logSink *event.LogFileSink
	result  runner.Result
}

// Init forwards to the inner App.
func (l launchApp) Init() tea.Cmd { return l.inner.Init() }

// Update intercepts messages relevant to the runner goroutine before letting
// the inner App handle the rest. The interception keeps app.go free of any
// reference to runner.Run, sinks, or the program handle.
func (l launchApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case StartRunMsg:
		l.inner.runner = NewRunnerPane(m.Tools)
		l.inner.screen = screenRunner
		return l, l.startRun(m.Tools)

	case RunEventMsg:
		// Open the conflict modal when the runner asks for a decision.
		if m.Event.Kind == event.ConflictPrompt && m.Event.Conflict != nil {
			modal := NewConflictModal(*m.Event.Conflict)
			l.inner.modal = &modal
		}
		// Forward to inner App so the runner pane consumes the event too.
		inner, cmd := l.inner.Update(msg)
		l.inner = inner.(App)
		return l, cmd

	case ConflictResolutionMsg:
		// Forward the user's choice into the resolver channel for whichever
		// target the modal was showing, then dismiss the modal.
		if l.inner.modal != nil {
			l.sink.Resolve(l.inner.modal.target, m.Action)
			l.inner.modal = nil
		}
		return l, nil

	case RunCompletedMsg:
		l.result = m.Result
		l.inner.screen = screenSummary
		return l, nil

	case RetryToolMsg:
		if m.Tool == nil {
			return l, nil
		}
		l.inner.runner = NewRunnerPane([]*tool.Tool{m.Tool})
		l.inner.screen = screenRunner
		return l, l.startRun([]*tool.Tool{m.Tool})
	}

	inner, cmd := l.inner.Update(msg)
	l.inner = inner.(App)
	return l, cmd
}

// View delegates to the inner App.
func (l launchApp) View() string { return l.inner.View() }

// startRun returns a tea.Cmd that, when scheduled by the Tea runtime in its
// goroutine pool, runs the plan to completion and yields RunCompletedMsg. The
// sink forwards individual events back to the program while the run is in
// flight, so the runner pane updates live.
func (l launchApp) startRun(tools []*tool.Tool) tea.Cmd {
	return func() tea.Msg {
		plan := runner.Plan{Tools: tool.Sort(tools)}
		var sink event.Sink = l.sink
		if l.logSink != nil {
			sink = event.Tee(l.sink, l.logSink)
		}
		result := runner.Run(context.Background(), plan, l.env, sink)
		return RunCompletedMsg{Result: result}
	}
}
