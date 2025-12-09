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
	"strings"
	"testing"

	"github.com/octobud-hq/octobud/backend/internal/query/parse"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestIntegration_EndToEnd(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantErr      bool
		wantContains []string // Strings that should appear in the WHERE clause
		wantJoins    int
	}{
		// Basic queries
		{
			name:         "simple repo filter",
			input:        "repo:cli",
			wantContains: []string{"r.full_name", "LIKE", "?"},
			wantJoins:    1,
		},
		{
			name:         "reason filter",
			input:        "reason:review_requested",
			wantContains: []string{"n.reason", "LIKE"},
			wantJoins:    0,
		},
		{
			name:         "is:unread",
			input:        "is:unread",
			wantContains: []string{"n.is_read = 0"},
			wantJoins:    0,
		},

		// Logical operators
		{
			name:         "AND operator",
			input:        "repo:cli AND is:unread",
			wantContains: []string{"r.full_name", "AND", "n.is_read = 0"},
			wantJoins:    1,
		},
		{
			name:         "OR operator",
			input:        "repo:cli OR repo:other",
			wantContains: []string{"r.full_name", "OR", "?"},
			wantJoins:    1,
		},
		{
			name:         "NOT operator",
			input:        "NOT author:bot",
			wantContains: []string{"NOT", "n.author_login"},
			wantJoins:    0,
		},

		// Grouping
		{
			name:         "grouped OR with AND",
			input:        "(repo:cli OR repo:other) AND is:unread",
			wantContains: []string{"r.full_name", "OR", "AND", "n.is_read = 0"},
			wantJoins:    1,
		},

		// in: operator
		{
			name:         "in:inbox",
			input:        "in:inbox",
			wantContains: []string{"n.archived = 0", "snoozed_until", "n.muted = 0"},
			wantJoins:    0,
		},
		{
			name:         "in:archive",
			input:        "in:archive",
			wantContains: []string{"n.archived = 1", "n.muted = 0"},
			wantJoins:    0,
		},
		{
			name:         "in:snoozed",
			input:        "in:snoozed",
			wantContains: []string{"n.snoozed_until IS NOT NULL", "strftime"},
			wantJoins:    0,
		},
		{
			name:         "in:anywhere",
			input:        "in:anywhere",
			wantContains: []string{"1"},
			wantJoins:    0,
		},

		// Comma-separated values (OR within field)
		{
			name:         "comma OR",
			input:        "repo:cli,other",
			wantContains: []string{"r.full_name", "OR", "?"},
			wantJoins:    1,
		},

		// Free text
		{
			name:         "free text search",
			input:        "urgent fix",
			wantContains: []string{"n.subject_title", "LIKE", "OR", "r.full_name"},
			wantJoins:    1,
		},

		// Complex real-world queries
		{
			name:         "complex: review requests from cli repo",
			input:        "repo:cli reason:review_requested is:unread",
			wantContains: []string{"r.full_name", "n.reason", "n.is_read = 0", "AND"},
			wantJoins:    1,
		},
		{
			name:         "complex: multiple repos with exclusion",
			input:        "(repo:cli OR repo:other) AND NOT author:bot",
			wantContains: []string{"r.full_name", "OR", "NOT", "n.author_login"},
			wantJoins:    1,
		},
		{
			name:         "complex: nested grouping",
			input:        "((repo:cli AND is:unread) OR (in:snoozed AND repo:other)) AND NOT author:bot",
			wantContains: []string{"r.full_name", "n.is_read", "snoozed_until", "OR", "AND", "NOT"},
			wantJoins:    1,
		},

		// Error cases
		{
			name:    "unknown field",
			input:   "unknown:value",
			wantErr: true,
		},
		{
			name:    "invalid in: value",
			input:   "in:invalid",
			wantErr: true,
		},
		{
			name:    "unbalanced parens",
			input:   "(repo:cli",
			wantErr: true,
		},
		{
			name:    "empty field",
			input:   ":value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.input, 50, 0)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check WHERE clause
			allWhere := ""
			for _, w := range query.Where {
				allWhere += w + " "
			}

			for _, want := range tt.wantContains {
				if !contains(allWhere, want) {
					t.Errorf("expected WHERE to contain %q\nGot: %s", want, allWhere)
				}
			}

			// Check joins
			if len(query.Joins) != tt.wantJoins {
				t.Errorf(
					"expected %d joins, got %d: %v",
					tt.wantJoins,
					len(query.Joins),
					query.Joins,
				)
			}

			// Check limit/offset
			if query.Limit != 50 {
				t.Errorf("expected limit 50, got %d", query.Limit)
			}
			if query.Offset != 0 {
				t.Errorf("expected offset 0, got %d", query.Offset)
			}
		})
	}
}

func TestIntegration_InboxQuery(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDefaults bool
	}{
		{
			name:         "empty query gets defaults",
			input:        "",
			wantDefaults: true,
		},
		{
			name:         "simple query gets muted-only defaults",
			input:        "repo:cli",
			wantDefaults: true, // Should have muted-only default (n.muted = 0)
		},
		{
			name:         "explicit in:inbox - provides its own filters",
			input:        "in:inbox",
			wantDefaults: true, // in:inbox provides the same filters, just not as "defaults"
		},
		{
			name:         "explicit in:anywhere - no defaults",
			input:        "in:anywhere repo:cli",
			wantDefaults: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.input, 50, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			allWhere := ""
			for _, w := range query.Where {
				allWhere += w + " "
			}

			// For empty query, check for inbox defaults (archived, snoozed, muted, filtered)
			// For non-empty query without in:, check for muted-only default
			// For in:inbox, it provides the same filters as inbox defaults (but not as "defaults")
			var hasDefaults bool
			//nolint:gocritic // ifElseChain: checking different string conditions, not suitable for switch
			if tt.input == "" {
				hasDefaults = contains(allWhere, "n.archived = 0") &&
					contains(allWhere, "n.muted = 0") &&
					contains(allWhere, "n.filtered = 0")
			} else if strings.Contains(tt.input, "in:inbox") {
				// in:inbox provides the same filters as inbox defaults
				hasDefaults = contains(allWhere, "n.archived = 0") &&
					contains(allWhere, "n.muted = 0") &&
					contains(allWhere, "n.filtered = 0")
			} else if !strings.Contains(tt.input, "in:") {
				hasDefaults = contains(allWhere, "n.muted = 0") &&
					!contains(allWhere, "n.archived = 0") // Should NOT have archived default
			} else {
				// Other in: operators (in:anywhere, in:archive, etc.) don't get defaults
				hasDefaults = false
			}

			if tt.wantDefaults != hasDefaults {
				t.Errorf(
					"expected defaults=%v, got=%v\nWHERE: %v",
					tt.wantDefaults,
					hasDefaults,
					query.Where,
				)
			}
		})
	}
}

