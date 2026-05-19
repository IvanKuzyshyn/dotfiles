package cli

import (
	"fmt"
	"strings"

	"github.com/ivankuzyshyn/dotfiles/internal/runner"
)

// FormatSummary produces a multi-line end-of-run report. logPath may be
// empty when no log file was written; the corresponding line is omitted.
func FormatSummary(result runner.Result, logPath string) string {
	s, sk, f := result.Counts()
	var b strings.Builder
	b.WriteString("\n─────────────────────────────────────────────────\n")
	fmt.Fprintf(&b, "%d succeeded, %d skipped, %d failed\n", s, sk, f)

	failed := toolsByState(result, runner.Failed)
	if len(failed) > 0 {
		b.WriteString("\nFailed tools:\n")
		for _, tr := range failed {
			detail := errDetail(tr.Err)
			if detail == "" {
				fmt.Fprintf(&b, "  ✗ %s\n", tr.Tool.Name)
			} else {
				fmt.Fprintf(&b, "  ✗ %s: %s\n", tr.Tool.Name, detail)
			}
		}
	}

	skipped := toolsByState(result, runner.Skipped)
	if len(skipped) > 0 {
		b.WriteString("\nSkipped tools:\n")
		for _, tr := range skipped {
			detail := errDetail(tr.Err)
			if detail == "" {
				fmt.Fprintf(&b, "  ~ %s\n", tr.Tool.Name)
			} else {
				fmt.Fprintf(&b, "  ~ %s (%s)\n", tr.Tool.Name, detail)
			}
		}
	}

	if logPath != "" {
		fmt.Fprintf(&b, "\nLogs: %s\n", logPath)
	}

	if len(failed) > 0 {
		names := make([]string, 0, len(failed))
		for _, tr := range failed {
			names = append(names, tr.Tool.Name)
		}
		fmt.Fprintf(&b, "Retry failed: dot install %s\n", strings.Join(names, " "))
	}

	return b.String()
}

// errDetail returns a truncated (~200 char) representation of err, or "".
func errDetail(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	const max = 200
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// toolsByState filters result.Tools by state.
func toolsByState(result runner.Result, state runner.State) []runner.ToolResult {
	var out []runner.ToolResult
	for _, tr := range result.Tools {
		if tr.State == state {
			out = append(out, tr)
		}
	}
	return out
}
