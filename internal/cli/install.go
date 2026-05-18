package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/platform"
	"github.com/ivankuzyshyn/dotfiles/internal/runner"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
	"github.com/spf13/cobra"
)

type installFlags struct {
	all    bool
	tag    string
	noDeps bool
}

// NewInstallCmd returns the `dot install` subcommand.
func NewInstallCmd(g *GlobalFlags) *cobra.Command {
	f := &installFlags{}
	cmd := &cobra.Command{
		Use:   "install [tool ...]",
		Short: "Install one or more tools",
		Long:  "install resolves the selection, expands dependencies, and runs each tool's steps. Failures are isolated per tool; dependents of a failed tool are skipped.",
		RunE: func(c *cobra.Command, args []string) error {
			return runInstall(c.Context(), os.Stderr, g, args, f)
		},
	}
	cmd.Flags().BoolVar(&f.all, "all", false, "install every known tool")
	cmd.Flags().StringVar(&f.tag, "tag", "", "install tools matching this tag")
	cmd.Flags().BoolVar(&f.noDeps, "no-deps", false, "do not expand transitive dependencies; fail if any required dep is missing")
	return cmd
}

// runInstall is the install command body, factored out so unit tests can call it.
func runInstall(ctx context.Context, errw io.Writer, g *GlobalFlags, args []string, f *installFlags) error {
	// Pre-flight: require a selection before doing any IO.
	if !f.all && f.tag == "" && len(args) == 0 {
		return wrapPreflight(errors.New("specify one or more tools, --all, or --tag"))
	}

	// 1. Load manifests
	manifests, err := loadAllManifests(g)
	if err != nil {
		return wrapPreflight(err)
	}
	if len(manifests) == 0 {
		fmt.Fprintln(errw, "no tools available")
		return nil
	}

	// 2. Validate
	if err := manifest.Validate(manifests, step.RegisteredTypes()); err != nil {
		return wrapPreflight(err)
	}

	// 3. Build registry
	reg, err := tool.NewRegistry(manifests)
	if err != nil {
		return wrapPreflight(err)
	}

	// 4. Resolve selection
	selected, err := tool.Select(reg, args, f.all, f.tag)
	if err != nil {
		return wrapPreflight(err)
	}
	if len(selected) == 0 {
		fmt.Fprintln(errw, "nothing matched")
		return nil
	}

	// 5. Expand deps
	expanded, err := tool.ExpandDeps(reg, selected, f.noDeps)
	if err != nil {
		return wrapPreflight(err)
	}

	// 6. Sort
	plan := runner.Plan{Tools: tool.Sort(expanded)}

	// 7. Build Env
	home, _ := os.UserHomeDir()
	dotfilesDir, resolveErr := (Resolver{FS: xfs.Real{}, Home: home, Cwd: getCwd()}).Resolve(g.DotfilesDir)
	if resolveErr != nil && needsDotfilesDir(plan) {
		return wrapPreflight(fmt.Errorf("dotfiles directory required but not found: %w", resolveErr))
	}
	env := step.Env{
		Exec:        xexec.Real{},
		FS:          xfs.Real{},
		Platform:    platform.Detect(),
		HomeDir:     home,
		DotfilesDir: dotfilesDir,
	}

	// 8. Build sink
	streamSink := event.StreamSink{W: os.Stderr}
	logSink, logErr := event.NewLogFileSink(stateDir(home), 10)
	var sink event.Sink
	if logErr == nil {
		defer logSink.Close()
		sink = event.Tee(streamSink, logSink)
	} else {
		sink = streamSink
		fmt.Fprintf(os.Stderr, "warning: could not open log file: %v\n", logErr)
	}

	// 9. Run
	result := runner.Run(ctx, plan, env, sink)

	// 10. Summary
	s, sk, failed := result.Counts()
	fmt.Fprintf(os.Stderr, "\nsucceeded=%d skipped=%d failed=%d\n", s, sk, failed)
	if logErr == nil {
		fmt.Fprintf(os.Stderr, "log: %s\n", logSink.Path())
	}
	if result.AnyFailed() {
		return preflightErr{msg: "one or more tools failed", code: 1}
	}
	return nil
}

func wrapPreflight(err error) error { return preflightErr{msg: err.Error(), code: 2} }

// preflightErr is the exit-code-carrying error returned by runInstall on
// pre-flight or runtime failure. main() inspects it to set the exit code.
type preflightErr struct {
	msg  string
	code int
}

func (e preflightErr) Error() string { return e.msg }

// ExitCode extracts the exit code from a preflightErr, defaulting to 1.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var pe preflightErr
	if errors.As(err, &pe) {
		return pe.code
	}
	return 1
}

func getCwd() string {
	d, _ := os.Getwd()
	return d
}

func stateDir(home string) string {
	if home == "" {
		return ".dot-state"
	}
	return filepath.Join(home, ".local", "state", "dot")
}

// needsDotfilesDir returns true if any tool in the plan declares configs
// to be deployed (Phase 2). Phase 1 has no configs so we don't strictly
// need DotfilesDir, but if Resolver fails AND any tool has configs, we
// must fail pre-flight.
func needsDotfilesDir(plan runner.Plan) bool {
	for _, t := range plan.Tools {
		if len(t.Configs) > 0 {
			return true
		}
	}
	return false
}