func TestIntegration_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrMsg string
	}{
		{
			name:       "unknown field",
			input:      "badfield:value",
			wantErrMsg: "unknown field",
		},
		{
			name:       "invalid in: value",
			input:      "in:badvalue",
			wantErrMsg: "invalid value for in: operator",
		},
		{
			name:       "invalid is: value",
			input:      "is:badvalue",
			wantErrMsg: "invalid value for is: operator",
		},
		{
			name:       "invalid boolean",
			input:      "read:maybe",
			wantErrMsg: "invalid boolean value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildQuery(tt.input, 50, 0)
			if err == nil {
				t.Fatal("expected error, got none")
			}

			if !contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("expected error containing %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}

func TestIntegration_RealWorldQueries(t *testing.T) {
	// Test queries that represent real use cases
	queries := []string{
		// Simple filtering
		"repo:cli/cli is:unread",
		"reason:review_requested,mention",
		"type:PullRequest state:open",

		// View-based
		"in:inbox repo:cli",
		"in:snoozed",
		"in:archive",
		"in:anywhere repo:cli/cli",

		// Complex filtering
		"(repo:cli OR repo:other) AND is:unread",
		"repo:cli reason:review_requested,mention is:unread",
		"NOT (author:bot OR author:dependabot)",

		// With free text
		"repo:cli urgent",
		`"fix: memory leak" repo:cli`,

		// Very complex
		"((repo:cli/cli AND is:unread) OR (in:snoozed AND repo:github/docs)) AND NOT author:bot",
		"(reason:review_requested OR reason:mention) AND is:unread AND repo:cli,other",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			result, err := BuildQuery(query, 50, 0)
			if err != nil {
				t.Fatalf("failed to build query: %v", err)
			}

			// Just verify we can build it and it produces some output
			if len(result.Where) == 0 && query != "" {
				t.Error("expected non-empty WHERE clause")
			}
		})
	}
}
func TestIntegration_APIUsagePatterns(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "inbox with is:unread",
			input:   "is:unread",
			wantErr: false,
		},
		{
			name:    "everything view with in:anywhere",
			input:   "is:unread in:anywhere",
			wantErr: false,
		},
		{
			name:    "archive view",
			input:   "in:archive",
			wantErr: false,
		},
		{
			name:    "custom view with repo filter",
			input:   "repo:cli is:unread",
			wantErr: false,
		},
		{
			name:    "validation catches invalid queries",
			input:   "badfield:value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test ParseAndValidate (used in API validation)
			_, err := ParseAndValidate(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}

			// Test BuildQuery (used in everything/archive views)
			if !tt.wantErr {
				_, err := BuildQuery(tt.input, 50, 0)
				if err != nil {
					t.Errorf("BuildQuery failed: %v", err)
				}
			}

			// Test BuildQuery (unified query builder)
			if !tt.wantErr {
				_, err := BuildQuery(tt.input, 50, 0)
				if err != nil {
					t.Errorf("BuildQuery failed: %v", err)
				}
			}
		})
	}
}

// TestIntegration_InboxDefaults tests that inbox queries apply defaults correctly
func TestIntegration_InboxDefaults(t *testing.T) {
	// Empty query should get default filters
	q1, err := BuildQuery("", 50, 0)
	if err != nil {
		t.Fatalf("empty query failed: %v", err)
	}
	if len(q1.Where) < 3 {
		t.Errorf("expected at least 3 default filters, got %d", len(q1.Where))
	}

	// Query with in:anywhere should not get defaults
	q2, err := BuildQuery("in:anywhere", 50, 0)
	if err != nil {
		t.Fatalf("in:anywhere query failed: %v", err)
	}
	// Should only have one WHERE clause for the in:anywhere
	if len(q2.Where) != 1 {
		t.Errorf("in:anywhere should have 1 WHERE clause, got %d: %v", len(q2.Where), q2.Where)
	}
}

// TestIntegration_EverythingView tests the everything view pattern
func TestIntegration_EverythingView(t *testing.T) {
	// Everything view: is:unread in:anywhere
	query, err := BuildQuery("is:unread in:anywhere", 50, 0)
	if err != nil {
		t.Fatalf("everything view query failed: %v", err)
	}

	// Should have WHERE for is_read and in:anywhere (which generates TRUE)
	if len(query.Where) != 1 {
		t.Errorf("expected 1 WHERE clause, got %d: %v", len(query.Where), query.Where)
	}

	// The WHERE should contain both conditions
	where := query.Where[0]
	if !contains(where, "is_read") {
		t.Errorf("WHERE should contain is_read check: %s", where)
	}
}
func TestIntegration_SystemViews(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "inbox (default)",
			input:   "is:unread",
			wantErr: false,
		},
		{
			name:    "everything",
			input:   "is:unread in:anywhere",
			wantErr: false,
		},
		{
			name:    "archive",
			input:   "in:archive is:unread",
			wantErr: false,
		},
		{
			name:    "snoozed",
			input:   "in:snoozed is:unread",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with BuildQuery (no defaults)
			query1, err := BuildQuery(tt.input, 50, 0)
			if err != nil {
				t.Fatalf("BuildQuery failed: %v", err)
			}
			if len(query1.Where) == 0 {
				t.Error("expected WHERE clauses, got none")
			}

			// Test with BuildQuery (unified query builder)
			query2, err := BuildQuery(tt.input, 50, 0)
			if err != nil {
				t.Fatalf("BuildQuery failed: %v", err)
			}
			if len(query2.Where) == 0 {
				t.Error("expected WHERE clauses, got none")
			}
		})
	}
}

// TestIntegration_FullSQL tests complete end-to-end SQL generation with exact expectations
//

