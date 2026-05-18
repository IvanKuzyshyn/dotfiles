// Package manifest defines the YAML schema for tool manifests and helpers
// for loading and validating them.
package manifest

// Tool is the unmarshalled form of a tool manifest YAML.
type Tool struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Platforms   []string `yaml:"platforms,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty"`
	Steps       []Step   `yaml:"steps"`
	Configs     []Config `yaml:"configs,omitempty"`
	// Source records where this manifest was loaded from. "embedded" for
	// manifests compiled into the binary, otherwise the absolute path of
	// the YAML file. Set by the loader, not by users; the YAML tag is "-"
	// so it never appears in serialized form.
	Source string `yaml:"-"`
}

// Step is one declarative install/check action. Type-specific fields are
// captured in Fields and decoded by the step constructors.
type Step struct {
	Type   string         `yaml:"type"`
	Name   string         `yaml:"name,omitempty"`
	Fields map[string]any `yaml:",inline"`
}

// Config maps a directory under configs/ in the dotfiles repo to a target
// directory in the user's home. Both paths can include "~" which the
// linker expands to the user's home directory.
type Config struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}
