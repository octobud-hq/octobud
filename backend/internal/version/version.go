// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package version provides version information for Octobud.
package version

// Version is the version of Octobud. This can be set at build time using:
//
//	go build -ldflags "-X github.com/octobud-hq/octobud/backend/internal/version.Version=v1.0.0"
//
// If not set at build time, it defaults to "dev".
var Version = "dev"

// Get returns the version string.
func Get() string {
	return Version
}
