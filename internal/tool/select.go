package tool

import (
	"fmt"
	"sort"
	"strings"
)

// Select resolves user-facing selection into a slice of *Tool. Exactly one
// of (all, tag, names) determines the result; ordering preference for the
// caller is: all → tag → names. Passing multiple is allowed and the first
// non-empty wins.
func Select(reg *Registry, names []string, all bool, tag string) ([]*Tool, error) {
	if reg == nil {
		return nil, nil
	}
	if all {
		return reg.All(), nil
	}
	if tag != "" {
		var out []*Tool
		for _, t := range reg.All() {
			for _, tg := range t.Tags {
				if tg == tag {
					out = append(out, t)
					break
				}
			}
		}
		return out, nil
	}
	if len(names) == 0 {
		return nil, nil
	}
	var out []*Tool
	for _, n := range names {
		t, ok := reg.Get(n)
		if !ok {
			return nil, fmt.Errorf("unknown tool %q%s", n, didYouMean(reg, n))
		}
		out = append(out, t)
	}
	return out, nil
}

func didYouMean(reg *Registry, name string) string {
	var matches []string
	low := strings.ToLower(name)
	for _, t := range reg.All() {
		ln := strings.ToLower(t.Name)
		if strings.HasPrefix(ln, low) || strings.Contains(ln, low) {
			matches = append(matches, t.Name)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	return fmt.Sprintf(" (did you mean: %s?)", strings.Join(matches, ", "))
}

// ExpandDeps transitively pulls in DependsOn targets. If noDeps is true,
// returns the selection unchanged but errors if any required dep is not
// present in the registry.
func ExpandDeps(reg *Registry, selected []*Tool, noDeps bool) ([]*Tool, error) {
	if noDeps {
		for _, t := range selected {
			for _, dep := range t.DependsOn {
				if _, ok := reg.Get(dep); !ok {
					return nil, fmt.Errorf("tool %q depends on %q which is not installed", t.Name, dep)
				}
			}
		}
		return selected, nil
	}

	included := make(map[string]*Tool, len(selected))
	var queue []*Tool
	for _, t := range selected {
		if _, seen := included[t.Name]; !seen {
			included[t.Name] = t
			queue = append(queue, t)
		}
	}
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		for _, dep := range t.DependsOn {
			if _, seen := included[dep]; seen {
				continue
			}
			dt, ok := reg.Get(dep)
			if !ok {
				return nil, fmt.Errorf("tool %q depends on unknown tool %q", t.Name, dep)
			}
			included[dep] = dt
			queue = append(queue, dt)
		}
	}

	out := make([]*Tool, 0, len(included))
	for _, t := range included {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Sort topologically orders tools so dependencies precede dependents.
// Deterministic by name on ties (Kahn's algorithm). Panics if a cycle is
// detected (validation should have caught it earlier).
func Sort(tools []*Tool) []*Tool {
	indexOf := make(map[string]int, len(tools))
	for i, t := range tools {
		indexOf[t.Name] = i
	}
	// Build adjacency: depToDependents[dep] = list of tool names that depend on dep
	indegree := make(map[string]int, len(tools))
	depToDependents := make(map[string][]string, len(tools))
	for _, t := range tools {
		indegree[t.Name] = 0
	}
	for _, t := range tools {
		for _, dep := range t.DependsOn {
			if _, present := indexOf[dep]; !present {
				continue // dep not in this set; ignored
			}
			indegree[t.Name]++
			depToDependents[dep] = append(depToDependents[dep], t.Name)
		}
	}
	// Initialize the ready set: tools with no in-degree in this set.
	var ready []string
	for _, t := range tools {
		if indegree[t.Name] == 0 {
			ready = append(ready, t.Name)
		}
	}
	sort.Strings(ready)
	var out []*Tool
	processed := 0
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		out = append(out, tools[indexOf[name]])
		processed++
		dependents := depToDependents[name]
		sort.Strings(dependents)
		for _, d := range dependents {
			indegree[d]--
			if indegree[d] == 0 {
				ready = append(ready, d)
			}
		}
		sort.Strings(ready) // keep ready sorted for determinism
	}
	if processed < len(tools) {
		panic("tool.Sort: cycle detected (should have been caught by manifest.Validate)")
	}
	return out
}
