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
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query/eval"
)

// ComputeActionHints determines which actions would dismiss the notification from the current query
// Uses unified query semantics that apply business rules based on query content
func ComputeActionHints(
	notif *db.Notification,
	repo *db.Repository,
	queryStr string,
) *models.ActionHints {
	evaluator, err := NewEvaluator(queryStr)
	if err != nil {
		// If query parsing fails, be conservative
		return &models.ActionHints{DismissedOn: []string{}}
	}

	return ComputeActionHintsWithEvaluator(notif, repo, evaluator)
}

// ComputeActionHintsWithEvaluator determines which actions would dismiss the notification
// using a pre-created evaluator. This is more efficient when computing hints for multiple
// notifications with the same query.
func ComputeActionHintsWithEvaluator(
	notif *db.Notification,
	repo *db.Repository,
	evaluator *eval.Evaluator,
) *models.ActionHints {
	dismissedOn := []string{}

	// Test ALL possible dismissive actions, not just the opposite of current state

	// Archive - test if archiving would dismiss
	if !notif.Archived && wouldDismissOnAction(notif, repo, evaluator, "archive") {
		dismissedOn = append(dismissedOn, "archive")
	}

	// Unarchive - test if unarchiving would dismiss
	if notif.Archived && wouldDismissOnAction(notif, repo, evaluator, "unarchive") {
		dismissedOn = append(dismissedOn, "unarchive")
	}

	// Mute - test if muting would dismiss
	if !notif.Muted && wouldDismissOnAction(notif, repo, evaluator, "mute") {
		dismissedOn = append(dismissedOn, "mute")
	}

	// Unmute - test if unmuting would dismiss
	if notif.Muted && wouldDismissOnAction(notif, repo, evaluator, "unmute") {
		dismissedOn = append(dismissedOn, "unmute")
	}

	// Snooze - test if snoozing would dismiss (only if not currently snoozed)
	if !notif.SnoozedUntil.Valid || notif.SnoozedUntil.Time.Before(time.Now()) {
		if wouldDismissOnAction(notif, repo, evaluator, "snooze") {
			dismissedOn = append(dismissedOn, "snooze")
		}
	}

	// Unsnooze - test if unsnoozing would dismiss (only if currently snoozed)
	if notif.SnoozedUntil.Valid && notif.SnoozedUntil.Time.After(time.Now()) {
		if wouldDismissOnAction(notif, repo, evaluator, "unsnooze") {
			dismissedOn = append(dismissedOn, "unsnooze")
		}
	}

	// Filter - test if filtering would dismiss (only if not currently filtered)
	if !notif.Filtered {
		if wouldDismissOnAction(notif, repo, evaluator, "filter") {
			dismissedOn = append(dismissedOn, "filter")
		}
	}

	// Unfilter - test if unfiltering would dismiss (only if currently filtered)
	if notif.Filtered {
		if wouldDismissOnAction(notif, repo, evaluator, "unfilter") {
			dismissedOn = append(dismissedOn, "unfilter")
		}
	}

	// Read/Unread: NEVER dismiss (Gmail UX - only refresh dismisses)
	// Star/Unstar: NEVER dismiss (per UX decision)

	return &models.ActionHints{DismissedOn: dismissedOn}
}

// wouldDismissOnAction tests if performing an action would dismiss the notification
func wouldDismissOnAction(
	notif *db.Notification,
	repo *db.Repository,
	evaluator *eval.Evaluator,
	action string,
) bool {
	clone := *notif

	// Apply the action to the clone
	switch action {
	case "archive":
		clone.Archived = true
	case "unarchive":
		clone.Archived = false
	case "mute":
		clone.Muted = true
	case "unmute":
		clone.Muted = false
	case "snooze":
		// Simulate snoozing (set to future time)
		clone.SnoozedUntil.Valid = true
		clone.SnoozedUntil.Time = time.Now().Add(24 * time.Hour)
	case "unsnooze":
		clone.SnoozedUntil.Valid = false
	case "filter":
		clone.Filtered = true
	case "unfilter":
		clone.Filtered = false
	default:
		return false
	}

	// Check if the modified notification still matches the query using unified Matches
	return !evaluator.Matches(&clone, repo)
}