func TestIntegration_FullSQL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantWhere []string // Exact WHERE clauses expected
		wantArgs  []string // Expected argument values (as strings for comparison)
		wantJoins []string // Expected JOIN clauses
		wantErr   bool
	}{
		{
			name:      "simple repo filter",
			input:     "repo:cli",
			wantWhere: []string{"r.full_name LIKE ?"},
			wantArgs:  []string{"%cli%"},
			wantJoins: []string{"LEFT JOIN repositories r ON r.id = n.repository_id"},
		},
		{
			name:      "reason filter (no joins)",
			input:     "reason:review_requested",
			wantWhere: []string{"n.reason LIKE ?"},
			wantArgs:  []string{"%review_requested%"},
			wantJoins: []string{},
		},
		{
			name:      "is:unread",
			input:     "is:unread",
			wantWhere: []string{"n.is_read = 0"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "is:read",
			input:     "is:read",
			wantWhere: []string{"n.is_read = 1"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "read:true (boolean field)",
			input:     "read:true",
			wantWhere: []string{"n.is_read = 1"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "read:false (boolean field)",
			input:     "read:false",
			wantWhere: []string{"n.is_read = 0"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "archived:true",
			input:     "archived:true",
			wantWhere: []string{"n.archived = 1"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "archived:false",
			input:     "archived:false",
			wantWhere: []string{"n.archived = 0"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "muted:true",
			input:     "muted:true",
			wantWhere: []string{"n.muted = 1"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "is:muted",
			input:     "is:muted",
			wantWhere: []string{"n.muted = 1"}, // Should NOT have n.muted = 0 default
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "muted:false",
			input:     "muted:false",
			wantWhere: []string{"n.muted = 0"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:  "snoozed:true",
			input: "snoozed:true",
			wantWhere: []string{
				"(n.snoozed_until IS NOT NULL AND n.snoozed_until > strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))",
			},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:  "snoozed:false",
			input: "snoozed:false",
			wantWhere: []string{
				"(n.snoozed_until IS NULL OR n.snoozed_until <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))",
			},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "explicit AND",
			input:     "repo:cli AND is:unread",
			wantWhere: []string{"(r.full_name LIKE ? AND n.is_read = 0)"},
			wantArgs:  []string{"%cli%"},
			wantJoins: []string{"LEFT JOIN repositories r ON r.id = n.repository_id"},
		},
		{
			name:      "explicit OR",
			input:     "repo:cli OR repo:other",
			wantWhere: []string{"(r.full_name LIKE ? OR r.full_name LIKE ?)"},
			wantArgs:  []string{"%cli%", "%other%"},
			wantJoins: []string{"LEFT JOIN repositories r ON r.id = n.repository_id"},
		},
		{
			name:      "NOT operator",
			input:     "NOT author:bot",
			wantWhere: []string{"NOT (n.author_login LIKE ?)"},
			wantArgs:  []string{"%bot%"},
			wantJoins: []string{},
		},
		{
			name:      "comma OR within field",
			input:     "reason:review_requested,mention",
			wantWhere: []string{"(n.reason LIKE ? OR n.reason LIKE ?)"},
			wantArgs:  []string{"%review_requested%", "%mention%"},
			wantJoins: []string{},
		},
		{
			name:  "in:inbox",
			input: "in:inbox",

			wantWhere: []string{
				"(n.archived = 0 AND (n.snoozed_until IS NULL OR n.snoozed_until <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) AND n.muted = 0 AND n.filtered = 0)",
			},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "in:archive",
			input:     "in:archive",
			wantWhere: []string{"(n.archived = 1 AND n.muted = 0)"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:  "in:snoozed",
			input: "in:snoozed",

			wantWhere: []string{
				"(n.snoozed_until IS NOT NULL AND n.snoozed_until > strftime('%Y-%m-%dT%H:%M:%SZ', 'now') AND n.archived = 0 AND n.muted = 0)",
			},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "in:anywhere",
			input:     "in:anywhere",
			wantWhere: []string{"1"},
			wantArgs:  []string{},
			wantJoins: []string{},
		},
		{
			name:      "org: prefix matching",
			input:     "org:github",
			wantWhere: []string{"r.full_name LIKE ?"},
			wantArgs:  []string{"github/%"},
			wantJoins: []string{"LEFT JOIN repositories r ON r.id = n.repository_id"},
		},
		{
			name:      "state field",
			input:     "state:open",
			wantWhere: []string{"n.subject_state = ?"},
			wantArgs:  []string{"open"},
			wantJoins: []string{},
		},
		{
			name:      "author field",
			input:     "author:username",
			wantWhere: []string{"n.author_login LIKE ?"},
			wantArgs:  []string{"%username%"},
			wantJoins: []string{},
		},
		{
			name:      "type field",
			input:     "type:PullRequest",
			wantWhere: []string{"n.subject_type LIKE ?"},
			wantArgs:  []string{"%PullRequest%"},
			wantJoins: []string{},
		},
		{
			name:  "grouped OR with AND",
			input: "(repo:cli OR repo:other) AND is:unread",
			wantWhere: []string{
				"((r.full_name LIKE ? OR r.full_name LIKE ?) AND n.is_read = 0)",
			},
			wantArgs:  []string{"%cli%", "%other%"},
			wantJoins: []string{"LEFT JOIN repositories r ON r.id = n.repository_id"},
		},
		{
			name:    "unknown field error",
			input:   "badfield:value",
			wantErr: true,
		},
		{
			name:    "invalid in: value",
			input:   "in:badvalue",
			wantErr: true,
		},
		{
			name:    "unbalanced parens",
			input:   "(repo:cli",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.input, 50, 0)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// For queries without in: operators, expect muted-only default
			// Check if query has in: operator (any in: operator, not just in:anywhere)
			hasInOp := strings.Contains(tt.input, "in:")
			// Check if query explicitly asks for muted (is:muted or muted:true)
			explicitlyAsksForMuted := strings.Contains(tt.input, "is:muted") ||
				strings.Contains(tt.input, "muted:true")

			expectedWhere := tt.wantWhere
			if !hasInOp && tt.input != "" && !tt.wantErr && !explicitlyAsksForMuted {
				// Non-empty query without in: should have muted-only default
				// UNLESS it explicitly asks for muted notifications
				expectedWhere = append(expectedWhere, "n.muted = 0")
			}

			// Check WHERE clauses match
			if len(query.Where) != len(expectedWhere) {
				t.Errorf("WHERE count mismatch: want %d, got %d\nWant: %v\nGot: %v",
					len(expectedWhere), len(query.Where), expectedWhere, query.Where)
			}

			for i, want := range expectedWhere {
				if i >= len(query.Where) {
					break
				}
				if query.Where[i] != want {
					t.Errorf("WHERE[%d] mismatch:\nWant: %s\nGot:  %s", i, want, query.Where[i])
				}
			}

			// Check arguments match
			if len(query.Args) != len(tt.wantArgs) {
				t.Errorf("Args count mismatch: want %d, got %d\nWant: %v\nGot: %v",
					len(tt.wantArgs), len(query.Args), tt.wantArgs, query.Args)
			}

			for i, want := range tt.wantArgs {
				if i >= len(query.Args) {
					break
				}
				gotStr, ok := query.Args[i].(string)
				if !ok {
					t.Errorf("Args[%d] is not a string", i)
					continue
				}
				if gotStr != want {
					t.Errorf("Args[%d] mismatch: want %q, got %q", i, want, gotStr)
				}
			}

			// Check JOINs match
			if len(query.Joins) != len(tt.wantJoins) {
				t.Errorf("Joins count mismatch: want %d, got %d\nWant: %v\nGot: %v",
					len(tt.wantJoins), len(query.Joins), tt.wantJoins, query.Joins)
			}

			for i, want := range tt.wantJoins {
				if i >= len(query.Joins) {
					break
				}
				if query.Joins[i] != want {
					t.Errorf("Joins[%d] mismatch:\nWant: %s\nGot:  %s", i, want, query.Joins[i])
				}
			}

			// Check limit/offset
			if query.Limit != 50 {
				t.Errorf("Limit mismatch: want 50, got %d", query.Limit)
			}
			if query.Offset != 0 {
				t.Errorf("Offset mismatch: want 0, got %d", query.Offset)
			}
		})
	}
}

// TestIntegration_ComplexSQL tests complex real-world queries with exact output
func TestIntegration_ComplexSQL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantWhere string // The complete WHERE clause
		wantArgs  []string
	}{
		{
			name:      "review requests from cli",
			input:     "repo:cli reason:review_requested is:unread",
			wantWhere: "((r.full_name LIKE ? AND n.reason LIKE ?) AND n.is_read = 0)",
			wantArgs:  []string{"%cli%", "%review_requested%"},
		},
		{
			name:      "multiple repos with bot exclusion",
			input:     "(repo:cli OR repo:other) AND NOT author:bot",
			wantWhere: "((r.full_name LIKE ? OR r.full_name LIKE ?) AND NOT (n.author_login LIKE ?))",
			wantArgs:  []string{"%cli%", "%other%", "%bot%"},
		},
		{
			name:      "comma OR with multiple fields",
			input:     "repo:cli,other reason:review_requested,mention",
			wantWhere: "((r.full_name LIKE ? OR r.full_name LIKE ?) AND (n.reason LIKE ? OR n.reason LIKE ?))",
			wantArgs:  []string{"%cli%", "%other%", "%review_requested%", "%mention%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.input, 50, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Non-empty queries without in: get muted-only default as second WHERE clause
			// UNLESS they explicitly ask for muted (is:muted or muted:true)
			expectedCount := 1
			hasInOp := strings.Contains(tt.input, "in:")
			explicitlyAsksForMuted := strings.Contains(tt.input, "is:muted") ||
				strings.Contains(tt.input, "muted:true")
			if !hasInOp && tt.input != "" && !explicitlyAsksForMuted {
				expectedCount = 2
			}
			if len(query.Where) != expectedCount {
				t.Fatalf(
					"expected %d WHERE clause(s), got %d: %v",
					expectedCount,
					len(query.Where),
					query.Where,
				)
			}

			if query.Where[0] != tt.wantWhere {
				t.Errorf("WHERE mismatch:\nWant: %s\nGot:  %s", tt.wantWhere, query.Where[0])
			}

			if len(query.Args) != len(tt.wantArgs) {
				t.Fatalf("args count mismatch: want %d, got %d", len(tt.wantArgs), len(query.Args))
			}

			for i, want := range tt.wantArgs {
				got, ok := query.Args[i].(string)
				if !ok {
					t.Errorf("Args[%d] is not a string", i)
					continue
				}
				if got != want {
					t.Errorf("Args[%d]: want %q, got %q", i, want, got)
				}
			}
		})
	}
}

// TestIntegration_ErrorMessages tests that error messages are helpful
func TestIntegration_ErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErrSubstr string
	}{
		{
			name:          "unknown field",
			input:         "badfield:value",
			wantErrSubstr: "unknown field: badfield",
		},
		{
			name:          "invalid in value",
			input:         "in:badvalue",
			wantErrSubstr: "invalid value for in: operator: badvalue",
		},
		{
			name:          "invalid is value",
			input:         "is:badvalue",
			wantErrSubstr: "invalid value for is: operator: badvalue",
		},
		{
			name:          "invalid boolean",
			input:         "read:maybe",
			wantErrSubstr: "invalid boolean value",
		},
		{
			name:          "unbalanced parens",
			input:         "(repo:cli",
			wantErrSubstr: "expected closing parenthesis",
		},
		{
			name:          "unterminated quote",
			input:         `"unterminated`,
			wantErrSubstr: "unterminated quoted string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildQuery(tt.input, 50, 0)
			if err == nil {
				t.Fatal("expected error, got none")
			}

			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf(
					"error message doesn't contain expected substring\nWant substring: %q\nGot error: %q",
					tt.wantErrSubstr,
					err.Error(),
				)
			}
		})
	}
}

