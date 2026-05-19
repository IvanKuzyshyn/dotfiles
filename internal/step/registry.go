package step

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
)

// Constructor builds a Step from the type-specific fields of a manifest.Step.
// `name` is the step's display name (manifest.Step.Name). `fields` is the
// inline-decoded map from manifest.Step.Fields.
type Constructor func(name string, fields map[string]any) (Step, error)

var (
	regMu sync.RWMutex
	reg   = map[string]Constructor{}
)

// Register attaches a Constructor to a step type. Called from step
// implementations' init() functions. Panics on duplicate registration.
func Register(typeName string, c Constructor) {
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := reg[typeName]; exists {
		panic(fmt.Sprintf("step: duplicate registration for type %q", typeName))
	}
	reg[typeName] = c
}

// Build constructs a Step from a manifest.Step using the registered
// Constructor for its Type. Returns an error if Type is unknown or the
// Constructor rejects the fields.
func Build(s manifest.Step) (Step, error) {
	regMu.RLock()
	c, ok := reg[s.Type]
	regMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("step: unknown type %q", s.Type)
	}
	return c(s.Name, s.Fields)
}

// RegisteredTypes returns the sorted list of step type names currently
// registered. Used by the manifest validator to check Step.Type fields.
func RegisteredTypes() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(reg))
	for k := range reg {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// isRegistered reports whether typeName has a Constructor.
func isRegistered(typeName string) bool {
	regMu.RLock()
	_, ok := reg[typeName]
	regMu.RUnlock()
	return ok
}

// resetRegistryForTest clears the registry. Tests use this to isolate from
// real init() registrations.
func resetRegistryForTest() {
	regMu.Lock()
	defer regMu.Unlock()
	reg = map[string]Constructor{}
}
