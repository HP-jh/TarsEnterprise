// Package update provides auto-update capabilities for Tars ecosystem binaries.
package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Semver represents a semantic version.
type Semver struct {
	Major int
	Minor int
	Patch int
	Pre   string
}

// ParseSemver parses a version string like "v1.2.3" or "v1.2.3-beta".
func ParseSemver(s string) (Semver, error) {
	s = strings.TrimPrefix(s, "v")
	var sv Semver
	preIdx := strings.IndexAny(s, "-+")
	core := s
	if preIdx >= 0 {
		core = s[:preIdx]
		sv.Pre = s[preIdx:]
	}
	parts := strings.Split(core, ".")
	if len(parts) < 2 {
		return sv, fmt.Errorf("invalid semver: %s", s)
	}
	var err error
	sv.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return sv, fmt.Errorf("invalid major: %w", err)
	}
	sv.Minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return sv, fmt.Errorf("invalid minor: %w", err)
	}
	if len(parts) >= 3 {
		sv.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return sv, fmt.Errorf("invalid patch: %w", err)
		}
	}
	return sv, nil
}

// String returns the version string with 'v' prefix.
func (sv Semver) String() string {
	s := fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
	if sv.Pre != "" {
		s += sv.Pre
	}
	return s
}

// IsNewerThan returns true if this version is newer than other.
func (sv Semver) IsNewerThan(other Semver) bool {
	if sv.Major != other.Major {
		return sv.Major > other.Major
	}
	if sv.Minor != other.Minor {
		return sv.Minor > other.Minor
	}
	if sv.Patch != other.Patch {
		return sv.Patch > other.Patch
	}
	// Pre-release versions are older than release versions
	if sv.Pre == "" && other.Pre != "" {
		return true
	}
	if sv.Pre != "" && other.Pre == "" {
		return false
	}
	return sv.Pre > other.Pre
}
