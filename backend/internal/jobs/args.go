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

// Package jobs provides the job arguments.
package jobs

import (
	"encoding/json"
	"time"
)

// ApplyRuleArgs contains arguments for applying a rule to notifications.
type ApplyRuleArgs struct {
	// UserID is the ID of the user whose notifications to apply the rule to
	UserID int64 `json:"user_id"`
	// RuleID is the ID of the rule to apply
	RuleID int64 `json:"rule_id"`
}

// ProcessNotificationArgs contains arguments for processing a single notification.
type ProcessNotificationArgs struct {
	// UserID is the ID of the user whose notification is being processed
	UserID int64 `json:"user_id"`
	// NotificationData is the raw notification JSON from GitHub
	NotificationData json.RawMessage `json:"notification_data"`
}

// SyncNotificationsArgs contains arguments for syncing notifications.
type SyncNotificationsArgs struct {
	// UserID is the ID of the user to sync notifications for
	UserID int64 `json:"user_id"`
}

// SyncOlderNotificationsArgs contains arguments for syncing older notifications.
type SyncOlderNotificationsArgs struct {
	// UserID is the ID of the user to sync notifications for
	UserID int64 `json:"user_id"`
	// Days is the number of days to sync back from UntilTime
	Days int `json:"days"`
	// UntilTime is the cutoff - only sync notifications older than this
	UntilTime time.Time `json:"untilTime"`
	// MaxCount is an optional limit on the number of notifications to sync
	MaxCount *int `json:"maxCount,omitempty"`
	// UnreadOnly filters to only sync unread notifications
	UnreadOnly bool `json:"unreadOnly"`
}
