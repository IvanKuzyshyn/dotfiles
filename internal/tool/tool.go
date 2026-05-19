// Package tool defines the runtime Tool model and a Registry for looking
// up tools after manifests have been parsed and validated.
package tool

import (
	"errors"
	"fmt"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
)

// Tool is a parsed manifest with its Steps materialized into Step values.
// It is the runtime form consumed by the runner.
type Tool struct {
	Name        string
	Description string
	Platforms   []string
	Tags        []string
	DependsOn   []string
	Steps       []step.Step
	Configs     []manifest.Config
	Source      string // "embedded:<file>" or absolute path
}

// Build materializes a Tool from a parsed manifest. Each manifest.Step is
// run through step.Build; constructor errors are accumulated and returned
// as a single joined error.
func Build(m manifest.Tool) (*Tool, error) {
	t := &Tool{
		Name:        m.Name,
		Description: m.Description,
		Platforms:   m.Platforms,
		Tags:        m.Tags,
		DependsOn:   m.DependsOn,
		Configs:     m.Configs,
		Source:      m.Source,
	}
	var errs []error
	t.Steps = make([]step.Step, 0, len(m.Steps))
	for i, ms := range m.Steps {
		s, err := step.Build(ms)
		if err != nil {
			errs = append(errs, fmt.Errorf("tool %q step[%d]: %w", m.Name, i, err))
			continue
		}
		t.Steps = append(t.Steps, s)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return t, nil
}
