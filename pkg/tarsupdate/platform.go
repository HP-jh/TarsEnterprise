package update

import (
	"fmt"
	"runtime"
	"strings"
)

// Platform holds the current OS and architecture.
type Platform struct {
	OS   string
	Arch string
}

// CurrentPlatform returns the detected platform.
func CurrentPlatform() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// AssetName generates the GitHub release asset name for a given binary base name.
// Example: AssetName("twg", "v0.2.0") → "twg-v0.2.0-linux-amd64"
func (p Platform) AssetName(binary, version string) string {
	name := fmt.Sprintf("%s-%s-%s-%s", binary, version, p.OS, p.Arch)
	if p.OS == "windows" {
		name += ".exe"
	}
	return name
}

// AssetNameForBinary returns the asset name for a specific binary.
func AssetNameForBinary(binary, version string) string {
	return CurrentPlatform().AssetName(binary, version)
}

// IsSupported reports whether the platform has prebuilt binaries.
func (p Platform) IsSupported() bool {
	supported := map[string][]string{
		"linux":   {"amd64", "arm64"},
		"darwin":  {"amd64", "arm64"},
		"windows": {"amd64", "arm64"},
		"android": {"arm64"},
	}
	arches, ok := supported[p.OS]
	if !ok {
		return false
	}
	for _, a := range arches {
		if a == p.Arch {
			return true
		}
	}
	return false
}

// NormalizeArch maps common architecture names to Go-style arch.
func NormalizeArch(arch string) string {
	switch strings.ToLower(arch) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return arch
	}
}
