package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/platform"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
	"github.com/spf13/cobra"
)

type statusFlags struct {
	all bool
	tag string
}

// NewStatusCmd returns the `dot status` subcommand. Read-only: runs every
// step's Check() and classifies each tool as installed/partial/missing.
func NewStatusCmd(g *GlobalFlags) *cobra.Command {
	f := &statusFlags{}
	cmd := &cobra.Command{
		Use:   "status [tool ...]",
		Short: "Report installation state of selected tools (read-only)",
		RunE: func(c *cobra.Command, args []string) error {
			return runStatus(c.Context(), c.OutOrStdout(), c.ErrOrStderr(), g, args, f)
		},
	}
	cmd.Flags().BoolVar(&f.all, "all", true, "report on all known tools (default)")
	cmd.Flags().StringVar(&f.tag, "tag", "", "report on tools matching this tag")
	return cmd
}

func runStatus(ctx context.Context, out, errw io.Writer, g *GlobalFlags, args []string, f *statusFlags) error {
	manifests, err := loadAllManifests(g)
	if err != nil {
		return wrapPreflight(err)
	}
	if len(manifests) == 0 {
		fmt.Fprintln(errw, "no tools available")
		return nil
	}
	if err := manifest.Validate(manifests, step.RegisteredTypes()); err != nil {
		return wrapPreflight(err)
	}
	reg, err := tool.NewRegistry(manifests)
	if err != nil {
		return wrapPreflight(err)
	}
	// When positional args are given, --all defaults to false.
	all := f.all
	if len(args) > 0 || f.tag != "" {
		all = false
	}
	selected, err := tool.Select(reg, args, all, f.tag)
	if err != nil {
		return wrapPreflight(err)
	}
	if len(selected) == 0 {
		fmt.Fprintln(errw, "nothing matched")
		return nil
	}

	home, _ := os.UserHomeDir()
	dotfilesDir, _ := (Resolver{FS: xfs.Real{}, Home: home, Cwd: getCwd()}).Resolve(g.DotfilesDir)
	// status is best-effort; tolerate missing DotfilesDir for tools without configs
	env := step.Env{
		Exec:        xexec.Real{},
		FS:          xfs.Real{},
		Platform:    platform.Detect(),
		HomeDir:     home,
		DotfilesDir: dotfilesDir,
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATE\tDETAIL")
	for _, t := range selected {
		satisfied, total, anyErr := evaluateTool(ctx, t, env)
		state := classify(satisfied, total)
		detail := fmt.Sprintf("%d/%d steps satisfied", satisfied, total)
		if anyErr {
			detail += " (one or more check errors)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", t.Name, state, detail)
	}
	return tw.Flush()
}

// evaluateTool runs every step's Check and returns (satisfied, total, anyError).
func evaluateTool(ctx context.Context, t *tool.Tool, env step.Env) (int, int, bool) {
	satisfied := 0
	anyErr := false
	for _, s := range t.Steps {
		ok, err := safeCheck(ctx, s, env)
		if err != nil {
			anyErr = true
			continue
		}
		if ok {
			satisfied++
		}
	}
	return satisfied, len(t.Steps), anyErr
}

func safeCheck(ctx context.Context, s step.Step, env step.Env) (ok bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("step %q check panicked: %v", s.Name(), r)
		}
	}()
	return s.Check(ctx, env)
}

func classify(satisfied, total int) string {
	if total == 0 {
		return "n/a"
	}
	if satisfied == total {
		return "installed"
	}
	if satisfied == 0 {
		return "missing"
	}
	return "partial"
}
