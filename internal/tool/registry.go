package tool

import (
	"errors"
	"fmt"
	"sort"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
)

// Registry stores built Tools keyed by Name. Use NewRegistry to construct
// from a slice of parsed manifests.
type Registry struct {
	byName map[string]*Tool
}

// NewRegistry builds Tools from parsed manifests and stores them by Name.
// If any tool fails to build (e.g., unknown step type), all errors are
// joined and returned. The Registry is still returned with partial
// contents so callers can inspect what loaded successfully.
func NewRegistry(ms []manifest.Tool) (*Registry, error) {
	r := &Registry{byName: make(map[string]*Tool, len(ms))}
	var errs []error
	for _, m := range ms {
		t, err := Build(m)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, dup := r.byName[t.Name]; dup {
			errs = append(errs, fmt.Errorf("tool %q: duplicate", t.Name))
			continue
		}
		r.byName[t.Name] = t
	}
	if len(errs) > 0 {
		return r, errors.Join(errs...)
	}
	return r, nil
}

// Get returns the tool with the given name and a boolean indicating
// whether it was found.
func (r *Registry) Get(name string) (*Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

// All returns every Tool sorted by Name.
func (r *Registry) All() []*Tool {
	out := make([]*Tool, 0, len(r.byName))
	for _, t := range r.byName {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
