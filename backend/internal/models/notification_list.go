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

package models

import (
	"github.com/octobud-hq/octobud/backend/internal/db"
)

// ListOptions captures the available filtering and pagination controls for listing notifications.
type ListOptions struct {
	//nolint:lll // Long comment with example
	Query          string // Combined query string with key-value pairs and free text (e.g., "repo:cli/cli urgent PR -author:bot")
	Page           int
	PageSize       int
	IncludeSubject bool // Whether to include subjectRaw in the response (default: false to reduce payload size)
}

// ListResult is the normalized output of a filtered list request.
type ListResult struct {
	Notifications []db.Notification
	Total         int64
	Page          int
	PageSize      int
}

// ListDetailsResult is the output of a filtered list request with enriched notification data.
type ListDetailsResult struct {
	Notifications []Notification
	Total         int64
	Page          int
	PageSize      int
}

// ListPollResult is the output of a filtered list request with only essential fields for polling.
type ListPollResult struct {
	Notifications []PollNotification
	Total         int64
	Page          int
	PageSize      int
}
