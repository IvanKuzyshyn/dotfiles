package manifest

import (
	"errors"
	"fmt"
	"slices"
)

// Validate checks the registry of tools for schema correctness and dependency
// integrity. Returns a joined error with one entry per problem found, or nil
// if everything is valid. `knownTypes` is the list of step types registered
// via `step.Registry` — pass `step.RegisteredTypes()` from the caller.
func Validate(tools []Tool, knownTypes []string) error {
	var errs []error

	byName := make(map[string]int, len(tools))
	for i, t := range tools {
		if t.Name == "" {
			errs = append(errs, fmt.Errorf("tool[%d]: missing name", i))
			continue
		}
		if _, dup := byName[t.Name]; dup {
			errs = append(errs, fmt.Errorf("tool %q: duplicate name", t.Name))
		}
		byName[t.Name] = i
	}

	for _, t := range tools {
		if t.Name == "" {
			continue // already reported
		}
		if len(t.Steps) == 0 && len(t.Configs) == 0 {
			errs = append(errs, fmt.Errorf("tool %q: no steps or configs", t.Name))
		}
		for j, s := range t.Steps {
			if s.Type == "" {
				errs = append(errs, fmt.Errorf("tool %q step[%d]: missing type", t.Name, j))
				continue
			}
			if !slices.Contains(knownTypes, s.Type) {
				errs = append(errs, fmt.Errorf("tool %q step[%d]: unknown type %q", t.Name, j, s.Type))
			}
		}
		for _, dep := range t.DependsOn {
			if _, ok := byName[dep]; !ok {
				errs = append(errs, fmt.Errorf("tool %q: depends_on unknown tool %q", t.Name, dep))
			}
		}
	}

	if cycle := findCycle(tools, byName); cycle != nil {
		errs = append(errs, fmt.Errorf("dependency cycle: %v", cycle))
	}

	return errors.Join(errs...)
}

// findCycle returns the first cycle found via DFS coloring, or nil.
func findCycle(tools []Tool, byName map[string]int) []string {
	const (
		white = 0 // unvisited
		grey  = 1 // in current DFS stack
		black = 2 // fully visited
	)
	color := make(map[string]int, len(tools))
	var stack []string
	var cycle []string

	var dfs func(name string) bool
	dfs = func(name string) bool {
		idx, ok := byName[name]
		if !ok {
			return false // unknown dep already reported elsewhere
		}
		switch color[name] {
		case grey:
			// Found a cycle. Extract the loop from the stack.
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i] == name {
					cycle = append([]string(nil), stack[i:]...)
					cycle = append(cycle, name)
					return true
				}
			}
			cycle = append(cycle, name) // shouldn't happen
			return true
		case black:
			return false
		}
		color[name] = grey
		stack = append(stack, name)
		for _, dep := range tools[idx].DependsOn {
			if dfs(dep) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		color[name] = black
		return false
	}

	for _, t := range tools {
		if t.Name == "" {
			continue
		}
		if color[t.Name] == white {
			if dfs(t.Name) {
				return cycle
			}
		}
	}
	return nil
}