// TestIntegration_InboxQueryExact tests inbox query helper with exact output
func TestIntegration_InboxQueryExact(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantWhere []string
	}{
		{
			name:  "empty query gets defaults",
			input: "",
			wantWhere: []string{
				"n.archived = 0",
				"(n.snoozed_until IS NULL OR n.snoozed_until <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))",
				"n.muted = 0",
				"n.filtered = 0", // Empty query gets inbox defaults including filtered
			},
		},
		{
			name:  "simple query gets muted-only defaults",
			input: "repo:cli",
			wantWhere: []string{
				"r.full_name LIKE ?",
				"n.muted = 0", // Non-empty query without in: only excludes muted
			},
		},
		{
			name:  "explicit in:anywhere - no defaults",
			input: "in:anywhere repo:cli",
			wantWhere: []string{
				"(1 AND r.full_name LIKE ?)",
			},
		},
		{
			name:  "explicit in:inbox - no additional defaults",
			input: "in:inbox repo:cli",
			wantWhere: []string{

				"((n.archived = 0 AND (n.snoozed_until IS NULL OR n.snoozed_until <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) AND n.muted = 0 AND n.filtered = 0) AND r.full_name LIKE ?)",
			},
		},
		{
			name:  "tag query with in:anywhere -is:muted",
			input: "tags:urgent in:anywhere -is:muted is:unread",
			wantWhere: []string{

				"(((EXISTS (SELECT 1 FROM tag_assignments ta JOIN tags t ON t.id = ta.tag_id WHERE ta.entity_type = 'notification' AND ta.entity_id = n.id AND (t.slug LIKE ?)) AND 1) AND NOT (n.muted = 1)) AND n.is_read = 0)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.input, 50, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(query.Where) != len(tt.wantWhere) {
				t.Fatalf("WHERE count mismatch: want %d, got %d\nWant: %v\nGot: %v",
					len(tt.wantWhere), len(query.Where), tt.wantWhere, query.Where)
			}

			for i, want := range tt.wantWhere {
				if query.Where[i] != want {
					t.Errorf("WHERE[%d] mismatch:\nWant: %s\nGot:  %s", i, want, query.Where[i])
				}
			}
		})
	}
}

// TestIntegration_QueryContext tests the new explicit query context semantics
//

