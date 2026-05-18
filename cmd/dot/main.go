// Command dot installs tools and deploys configuration files from a
// dotfiles repository. See `dot --help` for usage.
package main

import (
	"fmt"
	"os"

	"github.com/ivankuzyshyn/dotfiles/internal/cli"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	g := &cli.GlobalFlags{}
	root := cli.NewRoot(g)
	root.AddCommand(cli.NewListCmd(g))
	root.AddCommand(cli.NewInstallCmd(g))
	root.AddCommand(cli.NewUpdateCmd(g))
	root.AddCommand(cli.NewDeployCmd(g))
	root.AddCommand(cli.NewStatusCmd(g))
	root.AddCommand(cli.NewVersionCmd(version))
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(cli.ExitCode(err))
	}
}
