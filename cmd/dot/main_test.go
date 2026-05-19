package main

import "testing"

// defaultKnown mirrors the subcommands registered in main(). Kept local to
// the test so it does not couple to the production code's dynamic derivation.
func defaultKnown() map[string]struct{} {
	return map[string]struct{}{
		"list":    {},
		"install": {},
		"update":  {},
		"deploy":  {},
		"status":  {},
		"version": {},
		"help":    {},
	}
}

func TestShouldLaunchTUI(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		isTTY bool
		want  bool
	}{
		{
			name:  "no args with TTY launches TUI",
			args:  nil,
			isTTY: true,
			want:  true,
		},
		{
			name:  "no args without TTY does not launch TUI",
			args:  nil,
			isTTY: false,
			want:  false,
		},
		{
			name:  "known subcommand routes to Cobra",
			args:  []string{"list"},
			isTTY: true,
			want:  false,
		},
		{
			name:  "--help short-circuits to Cobra",
			args:  []string{"--help"},
			isTTY: true,
			want:  false,
		},
		{
			name:  "-h short-circuits to Cobra",
			args:  []string{"-h"},
			isTTY: true,
			want:  false,
		},
		{
			name:  "bare --non-interactive disables TUI",
			args:  []string{"--non-interactive"},
			isTTY: true,
			want:  false,
		},
		{
			name:  "--non-interactive=true disables TUI",
			args:  []string{"--non-interactive=true"},
			isTTY: true,
			want:  false,
		},
		{
			name:  "--non-interactive=false disables TUI",
			args:  []string{"--non-interactive=false"},
			isTTY: true,
			want:  false,
		},
		{
			name:  "global flag with value still launches TUI",
			args:  []string{"--config-dir", "/x"},
			isTTY: true,
			want:  true,
		},
		{
			name:  "global flag followed by subcommand routes to Cobra",
			args:  []string{"--config-dir", "/x", "list"},
			isTTY: true,
			want:  false,
		},
		{
			// Documented limitation: when a flag value happens to match a
			// known subcommand name, shouldLaunchTUI treats it as a
			// subcommand. Asserting the current behavior to lock it in.
			name:  "flag value matching subcommand name is misclassified",
			args:  []string{"--config-dir", "list"},
			isTTY: true,
			want:  false,
		},
		{
			// The combined --flag=value form avoids the ambiguity above.
			name:  "flag=value form with subcommand-shaped value launches TUI",
			args:  []string{"--config-dir=list"},
			isTTY: true,
			want:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldLaunchTUI(tc.args, defaultKnown(), tc.isTTY)
			if got != tc.want {
				t.Fatalf("shouldLaunchTUI(%v, known, isTTY=%v) = %v, want %v", tc.args, tc.isTTY, got, tc.want)
			}
		})
	}
}
