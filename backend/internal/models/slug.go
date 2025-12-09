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

// Package models provides slugification utilities.
package models

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	repeatedHyphen  = regexp.MustCompile(`-+`)
)

// Slugify converts the provided name into a URL-safe slug following the same
// normalization rules used in database migrations.
func Slugify(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return ""
	}

	slug := nonAlphanumeric.ReplaceAllString(lower, "-")
	slug = repeatedHyphen.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	return slug
}
