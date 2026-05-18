package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/spf13/cobra"
)

// NewListCmd returns the `dot list` subcommand. It enumerates every manifest
// discoverable in embedded + overlay locations, without filtering.
func NewListCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known tools (embedded + overlay manifests)",
		RunE: func(c *cobra.Command, _ []string) error {
			tools, err := loadAllManifests(g)
			if err != nil {
				return err
			}
			out := c.OutOrStdout()
			if len(tools) == 0 {
				fmt.Fprintln(out, "(no tools found)")
				return nil
			}
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tSOURCE\tPLATFORMS\tTAGS")
			for _, t := range tools {
				platforms := strings.Join(t.Platforms, ",")
				if platforms == "" {
					platforms = "any"
				}
				tags := strings.Join(t.Tags, ",")
				if tags == "" {
					tags = "-"
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", t.Name, sourceLabel(t.Source), platforms, tags)
			}
			return tw.Flush()
		},
	}
}

// loadAllManifests collects embedded manifests plus overlays from the
// config directory and dotfiles directory (if either has a tools/ subdir).
func loadAllManifests(g *GlobalFlags) ([]manifest.Tool, error) {
	embedded, err := manifest.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("load embedded: %w", err)
	}
	var overlays []manifest.Tool
	for _, dir := range overlayDirs(g) {
		more, err := manifest.LoadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", dir, err)
		}
		overlays = append(overlays, more...)
	}
	return manifest.Merge(embedded, overlays), nil
}

// overlayDirs returns existing directories to check for overlay manifests.
// The order is: $config-dir/tools, then $dotfiles-dir/tools (when set).
func overlayDirs(g *GlobalFlags) []string {
	var dirs []string
	if g.ConfigDir != "" {
		dirs = append(dirs, filepath.Join(g.ConfigDir, "tools"))
	}
	if g.DotfilesDir != "" {
		dirs = append(dirs, filepath.Join(g.DotfilesDir, "tools"))
	}
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			out = append(out, d)
		}
	}
	return out
}

// sourceLabel produces a compact display of the Tool.Source field.
//
//	"embedded:foo.yaml" → "embedded"
//	"/abs/path/foo.yaml" → "overlay"
//	"" → "-"
func sourceLabel(s string) string {
	switch {
	case s == "":
		return "-"
	case strings.HasPrefix(s, "embedded:"):
		return "embedded"
	default:
		return "overlay"
	}
}