func TestIntegration_QueryContext(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantInboxDefs  bool // Should have inbox defaults (archived, snoozed, muted, filtered)
		wantMutedOnly  bool // Should have muted-only default
		wantNoDefaults bool // Should have no defaults
	}{
		{
			name:          "empty query gets inbox defaults",
			query:         "",
			wantInboxDefs: true,
		},
		{
			name:           "in:inbox query gets no additional defaults",
			query:          "in:inbox",
			wantNoDefaults: true, // in:inbox provides its own filters, no additional defaults
		},
		{
			name:           "in:inbox with additional filters gets no additional defaults",
			query:          "in:inbox repo:cli",
			wantNoDefaults: true,
		},
		{
			name:          "query without in: gets muted-only defaults",
			query:         "repo:cli",
			wantMutedOnly: true,
		},
		{
			name:           "in:anywhere overrides defaults",
			query:          "in:anywhere repo:cli",
			wantNoDefaults: true,
		},
		{
			name:           "in:archive overrides defaults",
			query:          "in:archive",
			wantNoDefaults: true,
		},
		{
			name:          "free text gets muted-only defaults",
			query:         "urgent fix",
			wantMutedOnly: true,
		},
		{
			name:           "tag query with in:anywhere -is:muted gets no defaults",
			query:          "tags:urgent in:anywhere -is:muted",
			wantNoDefaults: true,
		},
		{
			name:           "is:muted explicitly asks for muted - no muted-only default",
			query:          "is:muted",
			wantNoDefaults: true, // Should NOT get n.muted = 0 default
		},
		{
			name:           "muted:true explicitly asks for muted - no muted-only default",
			query:          "muted:true",
			wantNoDefaults: true, // Should NOT get n.muted = 0 default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.query, 50, 0)
			if err != nil {
				t.Fatalf("BuildQuery failed: %v", err)
			}

			allWhere := strings.Join(query.Where, " ")
			hasArchived := strings.Contains(allWhere, "n.archived = 0")
			hasSnoozed := strings.Contains(allWhere, "snoozed_until")
			hasMuted := strings.Contains(allWhere, "n.muted = 0")
			hasFiltered := strings.Contains(allWhere, "n.filtered = 0")

			//nolint:gocritic // ifElseChain: checking boolean test flags, not suitable for switch
			if tt.wantInboxDefs {
				if !hasArchived || !hasSnoozed || !hasMuted || !hasFiltered {
					t.Errorf(
						"expected inbox defaults, got archived=%v snoozed=%v muted=%v filtered=%v\nWHERE: %v",
						hasArchived,
						hasSnoozed,
						hasMuted,
						hasFiltered,
						query.Where,
					)
				}
			} else if tt.wantMutedOnly {
				if !hasMuted || hasArchived || hasFiltered {
					t.Errorf("expected muted-only default, got archived=%v muted=%v filtered=%v\nWHERE: %v",
						hasArchived, hasMuted, hasFiltered, query.Where)
				}
			} else if tt.wantNoDefaults {
				// Should not have any of the default filters added beyond what's in the query
				// For in:inbox, it provides its own filters but they're not "defaults"
				if strings.Contains(tt.query, "in:inbox") {
					// in:inbox provides filters, but they're part of the in: operator, not defaults
					// Just verify it has the filters
					if !hasArchived || !hasMuted || !hasFiltered {
						t.Errorf("in:inbox should provide its own filters, got archived=%v muted=%v filtered=%v\nWHERE: %v",
							hasArchived, hasMuted, hasFiltered, query.Where)
					}
				} else if strings.Contains(tt.query, "in:anywhere") {
					// in:anywhere should have 1 (SQLite true) and no other defaults
					if !strings.Contains(allWhere, " 1") && !strings.Contains(allWhere, "(1") {
						t.Errorf("in:anywhere should have 1, got WHERE: %v", query.Where)
					}
				} else if strings.Contains(tt.query, "is:muted") || strings.Contains(tt.query, "muted:true") {
					// Explicitly asking for muted should NOT have n.muted = 0 default
					hasMutedTrue := strings.Contains(allWhere, "n.muted = 1")
					if !hasMutedTrue {
						t.Errorf("is:muted or muted:true should have n.muted = 1, got WHERE: %v", query.Where)
					}
					if hasMuted {
						t.Errorf("is:muted or muted:true should NOT have n.muted = 0 default, got WHERE: %v", query.Where)
					}
				}
			}
		})
	}
}

