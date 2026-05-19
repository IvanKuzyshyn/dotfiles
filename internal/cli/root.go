// Package cli houses Cobra subcommand wiring and globals shared between
// commands.
package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// GlobalFlags holds flags shared by every dot subcommand. The Cobra root
// binds them as persistent flags; subcommands read the same struct.
type GlobalFlags struct {
	NonInteractive bool
	Verbose        bool
	ConfigDir      string // user overlay directory, default ~/.config/dot
	DotfilesDir    string // empty means: use Resolver fallback chain
}

// NewRoot builds the root Cobra command and binds GlobalFlags onto it.
// Subcommands are added by the caller (cmd/dot/main.go).
func NewRoot(g *GlobalFlags) *cobra.Command {
	root := &cobra.Command{
		Use:           "dot",
		Short:         "Dotfiles installer and config deployer",
		Long:          "dot installs tools and deploys configuration files from a dotfiles repository. Run `dot list` to see available tools.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&g.NonInteractive, "non-interactive", false,
		"force CLI output (no TUI), fail on prompts")
	root.PersistentFlags().BoolVarP(&g.Verbose, "verbose", "v", false,
		"print all log lines to stderr as they happen")
	root.PersistentFlags().StringVar(&g.ConfigDir, "config-dir", defaultConfigDir(),
		"directory containing overlay manifests")
	root.PersistentFlags().StringVar(&g.DotfilesDir, "dotfiles-dir", "",
		"path to the dotfiles repository (overrides DOTFILES_DIR env var)")
	return root
}

// defaultConfigDir returns ~/.config/dot. Falls back to "" if HOME is unset.
func defaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "dot")
}
