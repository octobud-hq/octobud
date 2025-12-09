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

package query

import (
	"github.com/octobud-hq/octobud/backend/internal/db"
)

// ApplyInboxDefaults adds default inbox filters to a query
// (excludes archived, snoozed, muted, and filtered notifications)
func ApplyInboxDefaults(query db.NotificationQuery) db.NotificationQuery {
	// Use strftime to output ISO 8601 format matching stored RFC3339 timestamps
	nowFunc := "strftime('%Y-%m-%dT%H:%M:%SZ', 'now')"

	defaultFilters := []string{
		"n.archived = 0",
		"(n.snoozed_until IS NULL OR n.snoozed_until <= " + nowFunc + ")",
		"n.muted = 0",
		"n.filtered = 0",
	}

	query.Where = append(query.Where, defaultFilters...)
	return query
}

// ApplyMutedOnlyDefaults adds default filters to exclude muted only
// (allows archived, snoozed, and filtered notifications)
func ApplyMutedOnlyDefaults(query db.NotificationQuery) db.NotificationQuery {
	defaultFilters := []string{
		"n.muted = 0",
	}

	query.Where = append(query.Where, defaultFilters...)
	return query
}
