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

package db

// SystemView constants
const (
	SystemViewInbox      = "inbox"
	SystemViewEverything = "everything"
	SystemViewStarred    = "starred"
	SystemViewArchive    = "archive"
	SystemViewSnoozed    = "snoozed"
)

// SystemViewDefinition defines a virtual system view
type SystemViewDefinition struct {
	Slug        string
	Name        string
	Description string
	Icon        string
	IsDefault   bool
}

// SystemViews defines the system built in views for notifications.
var SystemViews = map[string]SystemViewDefinition{
	SystemViewInbox: {
		Slug:        "inbox",
		Name:        "Inbox",
		Description: "All notifications that need attention",
		Icon:        "inbox",
		IsDefault:   true,
	},
	SystemViewEverything: {
		Slug:        "everything",
		Name:        "Everything",
		Description: "All notifications including done",
		Icon:        "infinity",
		IsDefault:   false,
	},
	// Starred view
	SystemViewStarred: {
		Slug:        "starred",
		Name:        "Starred",
		Description: "Starred notifications",
		Icon:        "star",
		IsDefault:   false,
	},
	// Archive view
	SystemViewArchive: {
		Slug:        "archive",
		Name:        "Archive",
		Description: "Archived notifications",
		Icon:        "archive",
		IsDefault:   false,
	},
	// Snoozed view
	SystemViewSnoozed: {
		Slug:        "snoozed",
		Name:        "Snoozed",
		Description: "Snoozed notifications",
		Icon:        "snooze",
		IsDefault:   false,
	},
}

// IsSystemView checks if a slug is a system view
func IsSystemView(slug string) bool {
	_, exists := SystemViews[slug]
	return exists
}
