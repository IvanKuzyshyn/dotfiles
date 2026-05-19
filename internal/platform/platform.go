// Package platform detects and validates operating system and architecture.
package platform

import "runtime"

// Platform represents the detected OS and architecture.
type Platform struct {
	OS   string
	Arch string
}

// Detect returns the current platform.
func Detect() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// Supports checks whether this platform's OS is in the allowed list.
// Returns true if allowed is empty or if the platform's OS is in allowed.
func (p Platform) Supports(allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == p.OS {
			return true
		}
	}
	return false
}
