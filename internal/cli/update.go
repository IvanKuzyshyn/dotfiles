package cli

import "github.com/spf13/cobra"

// NewUpdateCmd is an alias of install: same flags, same behavior.
// The reconcile model makes update equivalent to install — steps are
// idempotent, so re-running install on already-installed tools refreshes
// them (e.g., git pull, brew upgrade) without duplicate work.
func NewUpdateCmd(g *GlobalFlags) *cobra.Command {
	f := &installFlags{}
	cmd := &cobra.Command{
		Use:   "update [tool ...]",
		Short: "Reconcile selected tools (alias of install)",
		Long:  "update re-runs install steps for selected tools. Because steps are idempotent (check-then-run, git pull, brew upgrade), updating already-installed tools refreshes them without duplicate work.",
		RunE: func(c *cobra.Command, args []string) error {
			return runInstall(c.Context(), c.ErrOrStderr(), g, args, f)
		},
	}
	cmd.Flags().BoolVar(&f.all, "all", false, "update every known tool")
	cmd.Flags().StringVar(&f.tag, "tag", "", "update tools matching this tag")
	cmd.Flags().BoolVar(&f.noDeps, "no-deps", false, "do not expand transitive dependencies")
	cmd.Flags().StringVar(&f.onConflict, "on-conflict", "abort", "what to do on a conflict during config deploy: backup|overwrite|skip|abort")
	return cmd
}
