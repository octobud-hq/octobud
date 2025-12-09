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
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int // 1 if v1 > v2, -1 if v1 < v2, 0 if equal
	}{
		// Basic version comparison
		{"basic newer", "1.0.1", "1.0.0", 1},
		{"basic older", "1.0.0", "1.0.1", -1},
		{"basic equal", "1.0.0", "1.0.0", 0},
		{"major version", "2.0.0", "1.9.9", 1},
		{"minor version", "1.1.0", "1.0.9", 1},

		// Pre-release versions (stable > pre-release)
		{"stable vs pre-release", "1.0.0", "1.0.0-beta", 1},
		{"pre-release vs stable", "1.0.0-beta", "1.0.0", -1},

		// Numeric pre-release identifiers (the bug we're fixing)
		{"numeric pre-release newer", "1.0.0-beta.10", "1.0.0-beta.2", 1},
		{"numeric pre-release older", "1.0.0-beta.2", "1.0.0-beta.10", -1},
		{"numeric pre-release equal", "1.0.0-beta.2", "1.0.0-beta.2", 0},

		// Pre-release precedence
		{"alpha vs beta", "1.0.0-alpha", "1.0.0-beta", -1},
		{"beta vs rc", "1.0.0-beta", "1.0.0-rc", -1},
		{"rc vs stable", "1.0.0-rc.1", "1.0.0", -1},

		// Dot-separated pre-release identifiers
		{"dot-separated pre-release", "1.0.0-alpha.1.beta", "1.0.0-alpha.1.alpha", 1},

		// Build metadata (should be ignored in comparison)
		{"build metadata equal", "1.0.0+build.123", "1.0.0+build.456", 0},
		{"build metadata with pre-release", "1.0.0-beta.1+build.123", "1.0.0-beta.1+build.456", 0},

		// Version with 'v' prefix (should be stripped)
		{"v prefix newer", "v1.0.1", "v1.0.0", 1},
		{"v prefix equal", "v1.0.0", "1.0.0", 0},
		{"v prefix mixed", "v1.0.0", "1.0.0-beta", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name           string
		latest         string
		current        string
		expectedNewer  bool
	}{
		{"newer version", "1.0.1", "1.0.0", true},
		{"older version", "1.0.0", "1.0.1", false},
		{"equal version", "1.0.0", "1.0.0", false},
		{"pre-release newer numeric", "1.0.0-beta.10", "1.0.0-beta.2", true},
		{"stable vs pre-release", "1.0.0", "1.0.0-beta", true},
		{"pre-release vs stable", "1.0.0-beta", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNewer(tt.latest, tt.current)
			if result != tt.expectedNewer {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.latest, tt.current, result, tt.expectedNewer)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"v1.0.0-beta", "1.0.0-beta"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

