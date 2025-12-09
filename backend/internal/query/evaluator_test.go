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
	"testing"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

func TestEvaluatorBasic(t *testing.T) {
	notif := &db.Notification{
		Archived: false,
		IsRead:   false,
		Muted:    false,
	}

	repo := &db.Repository{
		FullName: "owner/repo",
	}

	// Test empty query
	eval, err := NewEvaluator("")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if !eval.Matches(notif, repo) {
		t.Error("Empty query should match everything")
	}

	// Test is:unread
	eval, err = NewEvaluator("is:unread")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if !eval.Matches(notif, repo) {
		t.Error("Notification should match is:unread")
	}

	// Test is:read (should not match)
	eval, err = NewEvaluator("is:read")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if eval.Matches(notif, repo) {
		t.Error("Notification should not match is:read")
	}
}

func TestActionHintsBasic(t *testing.T) {
	notif := &db.Notification{
		Archived: false,
		IsRead:   false,
		Muted:    false,
	}

	repo := &db.Repository{
		FullName: "owner/repo",
	}

	// Test inbox query - archiving should dismiss
	hints := ComputeActionHints(notif, repo, "")
	if len(hints.DismissedOn) == 0 {
		t.Error("Expected some dismissal actions for inbox query")
	}

	var foundArchive bool
	for _, action := range hints.DismissedOn {
		if action == "archive" {
			foundArchive = true
		}
	}
	if !foundArchive {
		t.Error("Expected 'archive' to be in dismissedOn for inbox query")
	}
}

func TestFreeTextSearch(t *testing.T) {
	notif := &db.Notification{
		SubjectTitle: "Fix bug in authentication",
		SubjectType:  "PullRequest",
	}

	repo := &db.Repository{
		FullName: "myorg/myrepo",
	}

	// Test matching title
	eval, err := NewEvaluator("authentication")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if !eval.Matches(notif, repo) {
		t.Error("Should match free text in title")
	}

	// Test matching repo name
	eval, err = NewEvaluator("myorg")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if !eval.Matches(notif, repo) {
		t.Error("Should match free text in repo name")
	}

	// Test not matching
	eval, err = NewEvaluator("something-else")
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if eval.Matches(notif, repo) {
		t.Error("Should not match unrelated free text")
	}
}
