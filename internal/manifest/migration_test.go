package manifest_test

import (
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/manifest"
	"github.com/ivankuzyshyn/dotfiles/internal/step"
	"github.com/ivankuzyshyn/dotfiles/internal/tool"
)

// legacyTools is the pinned set of tools that existed in the bash-era
// dotfiles (install.sh + bootstrap.sh). The Go rewrite must surface all
// of them via the embedded manifest registry to preserve user-facing
// parity during the migration.
var legacyTools = []string{
	"homebrew", "rust", "oh-my-zsh", "nvm", "claude-code", "git-hooks",
	"git", "zsh", "vim", "ghostty", "k9s", "mise",
}

func TestMigrationParity_AllLegacyToolsPresent(t *testing.T) {
	manifests, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	reg, err := tool.NewRegistry(manifests)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	for _, name := range legacyTools {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("legacy tool %q is missing from the registry", name)
		}
	}
}

func TestMigrationParity_ClaudeTracked(t *testing.T) {
	manifests, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	reg, err := tool.NewRegistry(manifests)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if _, ok := reg.Get("claude"); !ok {
		t.Errorf("tool %q is missing from the registry", "claude")
	}
}

// TestMigrationParity_EmbeddedManifestsPassValidation guards against schema
// drift between the embedded manifests and the validator. Every CLI command
// except `dot list` runs `manifest.Validate` before dispatching, so a
// validation regression here breaks the binary end-to-end even though
// `LoadEmbedded` and `NewRegistry` succeed.
func TestMigrationParity_EmbeddedManifestsPassValidation(t *testing.T) {
	manifests, err := manifest.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	if err := manifest.Validate(manifests, step.RegisteredTypes()); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
