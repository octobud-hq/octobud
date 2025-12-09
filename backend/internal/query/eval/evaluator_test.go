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

package eval

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/query/parse"
)

func TestNewEvaluator(t *testing.T) {
	// Test with nil AST
	eval := NewEvaluator(nil)
	if eval == nil {
		t.Fatal("NewEvaluator(nil) should not return nil")
	}

	// Test with AST
	term := &parse.Term{Field: "is", Values: []string{"unread"}}
	eval = NewEvaluator(term)
	if eval == nil {
		t.Fatal("NewEvaluator(term) should not return nil")
	}
}

func TestEvaluator_Matches_EmptyAST(t *testing.T) {
	notif := &db.Notification{}
	repo := &db.Repository{FullName: "owner/repo"}

	eval := NewEvaluator(nil)
	if !eval.Matches(notif, repo) {
		t.Error("Empty AST should match everything")
	}
}

func TestEvaluator_Matches_Term(t *testing.T) {
	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		term     *parse.Term
		expected bool
	}{
		{
			name:     "matches is:unread for unread notification",
			notif:    &db.Notification{IsRead: false},
			repo:     &db.Repository{FullName: "owner/repo"},
			term:     &parse.Term{Field: "is", Values: []string{"unread"}},
			expected: true,
		},
		{
			name:     "does not match is:unread for read notification",
			notif:    &db.Notification{IsRead: true},
			repo:     &db.Repository{FullName: "owner/repo"},
			term:     &parse.Term{Field: "is", Values: []string{"unread"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewEvaluator(tt.term)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_Matches_BinaryExpr(t *testing.T) {
	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		expr     *parse.BinaryExpr
		expected bool
	}{
		{
			name:  "AND matches when both conditions are true",
			notif: &db.Notification{IsRead: false, Muted: false},
			repo:  &db.Repository{FullName: "owner/repo"},

			expr: &parse.BinaryExpr{
				Op:    "AND",
				Left:  &parse.Term{Field: "is", Values: []string{"unread"}},
				Right: &parse.Term{Field: "is", Values: []string{"unmuted"}},
			},
			expected: true,
		},
		{
			name:  "AND does not match when one condition is false",
			notif: &db.Notification{IsRead: false, Muted: true},
			repo:  &db.Repository{FullName: "owner/repo"},

			expr: &parse.BinaryExpr{
				Op:    "AND",
				Left:  &parse.Term{Field: "is", Values: []string{"unread"}},
				Right: &parse.Term{Field: "is", Values: []string{"unmuted"}},
			},
			expected: false,
		},
		{
			name:  "OR matches when at least one condition is true",
			notif: &db.Notification{IsRead: true, Muted: false},
			repo:  &db.Repository{FullName: "owner/repo"},

			expr: &parse.BinaryExpr{
				Op:    "OR",
				Left:  &parse.Term{Field: "is", Values: []string{"read"}},
				Right: &parse.Term{Field: "is", Values: []string{"unmuted"}},
			},
			expected: true,
		},
		{
			name:  "OR does not match when both conditions are false",
			notif: &db.Notification{IsRead: false, Muted: true},
			repo:  &db.Repository{FullName: "owner/repo"},

			expr: &parse.BinaryExpr{
				Op:    "OR",
				Left:  &parse.Term{Field: "is", Values: []string{"read"}},
				Right: &parse.Term{Field: "is", Values: []string{"unmuted"}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewEvaluator(tt.expr)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_Matches_NotExpr(t *testing.T) {
	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		expr     *parse.NotExpr
		expected bool
	}{
		{
			name:     "matches NOT when inner expression is false",
			notif:    &db.Notification{Archived: false},
			repo:     &db.Repository{FullName: "owner/repo"},
			expr:     &parse.NotExpr{Expr: &parse.Term{Field: "is", Values: []string{"archived"}}},
			expected: true,
		},
		{
			name:     "does not match NOT when inner expression is true",
			notif:    &db.Notification{Archived: true},
			repo:     &db.Repository{FullName: "owner/repo"},
			expr:     &parse.NotExpr{Expr: &parse.Term{Field: "is", Values: []string{"archived"}}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewEvaluator(tt.expr)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_Matches_ParenExpr(t *testing.T) {
	notif := &db.Notification{
		IsRead: false,
	}
	repo := &db.Repository{FullName: "owner/repo"}

	term := &parse.Term{Field: "is", Values: []string{"unread"}}
	paren := &parse.ParenExpr{Expr: term}
	eval := NewEvaluator(paren)

	if !eval.Matches(notif, repo) {
		t.Error("Should match parenthesized expression")
	}
}

func TestEvaluator_Matches_NegatedTerm(t *testing.T) {
	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		term     *parse.Term
		expected bool
	}{
		{
			name:     "matches negated term when field value is false",
			notif:    &db.Notification{Archived: false},
			repo:     &db.Repository{FullName: "owner/repo"},
			term:     &parse.Term{Field: "is", Values: []string{"archived"}, Negated: true},
			expected: true,
		},
		{
			name:     "does not match negated term when field value is true",
			notif:    &db.Notification{Archived: true},
			repo:     &db.Repository{FullName: "owner/repo"},
			term:     &parse.Term{Field: "is", Values: []string{"archived"}, Negated: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewEvaluator(tt.term)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_Matches_TermMultipleValues(t *testing.T) {
	notif := &db.Notification{
		IsRead: false,
	}
	repo := &db.Repository{FullName: "owner/repo"}

	// Term with multiple values (OR logic)
	term := &parse.Term{Field: "is", Values: []string{"read", "unread"}}
	eval := NewEvaluator(term)

	if !eval.Matches(notif, repo) {
		t.Error("Should match when any value matches (OR logic)")
	}
}

func TestEvaluator_evaluateIsCondition(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		notif    *db.Notification
		value    string
		expected bool
	}{
		{"read", &db.Notification{IsRead: true}, "read", true},
		{"unread", &db.Notification{IsRead: false}, "unread", true},
		{"archived", &db.Notification{Archived: true}, "archived", true},
		{"inbox", &db.Notification{Archived: false}, "inbox", true},
		{"muted", &db.Notification{Muted: true}, "muted", true},
		{"unmuted", &db.Notification{Muted: false}, "unmuted", true},
		{"starred", &db.Notification{Starred: true}, "starred", true},
		{"unstarred", &db.Notification{Starred: false}, "unstarred", true},
		{
			"snoozed",
			&db.Notification{SnoozedUntil: sql.NullTime{Valid: true, Time: now.Add(1 * time.Hour)}},
			"snoozed",
			true,
		},
		{
			"unsnoozed",
			&db.Notification{SnoozedUntil: sql.NullTime{Valid: false}},
			"unsnoozed",
			true,
		},
		{"active", &db.Notification{SnoozedUntil: sql.NullTime{Valid: false}}, "active", true},
		{"filtered", &db.Notification{Filtered: true}, "filtered", true},
		{"unknown", &db.Notification{}, "unknown", true}, // Unknown values default to true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := &parse.Term{Field: "is", Values: []string{tt.value}}
			eval := NewEvaluator(term)
			result := eval.Matches(tt.notif, nil)
			if result != tt.expected {
				t.Errorf("evaluateIsCondition(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestEvaluator_evaluateInCondition(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		notif    *db.Notification
		value    string
		expected bool
	}{
		{
			"inbox - matches",
			&db.Notification{
				Archived:     false,
				Muted:        false,
				Filtered:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			"inbox",
			true,
		},
		{
			"inbox - archived",
			&db.Notification{Archived: true, Muted: false, Filtered: false},
			"inbox",
			false,
		},
		{
			"inbox - muted",
			&db.Notification{Archived: false, Muted: true, Filtered: false},
			"inbox",
			false,
		},
		{
			"inbox - filtered",
			&db.Notification{Archived: false, Muted: false, Filtered: true},
			"inbox",
			false,
		},
		{
			"archive - matches",
			&db.Notification{Archived: true, Muted: false},
			"archive",
			true,
		},
		{
			"archive - not archived",
			&db.Notification{Archived: false, Muted: false},
			"archive",
			false,
		},
		{
			"archive - muted",
			&db.Notification{Archived: true, Muted: true},
			"archive",
			false,
		},
		{
			"snoozed - matches",

			&db.Notification{
				SnoozedUntil: sql.NullTime{Valid: true, Time: now.Add(1 * time.Hour)},
				Archived:     false,
				Muted:        false,
			},
			"snoozed",
			true,
		},
		{
			"snoozed - not snoozed",
			&db.Notification{
				SnoozedUntil: sql.NullTime{Valid: false},
				Archived:     false,
				Muted:        false,
			},
			"snoozed",
			false,
		},
		{
			"filtered - matches",
			&db.Notification{
				Filtered:     true,
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			"filtered",
			true,
		},
		{
			"filtered - archived",
			&db.Notification{
				Filtered:     true,
				Archived:     true,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			"filtered",
			false,
		},
		{
			"filtered - muted",
			&db.Notification{
				Filtered:     true,
				Archived:     false,
				Muted:        true,
				SnoozedUntil: sql.NullTime{Valid: false},
			},
			"filtered",
			false,
		},
		{
			"filtered - snoozed",
			&db.Notification{
				Filtered:     true,
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: true, Time: now.Add(1 * time.Hour)},
			},
			"filtered",
			false,
		},
		{
			"anywhere - always matches",
			&db.Notification{Archived: true, Muted: true, Filtered: true},
			"anywhere",
			true,
		},
		{
			"unknown - defaults to true",
			&db.Notification{},
			"unknown",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := &parse.Term{Field: "in", Values: []string{tt.value}}
			eval := NewEvaluator(term)
			result := eval.Matches(tt.notif, nil)
			if result != tt.expected {
				t.Errorf("evaluateInCondition(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestEvaluator_evaluateFieldValue(t *testing.T) {
	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		term     *parse.Term
		expected bool
	}{
		// Repo field tests
		{
			name:     "repo matches (case-insensitive substring)",
			notif:    &db.Notification{},
			repo:     &db.Repository{FullName: "owner/repo-name"},
			term:     &parse.Term{Field: "repo", Values: []string{"repo-name"}},
			expected: true,
		},
		{
			name:     "repo does not match different repo",
			notif:    &db.Notification{},
			repo:     &db.Repository{FullName: "owner/repo-name"},
			term:     &parse.Term{Field: "repo", Values: []string{"other"}},
			expected: false,
		},
		{
			name:     "repo does not match when repo is nil",
			notif:    &db.Notification{},
			repo:     nil,
			term:     &parse.Term{Field: "repo", Values: []string{"repo-name"}},
			expected: false,
		},
		// Author field tests
		{
			name: "author matches",
			notif: &db.Notification{
				AuthorLogin: sql.NullString{Valid: true, String: "username"},
			},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "author", Values: []string{"username"}},
			expected: true,
		},
		{
			name: "author does not match different author",
			notif: &db.Notification{
				AuthorLogin: sql.NullString{Valid: true, String: "username"},
			},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "author", Values: []string{"other"}},
			expected: false,
		},
		{
			name:     "author does not match when author is invalid",
			notif:    &db.Notification{AuthorLogin: sql.NullString{Valid: false}},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "author", Values: []string{"username"}},
			expected: false,
		},
		// Reason field tests
		{
			name: "reason matches",
			notif: &db.Notification{
				Reason: sql.NullString{Valid: true, String: "review_requested"},
			},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "reason", Values: []string{"review_requested"}},
			expected: true,
		},
		{
			name: "reason does not match different reason",
			notif: &db.Notification{
				Reason: sql.NullString{Valid: true, String: "review_requested"},
			},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "reason", Values: []string{"mention"}},
			expected: false,
		},
		{
			name:     "reason does not match when reason is invalid",
			notif:    &db.Notification{Reason: sql.NullString{Valid: false}},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "reason", Values: []string{"review_requested"}},
			expected: false,
		},
		// Type field tests
		{
			name:     "type matches (case-insensitive)",
			notif:    &db.Notification{SubjectType: "PullRequest"},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "type", Values: []string{"PullRequest"}},
			expected: true,
		},
		{
			name:     "type matches case-insensitively",
			notif:    &db.Notification{SubjectType: "PullRequest"},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "type", Values: []string{"pullrequest"}},
			expected: true,
		},
		{
			name:     "type does not match different type",
			notif:    &db.Notification{SubjectType: "PullRequest"},
			repo:     &db.Repository{},
			term:     &parse.Term{Field: "type", Values: []string{"Issue"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewEvaluator(tt.term)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_evaluateFieldValue_UnknownField(t *testing.T) {
	notif := &db.Notification{}
	repo := &db.Repository{}

	// Unknown fields should not filter (return true)
	term := &parse.Term{Field: "unknown", Values: []string{"value"}}
	eval := NewEvaluator(term)

	if !eval.Matches(notif, repo) {
		t.Error("Unknown fields should not filter (default to true)")
	}
}

func TestEvaluator_evaluateFreeText(t *testing.T) {
	stateJSON, err := json.Marshal(map[string]interface{}{"state": "open"})
	if err != nil {
		t.Fatalf("failed to marshal stateJSON in test setup: %v", err)
	}
	baseNotif := &db.Notification{
		SubjectTitle: "Fix authentication bug",
		SubjectType:  "PullRequest",
		AuthorLogin:  sql.NullString{Valid: true, String: "developer"},
		SubjectState: sql.NullString{Valid: true, String: "open"},
		SubjectRaw:   db.NullRawMessage{Valid: true, RawMessage: stateJSON},
	}
	repo := &db.Repository{FullName: "owner/repo-name"}

	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		text     string
		expected bool
	}{
		{
			name:     "matches free text in subject title",
			notif:    baseNotif,
			repo:     repo,
			text:     "authentication",
			expected: true,
		},
		{
			name:     "matches free text in subject type (case-insensitive)",
			notif:    baseNotif,
			repo:     repo,
			text:     "pullrequest",
			expected: true,
		},
		{
			name:     "matches free text in repository name",
			notif:    baseNotif,
			repo:     repo,
			text:     "repo-name",
			expected: true,
		},
		{
			name:     "matches free text in author login",
			notif:    baseNotif,
			repo:     repo,
			text:     "developer",
			expected: true,
		},
		{
			name:     "matches free text in subject_state column",
			notif:    baseNotif,
			repo:     repo,
			text:     "open",
			expected: true,
		},
		{
			name: "matches free text in subject_raw fallback when subject_state not available",
			notif: &db.Notification{
				SubjectTitle: "Fix bug",
				SubjectType:  "Issue",
				SubjectState: sql.NullString{Valid: false}, // Not set
				SubjectRaw:   db.NullRawMessage{Valid: true, RawMessage: stateJSON},
			},
			repo:     repo,
			text:     "open",
			expected: true,
		},
		{
			name:     "does not match unrelated free text",
			notif:    baseNotif,
			repo:     repo,
			text:     "nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freeText := &parse.FreeText{Text: tt.text}
			eval := NewEvaluator(freeText)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_Matches_Unified(t *testing.T) {
	tests := []struct {
		name     string
		notif    *db.Notification
		repo     *db.Repository
		ast      parse.Node
		expected bool
	}{
		{
			name: "empty query matches when inbox defaults pass",
			notif: &db.Notification{
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{},
			ast:      nil,
			expected: true,
		},
		{
			name: "empty query does not match when filtered (inbox default)",
			notif: &db.Notification{
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     true,
			},
			repo:     &db.Repository{},
			ast:      nil,
			expected: false,
		},
		{
			name: "empty query does not match when archived (inbox default)",
			notif: &db.Notification{
				Archived:     true,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{},
			ast:      nil,
			expected: false,
		},
		{
			name:     "in:anywhere skips all defaults",
			notif:    &db.Notification{Archived: true, Muted: true, Filtered: true},
			repo:     &db.Repository{},
			ast:      &parse.Term{Field: "in", Values: []string{"anywhere"}},
			expected: true,
		},
		{
			name: "non-empty query without in: excludes muted only - allows filtered",
			notif: &db.Notification{
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     true,
			},
			repo:     &db.Repository{FullName: "owner/test"},
			ast:      &parse.Term{Field: "repo", Values: []string{"test"}},
			expected: true,
		},
		{
			name: "non-empty query without in: excludes muted only - rejects muted",
			notif: &db.Notification{
				Archived:     false,
				Muted:        true,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{},
			ast:      &parse.Term{Field: "repo", Values: []string{"test"}},
			expected: false,
		},
		{
			name: "non-empty query without in: excludes muted only - allows archived",
			notif: &db.Notification{
				Archived:     true,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{FullName: "owner/test"},
			ast:      &parse.Term{Field: "repo", Values: []string{"test"}},
			expected: true,
		},
		{
			name: "in:inbox provides its own filters - no additional defaults",
			notif: &db.Notification{
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{FullName: "owner/test"},
			ast:      &parse.Term{Field: "in", Values: []string{"inbox"}},
			expected: true,
		},
		{
			name: "in:inbox rejects archived - no additional defaults",
			notif: &db.Notification{
				Archived:     true,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{},
			ast:      &parse.Term{Field: "in", Values: []string{"inbox"}},
			expected: false,
		},
		{
			name: "in:inbox rejects filtered - no additional defaults",
			notif: &db.Notification{
				Archived:     false,
				Muted:        false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     true,
			},
			repo:     &db.Repository{},
			ast:      &parse.Term{Field: "in", Values: []string{"inbox"}},
			expected: false,
		},
		{
			name: "is:muted explicitly asks for muted - should match muted notification",
			notif: &db.Notification{
				Muted:        true,
				Archived:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{},
			ast:      &parse.Term{Field: "is", Values: []string{"muted"}},
			expected: true,
		},
		{
			name: "is:muted should not match unmuted notification",
			notif: &db.Notification{
				Muted:        false,
				Archived:     false,
				SnoozedUntil: sql.NullTime{Valid: false},
				Filtered:     false,
			},
			repo:     &db.Repository{},
			ast:      &parse.Term{Field: "is", Values: []string{"muted"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewEvaluator(tt.ast)
			result := eval.Matches(tt.notif, tt.repo)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluator_ComplexNestedExpression(t *testing.T) {
	notif := &db.Notification{
		IsRead: false,
		Muted:  false,
	}
	repo := &db.Repository{FullName: "owner/repo"}

	// ((is:unread OR is:read) AND is:unmuted)
	left := &parse.BinaryExpr{
		Op:    "OR",
		Left:  &parse.Term{Field: "is", Values: []string{"unread"}},
		Right: &parse.Term{Field: "is", Values: []string{"read"}},
	}
	right := &parse.Term{Field: "is", Values: []string{"unmuted"}}
	inner := &parse.BinaryExpr{Op: "AND", Left: left, Right: right}
	outer := &parse.ParenExpr{Expr: inner}
	eval := NewEvaluator(outer)

	if !eval.Matches(notif, repo) {
		t.Error("Should match complex nested expression")
	}

	notif.Muted = true
	if eval.Matches(notif, repo) {
		t.Error("Should not match when muted")
	}
}
