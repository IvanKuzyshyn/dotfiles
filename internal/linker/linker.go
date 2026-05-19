// Package linker walks tool configs and produces symlink decisions and
// conflict reports. Apply (in actions.go) executes those decisions.
package linker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	xfs "github.com/ivankuzyshyn/dotfiles/internal/fs"
	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

// DecisionKind classifies what the linker should do for one target.
type DecisionKind int

const (
	DecideSymlink   DecisionKind = iota // target missing; create symlink
	DecideAlreadyOk                     // target is the correct symlink
	DecideConflict                      // something else is there
)

// ExistingKind labels what the linker found at a Conflict's target.
type ExistingKind string

const (
	ExistingFile         ExistingKind = "file"
	ExistingDir          ExistingKind = "dir"
	ExistingSymlinkOther ExistingKind = "symlink-other"
)

// Decision is one source→target classification.
type Decision struct {
	Source string // absolute path to source file under configs/
	Target string // absolute path to target under target dir
	Kind   DecisionKind
}

// Conflict carries the target path and what's existing there. Used by the
// runner to surface ConflictPrompt events.
type Conflict struct {
	Target       string
	ExistingKind ExistingKind
}

// Plan summarizes a linker.Inspect call.
type Plan struct {
	Decisions []Decision
	Conflicts []Conflict
}

// Inspect computes the linker plan for a tool's configs without applying
// anything. The result is deterministic (file paths sorted).
//
// All filesystem access goes through env.FS so tests can use the in-memory
// fake. The dotfilesDir is the root of the dotfiles repo (used to anchor
// configs[].Source paths). homeDir is used to expand "~" in target paths.
func Inspect(configs []manifest.Config, env step.Env) (Plan, error) {
	if env.FS == nil {
		return Plan{}, errors.New("linker.Inspect: env.FS is nil")
	}
	if env.DotfilesDir == "" {
		return Plan{}, errors.New("linker.Inspect: env.DotfilesDir is empty")
	}

	var plan Plan
	for _, c := range configs {
		sourceRoot := filepath.Join(env.DotfilesDir, "configs", c.Source)
		targetRoot := expandHome(c.Target, env.HomeDir)
		err := env.FS.Walk(sourceRoot, func(path string, info xfs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(sourceRoot, path)
			if err != nil {
				return err
			}
			target := filepath.Join(targetRoot, rel)
			d, conflict, err := decide(env.FS, path, target)
			if err != nil {
				return err
			}
			plan.Decisions = append(plan.Decisions, d)
			if conflict != nil {
				plan.Conflicts = append(plan.Conflicts, *conflict)
			}
			return nil
		})
		if err != nil {
			return plan, fmt.Errorf("walk %s: %w", sourceRoot, err)
		}
	}
	return plan, nil
}

func decide(fs xfs.FS, source, target string) (Decision, *Conflict, error) {
	info, err := fs.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Decision{Source: source, Target: target, Kind: DecideSymlink}, nil, nil
		}
		return Decision{}, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := fs.Readlink(target)
		if err != nil {
			return Decision{}, nil, err
		}
		if link == source {
			return Decision{Source: source, Target: target, Kind: DecideAlreadyOk}, nil, nil
		}
		c := Conflict{Target: target, ExistingKind: ExistingSymlinkOther}
		return Decision{Source: source, Target: target, Kind: DecideConflict}, &c, nil
	}
	if info.IsDir() {
		c := Conflict{Target: target, ExistingKind: ExistingDir}
		return Decision{Source: source, Target: target, Kind: DecideConflict}, &c, nil
	}
	c := Conflict{Target: target, ExistingKind: ExistingFile}
	return Decision{Source: source, Target: target, Kind: DecideConflict}, &c, nil
}

// expandHome replaces a leading "~/" or "~" in p with home. If home is empty,
// p is returned unchanged.
func expandHome(p, home string) string {
	if home == "" {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
