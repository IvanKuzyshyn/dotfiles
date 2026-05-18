package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ivankuzyshyn/dotfiles/internal/platform"
	"gopkg.in/yaml.v3"
)

// LoadFile reads and parses one manifest file. Source is set to the
// absolute path of the file.
func LoadFile(path string) (Tool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Tool{}, fmt.Errorf("read %s: %w", path, err)
	}
	var t Tool
	if err := yaml.Unmarshal(data, &t); err != nil {
		return Tool{}, fmt.Errorf("parse %s: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	t.Source = abs
	return t, nil
}

// LoadDir reads every *.yaml file under dir non-recursively. Returns a
// joined error if any file fails to parse, but other files still load.
func LoadDir(dir string) ([]Tool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var tools []Tool
	var errs []error
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		t, err := LoadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tools = append(tools, t)
	}
	return tools, errors.Join(errs...)
}

// Merge combines embedded defaults with an overlay set. When two tools
// share the same Name, the overlay tool wins entirely (no field merge).
// Tools without conflicts are kept as-is. The returned slice is ordered
// alphabetically by Name for determinism.
func Merge(embedded, overlay []Tool) []Tool {
	by := make(map[string]Tool, len(embedded)+len(overlay))
	for _, t := range embedded {
		by[t.Name] = t
	}
	for _, t := range overlay {
		by[t.Name] = t // overlay replaces embedded
	}
	out := make([]Tool, 0, len(by))
	for _, t := range by {
		out = append(out, t)
	}
	sortToolsByName(out)
	return out
}

// FilterPlatform drops tools whose Platforms list excludes p. A tool with
// an empty Platforms list is considered universal and kept.
func FilterPlatform(tools []Tool, p platform.Platform) []Tool {
	out := make([]Tool, 0, len(tools))
	for _, t := range tools {
		if p.Supports(t.Platforms) {
			out = append(out, t)
		}
	}
	return out
}

func sortToolsByName(tools []Tool) {
	for i := 1; i < len(tools); i++ {
		for j := i; j > 0 && tools[j-1].Name > tools[j].Name; j-- {
			tools[j-1], tools[j] = tools[j], tools[j-1]
		}
	}
}
