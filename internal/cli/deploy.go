package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ivankuzyshyn/dotfiles/internal/event"
	xexec "github.com/ivankuzyshyn/dotfiles/internal/exec"
	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/linker"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/platform"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
	"github.com/spf13/cobra"
)

type deployFlags struct {
	all        bool
	tag        string
	noDeps     bool
	onConflict string
}

// NewDeployCmd returns the `dot deploy` subcommand: symlinks only, no
// install steps. Useful for re-deploying configs without running installs.
func NewDeployCmd(g *GlobalFlags) *cobra.Command {
	f := &deployFlags{}
	cmd := &cobra.Command{
		Use:   "deploy [tool ...]",
		Short: "Deploy configuration files via symlinks (skip install steps)",
		RunE: func(c *cobra.Command, args []string) error {
			return runDeploy(c.Context(), c.ErrOrStderr(), g, args, f)
		},
	}
	cmd.Flags().BoolVar(&f.all, "all", false, "deploy every known tool's configs")
	cmd.Flags().StringVar(&f.tag, "tag", "", "deploy tools matching this tag")
	cmd.Flags().BoolVar(&f.noDeps, "no-deps", false, "do not expand transitive dependencies")
	cmd.Flags().StringVar(&f.onConflict, "on-conflict", "abort",
		"what to do on a conflict: backup|overwrite|skip|abort")
	return cmd
}

func runDeploy(ctx context.Context, errw io.Writer, g *GlobalFlags, args []string, f *deployFlags) error {
	if !f.all && f.tag == "" && len(args) == 0 {
		return wrapPreflight(errors.New("specify one or more tools, --all, or --tag"))
	}
	action, err := ParseConflictAction(f.onConflict)
	if err != nil {
		return wrapPreflight(err)
	}

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
	selected, err := tool.Select(reg, args, f.all, f.tag)
	if err != nil {
		return wrapPreflight(err)
	}
	if len(selected) == 0 {
		fmt.Fprintln(errw, "nothing matched")
		return nil
	}
	expanded, err := tool.ExpandDeps(reg, selected, f.noDeps)
	if err != nil {
		return wrapPreflight(err)
	}

	home, _ := os.UserHomeDir()
	dotfilesDir, err := (Resolver{FS: xfs.Real{}, Home: home, Cwd: getCwd()}).Resolve(g.DotfilesDir)
	if err != nil {
		return wrapPreflight(err)
	}
	env := step.Env{
		Exec:        xexec.Real{},
		FS:          xfs.Real{},
		Platform:    platform.Detect(),
		HomeDir:     home,
		DotfilesDir: dotfilesDir,
	}

	streamSink := event.StreamSink{W: os.Stderr}
	sink := event.Sink(streamSink)
	sink = FlagResolverSink{Inner: sink, Action: action}

	backupRoot := filepath.Join(home, ".dotfiles_backup_"+time.Now().UTC().Format("20060102T150405Z"))

	var succeeded, failed int
	for _, t := range expanded {
		if len(t.Configs) == 0 {
			continue
		}
		sink.Send(event.Event{Kind: event.ToolStarted, Tool: t.Name})
		if err := deployTool(ctx, t, env, sink, backupRoot); err != nil {
			sink.Send(event.Event{Kind: event.ToolFailed, Tool: t.Name, Err: err})
			failed++
			continue
		}
		sink.Send(event.Event{Kind: event.ToolFinished, Tool: t.Name})
		succeeded++
	}

	fmt.Fprintf(errw, "\nsucceeded=%d failed=%d\n", succeeded, failed)
	if failed > 0 {
		return preflightErr{msg: "one or more deploys failed", code: 1}
	}
	return nil
}

// deployTool runs Inspect + Apply for one tool's configs. Mirrors the
// linker block of runner.runLinker but without runner-specific state.
func deployTool(ctx context.Context, t *tool.Tool, env step.Env, sink event.Sink, backupRoot string) error {
	plan, err := linker.Inspect(t.Configs, env)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}
	for _, d := range plan.Decisions {
		var c *linker.Conflict
		var action event.ConflictAction
		if d.Kind == linker.DecideConflict {
			for j := range plan.Conflicts {
				if plan.Conflicts[j].Target == d.Target {
					c = &plan.Conflicts[j]
					break
				}
			}
			if c == nil {
				return fmt.Errorf("internal: conflict decision without matching Conflict at %s", d.Target)
			}
			resolverCh := make(chan event.ConflictAction, 1)
			sink.Send(event.Event{
				Kind: event.ConflictPrompt,
				Tool: t.Name,
				Conflict: &event.Conflict{
					TargetPath:   c.Target,
					ExistingKind: string(c.ExistingKind),
					Resolver:     resolverCh,
				},
			})
			select {
			case action = <-resolverCh:
			case <-ctx.Done():
				return ctx.Err()
			}
			sink.Send(event.Event{Kind: event.ConflictResolved, Tool: t.Name})
		}
		if err := linker.Apply(d, c, action, backupRoot, env.FS); err != nil {
			return fmt.Errorf("apply %s: %w", d.Target, err)
		}
	}
	return nil
}
