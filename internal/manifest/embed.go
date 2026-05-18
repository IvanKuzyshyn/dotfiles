package manifest

import (
	"fmt"
	"io/fs"
	"path"

	toolsfs "github.com/ivankuzyshyn/dotfiles"
	"gopkg.in/yaml.v3"
)

// LoadEmbedded reads all *.yaml files compiled into the binary.
func LoadEmbedded() ([]Tool, error) {
	root := toolsfs.Root
	entries, err := toolsfs.FS.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read embedded tools: %w", err)
	}
	var tools []Tool
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".yaml" {
			continue
		}
		data, err := fs.ReadFile(toolsfs.FS, path.Join(root, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read embedded %s: %w", e.Name(), err)
		}
		var t Tool
		if err := yaml.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parse embedded %s: %w", e.Name(), err)
		}
		t.Source = "embedded:" + e.Name()
		tools = append(tools, t)
	}
	return tools, nil
}
