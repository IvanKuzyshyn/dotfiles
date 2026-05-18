// Command dot installs tools and deploys configuration files from a
// dotfiles repository. See `dot --help` for usage.
package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/ivankuzyshyn/dotfiles/internal/cli"
	"github.com/ivankuzyshyn/dotfiles/internal/tui"
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

	// Build the known-subcommand set dynamically from Cobra so future
	// additions cannot drift out of sync with shouldLaunchTUI.
	known := make(map[string]struct{})
	for _, c := range root.Commands() {
		known[c.Name()] = struct{}{}
		for _, a := range c.Aliases {
			known[a] = struct{}{}
		}
	}

	if shouldLaunchTUI(os.Args[1:], known, term.IsTerminal(int(os.Stdout.Fd()))) {
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
//   - isTTY is true,
//   - argv contains no known subcommand,
//   - argv does not contain --non-interactive (in any form, including
//     --non-interactive=true).
//
// Flag values that follow long flags (e.g. "--config-dir /x") are tolerated
// because we only inspect tokens against the known-subcommand set rather
// than counting non-flag positions.
//
// Known limitation: if a flag value happens to match a subcommand name
// (e.g. "--config-dir list" where "list" is the value), this function will
// treat the value as a subcommand and return false. Use the combined form
// (--config-dir=list) to avoid this ambiguity.
func shouldLaunchTUI(args []string, known map[string]struct{}, isTTY bool) bool {
	if !isTTY {
		return false
	}
	for _, a := range args {
		if a == "--non-interactive" || strings.HasPrefix(a, "--non-interactive=") {
			return false
		}
		if a == "-h" || a == "--help" {
			return false
		}
		if _, ok := known[a]; ok {
			return false
		}
	}
	return true
}
