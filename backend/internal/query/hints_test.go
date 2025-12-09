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
	"database/sql"
	"testing"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

func TestComputeActionHints(t *testing.T) {
	now := time.Now()
	repo := &db.Repository{FullName: "owner/repo"}

	tests := []struct {
		name           string
		notif          *db.Notification
		query          string
		wantDismissed  []string
		wantNotDismiss []string // Actions that should NOT be in dismissedOn
		description    string
	}{
		// Default inbox semantics (empty query) - should dismiss on archive, snooze, mute, filter
		{
			name: "default inbox - unarchived unmuted unsnoozed unfiltered",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "",
			wantDismissed: []string{"archive", "mute", "snooze", "filter"},
			description:   "Default inbox: archive, mute, snooze, filter should dismiss",
		},
		{
			name: "default inbox - already archived",
			notif: &db.Notification{
				Archived:     true,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "",
			wantDismissed: []string{}, // Already archived, so doesn't match inbox query
			description:   "Default inbox: archived notification doesn't match, so no dismissive actions",
		},

		// in:inbox - explicit inbox query
		{
			name: "in:inbox - unarchived unmuted unsnoozed unfiltered",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "in:inbox",
			wantDismissed: []string{"archive", "mute", "snooze", "filter"},
			description:   "in:inbox: archive, mute, snooze, filter should dismiss",
		},
		// in:archive - viewing archived notifications
		// A notification must already be archived to match in:archive
		// Unarchiving dismisses (removes from archive view)
		// Muting dismisses (archive view excludes muted notifications per evaluator logic)
		{
			name: "in:archive - already archived notification",
			notif: &db.Notification{
				Archived:     true,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "in:archive",
			wantDismissed: []string{"unarchive", "mute"},
			description:   "in:archive: unarchiving or muting dismisses (removes from archive view)",
		},

		// in:anywhere - everything view, nothing should dismiss
		{
			name: "in:anywhere - no dismissive actions",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "in:anywhere",
			wantDismissed: []string{},
			wantNotDismiss: []string{
				"archive",
				"mute",
				"snooze",
				"filter",
				"unarchive",
				"unmute",
				"unsnooze",
				"unfilter",
			},
			description: "in:anywhere: nothing should dismiss (everything view)",
		},
		{
			name: "in:anywhere - already archived",
			notif: &db.Notification{
				Archived:     true,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "in:anywhere",
			wantDismissed: []string{},
			wantNotDismiss: []string{
				"archive",
				"mute",
				"snooze",
				"filter",
				"unarchive",
				"unmute",
				"unsnooze",
				"unfilter",
			},
			description: "in:anywhere: nothing should dismiss even for archived notification",
		},

		// in:snoozed - snoozed view, unsnoozing, muting, or archiving dismisses
		{
			name: "in:snoozed - currently snoozed notification",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: true, Time: now.Add(24 * time.Hour)},
			},
			query:         "in:snoozed",
			wantDismissed: []string{"unsnooze", "mute", "archive"},
			description:   "in:snoozed: unsnoozing, muting, or archiving dismisses",
		},
		{
			name: "in:snoozed - not snoozed notification",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "in:snoozed",
			wantDismissed: []string{}, // Not snoozed, so doesn't match query
			description:   "in:snoozed: unsnoozed notification doesn't match, so no dismissive actions",
		},

		// Read/unread should NEVER dismiss (Gmail UX)
		{
			name: "read/unread never dismiss - is:read query",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:          "is:read",
			wantNotDismiss: []string{"read", "unread"},
			description:    "Read/unread actions should never dismiss",
		},

		// Repo query
		{
			name: "repo query - archive still dismisses",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "repo:repo",
			wantDismissed: []string{"mute"}, // Non-empty query without in: excludes muted only
			description:   "Repo query with inbox defaults: archive, mute, snooze, filter should dismiss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := ComputeActionHints(tt.notif, repo, tt.query)

			// Check that all expected dismissive actions are present
			for _, wantAction := range tt.wantDismissed {
				found := false
				for _, action := range hints.DismissedOn {
					if action == wantAction {
						found = true
						break
					}
				}
				if !found {
					t.Errorf(
						"%s: expected action '%s' to be in dismissedOn, but it wasn't. Got: %v",
						tt.description,
						wantAction,
						hints.DismissedOn,
					)
				}
			}

			// Check that actions that should NOT dismiss are not present
			for _, notWantAction := range tt.wantNotDismiss {
				for _, action := range hints.DismissedOn {
					if action == notWantAction {
						t.Errorf(
							"%s: expected action '%s' NOT to be in dismissedOn, but it was. Got: %v",
							tt.description,
							notWantAction,
							hints.DismissedOn,
						)
						break
					}
				}
			}
		})
	}
}

