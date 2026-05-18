package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCmd returns the `dot version` subcommand. It prints the version
// string injected at build time via -ldflags.
func NewVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the dot version",
		RunE: func(c *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(c.OutOrStdout(), version)
			return err
		},
	}
}
