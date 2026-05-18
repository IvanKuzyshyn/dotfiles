package platform_test

import (
	"testing"

	"github.com/ivankuzyshyn/dotfiles/internal/platform"
)

func TestDetect(t *testing.T) {
	p := platform.Detect()
	if p.OS == "" || p.Arch == "" {
		t.Fatalf("expected non-empty OS and Arch, got %+v", p)
	}
}

func TestSupports(t *testing.T) {
	p := platform.Platform{OS: "darwin"}
	if !p.Supports([]string{"darwin", "linux"}) {
		t.Errorf("darwin should be in [darwin, linux]")
	}
	if !p.Supports(nil) {
		t.Errorf("empty platforms list should match any OS")
	}
	if p.Supports([]string{"linux"}) {
		t.Errorf("darwin should not match [linux]")
	}
}