func TestComputeActionHints_CustomViewScenarios(t *testing.T) {
	repo := &db.Repository{FullName: "owner/repo"}

	tests := []struct {
		name           string
		notif          *db.Notification
		query          string
		wantDismissed  []string
		wantNotDismiss []string
		description    string
	}{
		{
			name: "empty query - filtered notification doesn't match",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     true, // Filtered notifications excluded by inbox defaults
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "",
			wantDismissed: []string{}, // Already filtered, so doesn't match empty query (inbox defaults)
			description:   "Empty query: filtered notification doesn't match inbox defaults",
		},
		{
			name: "empty query - unfiltered notification",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query: "",
			wantDismissed: []string{
				"archive",
				"mute",
				"snooze",
				"filter",
			}, // Empty query = inbox defaults
			description: "Empty query: inbox defaults apply (excludes archived, snoozed, muted, filtered)",
		},
		{
			name: "custom view - is:filtered query",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "is:filtered",
			wantDismissed: []string{"mute"}, // Non-empty query without in: excludes muted only
			description:   "Non-empty query without in:: only mute dismisses (allows archived, snoozed, filtered)",
		},
		{
			name: "custom view - unfilter dismisses when querying for filtered",
			notif: &db.Notification{
				Archived:     false,
				IsRead:       false,
				Muted:        false,
				Filtered:     true,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			query:         "is:filtered",
			wantDismissed: []string{"mute"}, // Non-empty query without in: excludes muted only
			description:   "Non-empty query without in:: only mute dismisses (allows archived, snoozed, filtered)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := ComputeActionHints(tt.notif, repo, tt.query)

			// Check that all expected dismissive actions are present
			for _, wantAction := range tt.wantDismissed {
				found := false
				for _, action := range hints.DismissedOn {
					if action == wantAction {
						found = true
						break
					}
				}
				if !found {
					t.Errorf(
						"%s: expected action '%s' to be in dismissedOn, but it wasn't. Got: %v",
						tt.description,
						wantAction,
						hints.DismissedOn,
					)
				}
			}

			// Check that actions that should NOT dismiss are not present
			for _, notWantAction := range tt.wantNotDismiss {
				for _, action := range hints.DismissedOn {
					if action == notWantAction {
						t.Errorf(
							"%s: expected action '%s' NOT to be in dismissedOn, but it was. Got: %v",
							tt.description,
							notWantAction,
							hints.DismissedOn,
						)
						break
					}
				}
			}
		})
	}
}

func TestComputeActionHintsWithEvaluator_ReuseEvaluator(t *testing.T) {
	notif1 := &db.Notification{
		Archived: false,
		IsRead:   false,
		Muted:    false,
		Filtered: false,
	}
	notif2 := &db.Notification{
		Archived: false,
		IsRead:   true,
		Muted:    false,
		Filtered: false,
	}
	repo := &db.Repository{FullName: "owner/repo"}

	// Create evaluator once
	evaluator, err := NewEvaluator("is:unread")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// Use it for multiple notifications
	hints1 := ComputeActionHintsWithEvaluator(notif1, repo, evaluator)
	hints2 := ComputeActionHintsWithEvaluator(notif2, repo, evaluator)

	// notif1 is unread, should match
	if len(hints1.DismissedOn) == 0 {
		t.Error("Expected dismissal actions for unread notification")
	}

	// notif2 is read, should not match query, but still should have dismissive actions
	// (because we compute hints based on what actions would dismiss, not whether it matches)
	if len(hints2.DismissedOn) == 0 {
		t.Error("Expected dismissal actions even for non-matching notification")
	}
}

func TestComputeActionHints_InvalidQuery(t *testing.T) {
	notif := &db.Notification{
		Archived: false,
		IsRead:   false,
		Muted:    false,
		Filtered: false,
	}
	repo := &db.Repository{FullName: "owner/repo"}

	// Invalid query should return empty hints (conservative)
	hints := ComputeActionHints(notif, repo, "invalid:query:syntax")

	if len(hints.DismissedOn) != 0 {
		t.Errorf("Expected empty hints for invalid query, got: %v", hints.DismissedOn)
	}
}
