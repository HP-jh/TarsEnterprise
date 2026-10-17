package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Metadata   string
}

// ParseVersion parses a semver string like "v0.2.1" or "v1.0.0-beta.1".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	var v Version

	// Split metadata
	if i := strings.Index(s, "+"); i >= 0 {
		v.Metadata = s[i+1:]
		s = s[:i]
	}

	// Split prerelease
	if i := strings.Index(s, "-"); i >= 0 {
		v.Prerelease = s[i+1:]
		s = s[:i]
	}

	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return v, fmt.Errorf("invalid version: %s", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return v, fmt.Errorf("invalid major: %w", err)
	}
	v.Major = major

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return v, fmt.Errorf("invalid minor: %w", err)
	}
	v.Minor = minor

	if len(parts) >= 3 {
		patch, err := strconv.Atoi(parts[2])
		if err != nil {
			return v, fmt.Errorf("invalid patch: %w", err)
		}
		v.Patch = patch
	}

	return v, nil
}

// String returns the version string with "v" prefix.
func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Metadata != "" {
		s += "+" + v.Metadata
	}
	return s
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}
	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}
	// Prerelease: no prerelease > any prerelease
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease != other.Prerelease {
		// Simple string compare for prerelease
		if v.Prerelease > other.Prerelease {
			return 1
		}
		return -1
	}
	return 0
}

// IsNewerThan reports whether v is newer (greater) than other.
func (v Version) IsNewerThan(other Version) bool {
	return v.Compare(other) > 0
}