// TestIntegration_FreeText tests free text search SQL generation
func TestIntegration_FreeText(t *testing.T) {
	query, err := BuildQuery("urgent fix", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Free text generates complex OR clause across multiple fields
	// Non-empty query without in: also gets muted-only default
	if len(query.Where) != 2 {
		t.Fatalf("expected 2 WHERE clauses (query + muted default), got %d", len(query.Where))
	}

	where := query.Where[0]

	// Should search across title, type, repo, author, state
	expectedParts := []string{
		"n.subject_title LIKE",
		"n.subject_type LIKE",
		"r.full_name LIKE",
		"n.author_login LIKE",
		"n.subject_state LIKE",
		"OR",
	}

	for _, part := range expectedParts {
		if !strings.Contains(where, part) {
			t.Errorf("expected WHERE to contain %q, got: %s", part, where)
		}
	}

	// Should have 12 args (6 fields × 2 words: title, type, repo, author, state, subject_number)
	if len(query.Args) != 12 {
		t.Errorf("expected 12 args for 2 words across 6 fields, got %d", len(query.Args))
	}

	// Should require repo join
	if len(query.Joins) != 1 {
		t.Errorf("expected 1 join, got %d", len(query.Joins))
	}
}

// TestCoverage_TokenString tests Token.String() method for coverage
func TestCoverage_TokenString(t *testing.T) {
	tokens := []parse.Token{
		{Type: parse.TokenEOF, Pos: 0},
		{Type: parse.TokenLParen, Value: "(", Pos: 5},
		{Type: parse.TokenRParen, Value: ")", Pos: 10},
		{Type: parse.TokenAnd, Value: "AND", Pos: 15},
		{Type: parse.TokenOr, Value: "OR", Pos: 20},
		{Type: parse.TokenNot, Value: "NOT", Pos: 25},
		{Type: parse.TokenColon, Value: ":", Pos: 30},
		{Type: parse.TokenComma, Value: ",", Pos: 35},
		{Type: parse.TokenField, Value: "repo", Pos: 40},
		{Type: parse.TokenValue, Value: "cli", Pos: 45},
		{Type: parse.TokenFreeText, Value: "text", Pos: 50},
	}

	for _, tok := range tokens {
		// Call string and ensure non-nil output.
		str := tok.String()
		if str == "" {
			t.Errorf("expected non-empty string for token type %s, got empty", tok.Type)
		}
		typeStr := tok.Type.String()
		if typeStr == "" {
			t.Errorf("expected non-empty string for token type %s, got empty", tok.Type)
		}
	}
}

// TestCoverage_ASTString tests AST node String() methods
func TestCoverage_ASTString(t *testing.T) {
	// Test all AST node String() methods
	term := &parse.Term{Field: "repo", Values: []string{"cli", "other"}}
	str := term.String()
	if str == "" {
		t.Errorf("expected non-empty string for term, got empty")
	}

	termNegated := &parse.Term{Field: "repo", Values: []string{"cli"}, Negated: true}
	str = termNegated.String()
	if str == "" {
		t.Errorf("expected non-empty string for term negated, got empty")
	}

	freeText := &parse.FreeText{Text: "urgent"}
	str = freeText.String()
	if str == "" {
		t.Errorf("expected non-empty string for free text, got empty")
	}

	notExpr := &parse.NotExpr{Expr: term}
	str = notExpr.String()
	if str == "" {
		t.Errorf("expected non-empty string for not expr, got empty")
	}

	binaryExpr := &parse.BinaryExpr{Op: "AND", Left: term, Right: freeText}
	str = binaryExpr.String()
	if str == "" {
		t.Errorf("expected non-empty string for binary expr, got empty")
	}

	parenExpr := &parse.ParenExpr{Expr: binaryExpr}
	str = parenExpr.String()
	if str == "" {
		t.Errorf("expected non-empty string for paren expr, got empty")
	}
}

// TestCoverage_ValidationNilNode tests validator with nil node
func TestCoverage_ValidationNilNode(t *testing.T) {
	validator := parse.NewValidator()
	err := validator.Validate(nil)
	if err != nil {
		t.Errorf("expected no error for nil node, got %v", err)
	}
}

// TestCoverage_BooleanValueVariations tests all boolean value formats
func TestCoverage_BooleanValueVariations(t *testing.T) {
	// Test all supported boolean formats
	boolValues := []string{"true", "false", "yes", "no", "1", "0"}

	for _, val := range boolValues {
		// Test read field
		query, err := BuildQuery("read:"+val, 50, 0)
		if err != nil {
			t.Errorf("read:%s failed: %v", val, err)
		}
		if len(query.Where) == 0 {
			t.Errorf("read:%s produced no WHERE clause", val)
		}

		// Test archived field
		query, err = BuildQuery("archived:"+val, 50, 0)
		if err != nil {
			t.Errorf("archived:%s failed: %v", val, err)
		}
		if len(query.Where) == 0 {
			t.Errorf("archived:%s produced no WHERE clause", val)
		}

		// Test muted field
		query, err = BuildQuery("muted:"+val, 50, 0)
		if err != nil {
			t.Errorf("muted:%s failed: %v", val, err)
		}
		if len(query.Where) == 0 {
			t.Errorf("muted:%s produced no WHERE clause", val)
		}

		// Test snoozed field
		query, err = BuildQuery("snoozed:"+val, 50, 0)
		if err != nil {
			t.Errorf("snoozed:%s failed: %v", val, err)
		}
		if len(query.Where) == 0 {
			t.Errorf("snoozed:%s produced no WHERE clause", val)
		}
	}
}

// TestCoverage_MultipleValuesForBooleans tests comma-separated boolean values
func TestCoverage_MultipleValuesForBooleans(t *testing.T) {
	// Test OR within boolean fields
	tests := []string{
		"read:true,false",
		"archived:true,false",
		"muted:yes,no",
		"snoozed:1,0",
	}

	for _, queryStr := range tests {
		query, err := BuildQuery(queryStr, 50, 0)
		if err != nil {
			t.Errorf("%s failed: %v", queryStr, err)
		}
		if len(query.Where) == 0 {
			t.Errorf("%s produced no WHERE clause", queryStr)
		}
		// Should contain OR for multiple values
		if query.Where[0] != "" && !contains(query.Where[0], "OR") {
			t.Errorf("%s should contain OR, got: %s", queryStr, query.Where[0])
		}
	}
}

// TestCoverage_ErrorPaths tests various error conditions
func TestCoverage_ErrorPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unsupported field", "badfield:value"},
		{"invalid in value", "in:badvalue"},
		{"invalid is value", "is:badvalue"},
		{"invalid boolean", "read:maybe"},
		{"unbalanced parens", "(repo:cli"},
		{"unexpected closing paren", "repo:cli)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildQuery(tt.input, 50, 0)
			if err == nil {
				t.Error("expected error, got none")
			}
		})
	}
}

// TestCoverage_EdgeCases tests edge cases
func TestCoverage_EdgeCases(t *testing.T) {
	// Empty value after colon (should error in parser)
	_, err := BuildQuery("repo:", 50, 0)
	if err == nil {
		t.Error("expected error for empty value")
	}

	// Only whitespace - parses as empty query, gets inbox defaults
	query, err := BuildQuery("   ", 50, 0)
	if err != nil {
		t.Errorf("whitespace-only query should parse as empty: %v", err)
	}
	// Empty query gets inbox defaults (archived, snoozed, muted, filtered)
	if len(query.Where) == 0 {
		t.Error("whitespace-only query (empty) should have inbox default WHERE clauses")
	}

	// Multiple operators in sequence
	_, err = BuildQuery("AND AND", 50, 0)
	if err == nil {
		t.Error("expected error for operators without operands")
	}

	// Negation without operand
	_, err = BuildQuery("NOT", 50, 0)
	if err == nil {
		t.Error("expected error for NOT without operand")
	}

	// Empty parentheses
	_, err = BuildQuery("()", 50, 0)
	if err == nil {
		t.Error("expected error for empty parentheses")
	}
}

// TestCoverage_LexerEdgeCases tests lexer edge cases
func TestCoverage_LexerEdgeCases(t *testing.T) {
	// Hyphen in middle of word (not NOT operator)
	query, err := BuildQuery("some-word", 50, 0)
	if err != nil {
		t.Errorf("hyphenated word should work: %v", err)
	}
	if len(query.Where) == 0 {
		t.Error("hyphenated word should produce WHERE clause")
	}

	// Multiple hyphens
	query, err = BuildQuery("my-test-word", 50, 0)
	if err != nil {
		t.Errorf("multi-hyphenated word should work: %v", err)
	}

	// Special characters in values
	query, err = BuildQuery("repo:org/repo-name", 50, 0)
	if err != nil {
		t.Errorf("slash and hyphen in value should work: %v", err)
	}

	// At sign in values
	query, err = BuildQuery("author:user@example.com", 50, 0)
	if err != nil {
		t.Errorf("at sign in value should work: %v", err)
	}

	// Dots in values
	query, err = BuildQuery("file:config.yaml", 50, 0)
	if err == nil {
		// file is not a known field, so should error in validation
		t.Error("unknown field should error")
	}
}

// TestCoverage_ParserAdvance tests parser advance edge cases
// Note: This test is removed as parser.current and parser.advance are unexported.
// The parser's internal state is not part of the public API.

