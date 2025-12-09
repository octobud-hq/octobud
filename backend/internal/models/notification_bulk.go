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

// BulkOperationType represents the type of bulk operation to perform
type BulkOperationType string

// BulkOperationType constants
const (
	BulkOpMarkRead   BulkOperationType = "mark-read"
	BulkOpMarkUnread BulkOperationType = "mark-unread"
	BulkOpArchive    BulkOperationType = "archive"
	BulkOpUnarchive  BulkOperationType = "unarchive"
	BulkOpMute       BulkOperationType = "mute"
	BulkOpUnmute     BulkOperationType = "unmute"
	BulkOpStar       BulkOperationType = "star"
	BulkOpUnstar     BulkOperationType = "unstar"
	BulkOpUnfilter   BulkOperationType = "unfilter"
	BulkOpSnooze     BulkOperationType = "snooze"
	BulkOpUnsnooze   BulkOperationType = "unsnooze"
)

// BulkOperationTarget specifies how to target notifications for bulk operations
type BulkOperationTarget struct {
	// IDs is a list of GitHub notification IDs. Mutually exclusive with Query.
	IDs []string
	// Query is a query string to find notifications. Mutually exclusive with IDs.
	Query string
}

// BulkUpdateParams holds optional parameters for bulk operations
type BulkUpdateParams struct {
	// SnoozedUntil is required for snooze operations (RFC3339 format)
	SnoozedUntil string
}
