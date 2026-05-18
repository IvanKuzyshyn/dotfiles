// Package runner orchestrates installing tools by executing their Steps in
// order and emitting structured events.
package runner

import "github.com/ivankuzyshyn/dotfiles/internal/tool"

// Plan is an ordered list of tools the runner will execute, top of slice
// first. Callers (CLI/TUI) build Plans via tool.Sort after dependency
// expansion.
type Plan struct {
	Tools []*tool.Tool
}
