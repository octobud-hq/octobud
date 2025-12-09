// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package update

import (
	"strings"

	"github.com/hashicorp/go-version"
)

// CompareVersions compares two semantic versions using HashiCorp's go-version library.
// Returns:
//   - 1 if v1 > v2 (v1 is newer)
//   - -1 if v1 < v2 (v1 is older)
//   - 0 if v1 == v2
//
// Supports full SemVer 2.0.0 specification including:
// - Standard versions: "1.0.0", "2.1.3", "0.1.0"
// - Pre-release versions: "1.0.0-beta.1", "1.0.0-rc.2", "1.0.0-alpha.1.beta"
// - Build metadata: "1.0.0+20230101", "1.0.0-beta.1+build.123"
// - Numeric pre-release identifiers: correctly compares "beta.10" > "beta.2"
//
// The 'v' prefix is automatically stripped if present.
func CompareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Parse versions using HashiCorp's library (SemVer 2.0.0 compliant)
	ver1, err1 := version.NewVersion(v1)
	ver2, err2 := version.NewVersion(v2)

	// If either version fails to parse, fall back to string comparison
	// This handles edge cases where versions might not be strictly SemVer compliant
	if err1 != nil || err2 != nil {
		// If both fail, treat as equal (shouldn't happen in practice)
		if err1 != nil && err2 != nil {
			return 0
		}
		// If only one fails, the valid one is considered newer
		if err1 != nil {
			return -1 // v2 is valid, so v2 > v1
		}
		return 1 // v1 is valid, so v1 > v2
	}

	// Use the library's built-in comparison (SemVer 2.0.0 compliant)
	if ver1.GreaterThan(ver2) {
		return 1
	}
	if ver1.LessThan(ver2) {
		return -1
	}
	return 0
}

// IsNewer returns true if latestVersion is newer than currentVersion.
// Uses SemVer 2.0.0 compliant comparison, properly handling:
// - Pre-release versions (beta.10 > beta.2)
// - Dot-separated pre-release identifiers
// - Build metadata (ignored in comparison)
func IsNewer(latestVersion, currentVersion string) bool {
	return CompareVersions(latestVersion, currentVersion) > 0
}

// NormalizeVersion normalizes a version string by removing the 'v' prefix
// and ensuring it's in a format that can be parsed by SemVer libraries.
// Useful for consistent version formatting.
func NormalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// ParseVersion parses a version string and returns a HashiCorp version object.
// Returns an error if the version string is invalid.
func ParseVersion(v string) (*version.Version, error) {
	v = NormalizeVersion(v)
	return version.NewVersion(v)
}
