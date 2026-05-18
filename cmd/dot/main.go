// Command dot installs tools and deploys configuration files from a
// dotfiles repository. See `dot --help` for usage.
package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/ivankuzyshyn/dotfiles/internal/cli"
	"github.com/ivankuzyshyn/dotfiles/internal/tui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

// knownSubcommands lists every subcommand name registered on the root. It is
// used by shouldLaunchTUI to distinguish "dot foo" (subcommand) from "dot
// --flag value" (flags only, TUI launch).
var knownSubcommands = map[string]struct{}{
	"list":    {},
	"install": {},
	"update":  {},
	"deploy":  {},
	"status":  {},
	"version": {},
	"help":    {},
}

func main() {
	g := &cli.GlobalFlags{}
	root := cli.NewRoot(g)
	root.AddCommand(cli.NewListCmd(g))
	root.AddCommand(cli.NewInstallCmd(g))
	root.AddCommand(cli.NewUpdateCmd(g))
	root.AddCommand(cli.NewDeployCmd(g))
	root.AddCommand(cli.NewStatusCmd(g))
	root.AddCommand(cli.NewVersionCmd(version))

	if shouldLaunchTUI(os.Args[1:]) {
		// Bind persistent flags from argv so --config-dir / --dotfiles-dir
		// are honored when launching the TUI. ParseFlags runs Cobra's
		// flag parser without executing any RunE.
		_ = root.ParseFlags(os.Args[1:])
		if err := tui.Launch(g); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(cli.ExitCode(err))
		}
		return
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(cli.ExitCode(err))
	}
}

// shouldLaunchTUI reports whether the binary should open the Bubble Tea
// picker instead of dispatching through Cobra. It returns true when:
//   - stdout is a TTY,
//   - argv contains no known subcommand,
//   - argv does not contain --non-interactive (in any form).
//
// Flag values that follow long flags (e.g. "--config-dir /x") are tolerated
// because we only inspect tokens against the known-subcommand set rather
// than counting non-flag positions.
func shouldLaunchTUI(args []string) bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	for _, a := range args {
		if a == "--non-interactive" {
			return false
		}
		if a == "-h" || a == "--help" {
			return false
		}
		if _, ok := knownSubcommands[a]; ok {
			return false
		}
	}
	return true
}
