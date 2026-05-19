// Package toolsfs exposes the embedded tools directory. This file lives at
// the module root so the //go:embed directive can reference the sibling
// tools/ directory without a prohibited ".." path.
package toolsfs

import "embed"

//go:embed all:tools
var FS embed.FS

// Root is the path prefix inside FS where tool YAML files live.
const Root = "tools"