// TestCoverage_BuildInboxQueryError tests error path in BuildInboxQuery
func TestCoverage_BuildInboxQueryError(t *testing.T) {
	// Create a query that will parse but fail in SQL generation
	// (though this is hard since validation catches most issues)

	// Test with invalid syntax
	_, err := BuildQuery("(repo:cli", 50, 0)
	if err == nil {
		t.Error("expected error for invalid syntax")
	}

	// Test with validation error
	_, err = BuildQuery("badfield:value", 50, 0)
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

// TestCoverage_HasInOperatorAllPaths tests HasInOperator with all node types
func TestCoverage_HasInOperatorAllPaths(t *testing.T) {
	// Create AST nodes and test HasInOperator
	inTerm := &parse.Term{Field: "in", Values: []string{"inbox"}}
	if !parse.HasInOperator(inTerm) {
		t.Error("HasInOperator should return true for in: term")
	}

	repoTerm := &parse.Term{Field: "repo", Values: []string{"cli"}}
	if parse.HasInOperator(repoTerm) {
		t.Error("HasInOperator should return false for non-in term")
	}

	binaryWithIn := &parse.BinaryExpr{Op: "AND", Left: inTerm, Right: repoTerm}
	if !parse.HasInOperator(binaryWithIn) {
		t.Error("HasInOperator should return true for binary expr containing in:")
	}

	binaryWithoutIn := &parse.BinaryExpr{
		Op:    "AND",
		Left:  repoTerm,
		Right: &parse.Term{Field: "reason", Values: []string{"mention"}},
	}
	if parse.HasInOperator(binaryWithoutIn) {
		t.Error("HasInOperator should return false for binary expr without in:")
	}

	notWithIn := &parse.NotExpr{Expr: inTerm}
	if !parse.HasInOperator(notWithIn) {
		t.Error("HasInOperator should return true for NOT expr containing in:")
	}

	parenWithIn := &parse.ParenExpr{Expr: inTerm}
	if !parse.HasInOperator(parenWithIn) {
		t.Error("HasInOperator should return true for paren expr containing in:")
	}

	freeText := &parse.FreeText{Text: "urgent"}
	if parse.HasInOperator(freeText) {
		t.Error("HasInOperator should return false for free text")
	}

	if parse.HasInOperator(nil) {
		t.Error("HasInOperator should return false for nil")
	}
}

// TestCoverage_OrgMultipleValues tests org with multiple values
func TestCoverage_OrgMultipleValues(t *testing.T) {
	query, err := BuildQuery("org:github,microsoft", 50, 0)
	if err != nil {
		t.Fatalf("org with multiple values failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain OR for multiple orgs
	if !contains(query.Where[0], "OR") {
		t.Error("multiple org values should use OR")
	}

	// Should use prefix matching (/)
	if len(query.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(query.Args))
	}

	arg1, ok := query.Args[0].(string)
	if !ok {
		t.Fatalf("Args[0] is not a string")
	}
	if arg1 != "github/%" {
		t.Errorf("expected 'github/%%' for org prefix, got %q", arg1)
	}
}

// TestCoverage_StateMultipleValues tests state with multiple values
func TestCoverage_StateMultipleValues(t *testing.T) {
	query, err := BuildQuery("state:open,closed", 50, 0)
	if err != nil {
		t.Fatalf("state with multiple values failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain OR
	if !contains(query.Where[0], "OR") {
		t.Error("multiple state values should use OR")
	}

	// Should use exact matching (not LIKE)
	if !contains(query.Where[0], "=") {
		t.Error("state should use = operator")
	}
}

// TestCoverage_InOperatorMultipleValues tests in: with multiple values
func TestCoverage_InOperatorMultipleValues(t *testing.T) {
	query, err := BuildQuery("in:inbox,archive", 50, 0)
	if err != nil {
		t.Fatalf("in: with multiple values failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain OR for multiple locations
	if !contains(query.Where[0], "OR") {
		t.Error("multiple in: values should use OR")
	}
}

// TestCoverage_IsOperatorMultipleValues tests is: with multiple values
func TestCoverage_IsOperatorMultipleValues(t *testing.T) {
	query, err := BuildQuery("is:read,unread", 50, 0)
	if err != nil {
		t.Fatalf("is: with multiple values failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain OR
	if !contains(query.Where[0], "OR") {
		t.Error("multiple is: values should use OR")
	}
}

// TestCoverage_LexerQuotedStrings tests various quoted string scenarios
func TestCoverage_LexerQuotedStrings(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple quoted", `"hello world"`, false},
		{"escaped quote", `"say \"hello\""`, false},
		{"empty quoted", `""`, false},
		{"unterminated quote", `"unterminated`, true},
		{"quote at end of input", `repo:"test`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := parse.NewLexer(tt.input)
			_, err := lexer.Tokenize()
			if tt.wantErr && err == nil {
				t.Error("expected error, got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestCoverage_ParserImplicitAND tests implicit AND behavior
func TestCoverage_ParserImplicitAND(t *testing.T) {
	// Implicit AND (space between terms)
	query, err := BuildQuery("repo:cli is:unread", 50, 0)
	if err != nil {
		t.Fatalf("implicit AND failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should create AND expression
	if !strings.Contains(query.Where[0], "AND") {
		t.Error("implicit AND should create AND expression")
	}
}

// TestCoverage_ParserComplexNesting tests deeply nested expressions
func TestCoverage_ParserComplexNesting(t *testing.T) {
	query, err := BuildQuery(
		"((repo:cli OR repo:other) AND is:unread) OR (repo:test AND is:read)",
		50,
		0,
	)
	if err != nil {
		t.Fatalf("complex nesting failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain both AND and OR
	where := query.Where[0]
	if !strings.Contains(where, "AND") || !strings.Contains(where, "OR") {
		t.Error("complex nesting should contain both AND and OR")
	}
}

// TestCoverage_FreeTextEdgeCases tests free text with various characters
func TestCoverage_FreeTextEdgeCases(t *testing.T) {
	tests := []string{
		"single",
		"multiple words",
		"with-hyphen",
		"with_underscore",
		"with123numbers",
		"UPPERCASE",
		"MixedCase",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			query, err := BuildQuery(input, 50, 0)
			if err != nil {
				t.Errorf("free text %q failed: %v", input, err)
			}
			if len(query.Where) == 0 {
				t.Errorf("free text %q produced no WHERE clause", input)
			}
		})
	}
}

// TestCoverage_NegatedTerms tests negated field:value terms
func TestCoverage_NegatedTerms(t *testing.T) {
	query, err := BuildQuery("-repo:unwanted", 50, 0)
	if err != nil {
		t.Fatalf("negated term failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain NOT
	if !strings.Contains(query.Where[0], "NOT") {
		t.Error("negated term should contain NOT")
	}
}

// TestCoverage_NegatedComplexExpression tests NOT with complex expressions
func TestCoverage_NegatedComplexExpression(t *testing.T) {
	query, err := BuildQuery("NOT (repo:unwanted OR author:bot)", 50, 0)
	if err != nil {
		t.Fatalf("negated complex expression failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain NOT and OR
	where := query.Where[0]
	if !strings.Contains(where, "NOT") || !strings.Contains(where, "OR") {
		t.Error("negated complex expression should contain NOT and OR")
	}
}

// TestCoverage_MultipleNegations tests multiple NOT operators
func TestCoverage_MultipleNegations(t *testing.T) {
	query, err := BuildQuery("-repo:unwanted -author:bot", 50, 0)
	if err != nil {
		t.Fatalf("multiple negations failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain multiple NOT
	where := query.Where[0]
	notCount := 0
	for i := 0; i < len(where)-2; i++ {
		if where[i:i+3] == "NOT" {
			notCount++
		}
	}
	if notCount < 2 {
		t.Errorf("expected at least 2 NOT operators, found %d in: %s", notCount, where)
	}
}

// TestCoverage_InOperatorAllValues tests all in: operator values
func TestCoverage_InOperatorAllValues(t *testing.T) {
	values := []string{"inbox", "archive", "snoozed", "anywhere"}

	for _, val := range values {
		t.Run("in:"+val, func(t *testing.T) {
			query, err := BuildQuery("in:"+val, 50, 0)
			if err != nil {
				t.Errorf("in:%s failed: %v", val, err)
			}
			if len(query.Where) == 0 {
				t.Errorf("in:%s produced no WHERE clause", val)
			}
		})
	}
}

// TestCoverage_IsOperatorAllValues tests all is: operator values
func TestCoverage_IsOperatorAllValues(t *testing.T) {
	values := []struct {
		val     string
		wantErr bool
	}{
		{"read", false},
		{"unread", false},
		{"muted", false},
		{"starred", false}, // Now implemented
	}

	for _, tc := range values {
		t.Run("is:"+tc.val, func(t *testing.T) {
			_, err := BuildQuery("is:"+tc.val, 50, 0)
			if tc.wantErr && err == nil {
				t.Errorf("is:%s should error but didn't", tc.val)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("is:%s failed: %v", tc.val, err)
			}
		})
	}
}

// TestCoverage_ValidatorWithComplexAST tests validator with complex AST structures
func TestCoverage_ValidatorWithComplexAST(t *testing.T) {
	// Test validator with deeply nested structures
	ast, err := ParseAndValidate(
		"((repo:cli OR repo:other) AND (is:unread OR is:read)) OR (author:bot AND -repo:test)",
	)
	if err != nil {
		t.Fatalf("complex validation failed: %v", err)
	}

	if ast == nil {
		t.Error("expected non-nil AST")
	}
}

// TestCoverage_LexerAtEndOfInput tests lexer at end of input
func TestCoverage_LexerAtEndOfInput(t *testing.T) {
	lexer := parse.NewLexer("repo:cli")
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	// Last token should be EOF
	lastToken := tokens[len(tokens)-1]
	if lastToken.Type != parse.TokenEOF {
		t.Errorf("last token should be EOF, got %v", lastToken.Type)
	}
}

// TestCoverage_ParserTrailingWhitespace tests parser with trailing whitespace
func TestCoverage_ParserTrailingWhitespace(t *testing.T) {
	inputs := []string{
		"repo:cli   ",
		"  repo:cli",
		"  repo:cli  ",
		"repo:cli\n",
		"repo:cli\t",
	}

	for _, input := range inputs {
		t.Run("input with whitespace", func(t *testing.T) {
			query, err := BuildQuery(input, 50, 0)
			if err != nil {
				t.Errorf("query with whitespace failed: %v", err)
			}
			if len(query.Where) == 0 {
				t.Error("query with whitespace produced no WHERE clause")
			}
		})
	}
}

// TestCoverage_SQLBuilderComplexORCombinations tests complex OR combinations
func TestCoverage_SQLBuilderComplexORCombinations(t *testing.T) {
	// Test OR with different field types
	query, err := BuildQuery("(repo:cli OR reason:mention) AND (author:user OR is:unread)", 50, 0)
	if err != nil {
		t.Fatalf("complex OR combinations failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should have repo join
	if len(query.Joins) == 0 {
		t.Error("expected JOIN for repo field")
	}
}

// TestCoverage_NegatedFreeText tests negated free text
func TestCoverage_NegatedFreeText(t *testing.T) {
	query, err := BuildQuery("-urgent", 50, 0)
	if err != nil {
		t.Fatalf("negated free text failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	// Should contain NOT
	if !strings.Contains(query.Where[0], "NOT") {
		t.Error("negated free text should contain NOT")
	}
}

// TestCoverage_EmptyInboxQuery tests BuildQuery with empty query
func TestCoverage_EmptyInboxQuery(t *testing.T) {
	query, err := BuildQuery("", 100, 10)
	if err != nil {
		t.Fatalf("empty inbox query failed: %v", err)
	}

	// Should have default filters
	if len(query.Where) < 3 {
		t.Errorf(
			"empty inbox query should have at least 3 default filters, got %d",
			len(query.Where),
		)
	}

	// Should have correct limit and offset
	if query.Limit != 100 {
		t.Errorf("limit should be 100, got %d", query.Limit)
	}
	if query.Offset != 10 {
		t.Errorf("offset should be 10, got %d", query.Offset)
	}
}

// TestCoverage_MixedOperatorsAndGrouping tests mixing explicit and implicit operators
func TestCoverage_MixedOperatorsAndGrouping(t *testing.T) {
	// Mix of implicit AND, explicit OR, and grouping
	query, err := BuildQuery("repo:cli (is:unread OR is:read) reason:mention", 50, 0)
	if err != nil {
		t.Fatalf("mixed operators failed: %v", err)
	}

	if len(query.Where) == 0 {
		t.Fatal("expected WHERE clause")
	}

	where := query.Where[0]
	if !strings.Contains(where, "AND") || !strings.Contains(where, "OR") {
		t.Error("mixed operators should contain both AND and OR")
	}
}

func TestIntegration_TagsQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantErr      bool
		wantContains string // What the SQL should contain
	}{
		{
			name:         "simple tag query",
			query:        "tags:urgent",
			wantErr:      false,
			wantContains: "slug LIKE",
		},
		{
			name:         "tag with NOT operator",
			query:        "tags:urgent AND NOT tags:resolved",
			wantErr:      false,
			wantContains: "slug LIKE",
		},
		{
			name:         "tag with negated syntax",
			query:        "tags:urgent AND -tags:resolved",
			wantErr:      false,
			wantContains: "slug LIKE",
		},
		{
			name:         "multiple tags OR",
			query:        "tags:urgent,bug",
			wantErr:      false,
			wantContains: "slug LIKE",
		},
		{
			name:         "partial tag match",
			query:        "tags:urg",
			wantErr:      false,
			wantContains: "slug LIKE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := BuildQuery(tt.query, 50, 0)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.wantContains != "" {
				whereClause := strings.Join(query.Where, " ")
				if !strings.Contains(whereClause, tt.wantContains) {
					t.Errorf(
						"BuildQuery() where = %v, should contain %v\nArgs: %v",
						whereClause,
						tt.wantContains,
						query.Args,
					)
				}
			}
		})
	}
}
