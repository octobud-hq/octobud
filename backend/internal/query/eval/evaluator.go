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

// Package eval provides the evaluator for the query language.
package eval

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/query/parse"
)

// Evaluator evaluates whether a notification matches a query AST
type Evaluator struct {
	ast parse.Node
}

// NewEvaluator creates a new evaluator for the given AST node
// This is a pure translation - no default filters are applied.
func NewEvaluator(ast parse.Node) *Evaluator {
	return &Evaluator{ast: ast}
}

// Matches returns true if the notification matches the query with explicit query context:
// - Empty query "" → Default inbox: exclude archived, snoozed (active), muted, filtered (backward compatibility)
// - Query with in: operator (any value) → No defaults (in: operator explicitly handles lifecycle)
// - Query without in: operator (non-empty) → Apply muted-only default (exclude muted unless explicitly requested)
func (e *Evaluator) Matches(notif *db.Notification, repo *db.Repository) bool {
	// 1. Empty query → Default inbox: exclude archived, snoozed (active), muted, filtered (backward compatibility)
	if e.ast == nil {
		return e.matchesInboxDefaults(notif)
	}

	// 2. Query with in: operator (any value) → No defaults (in: operator explicitly handles lifecycle)
	// This includes in:inbox, in:archive, in:snoozed, in:filtered, in:anywhere, etc.
	if parse.HasInOperator(e.ast) {
		return e.evaluateNode(notif, repo, e.ast)
	}

	// 3. Query without in: operator (non-empty) → Apply muted-only default
	// BUT: if query explicitly asks for muted (is:muted or muted:true), allow it
	if notif.Muted && !parse.HasExplicitMuted(e.ast) {
		return false
	}

	// Evaluate the query AST
	return e.evaluateNode(notif, repo, e.ast)
}

// matchesInboxDefaults checks if notification passes inbox default filters
func (e *Evaluator) matchesInboxDefaults(notif *db.Notification) bool {
	// Inbox defaults: exclude archived, snoozed (active), muted, filtered
	if notif.Archived {
		return false
	}
	if notif.SnoozedUntil.Valid && notif.SnoozedUntil.Time.After(time.Now()) {
		return false
	}
	if notif.Muted {
		return false
	}
	if notif.Filtered {
		return false
	}
	return true
}

func (e *Evaluator) evaluateNode(
	notif *db.Notification,
	repo *db.Repository,
	node parse.Node,
) bool {
	switch n := node.(type) {
	case *parse.BinaryExpr:
		left := e.evaluateNode(notif, repo, n.Left)
		right := e.evaluateNode(notif, repo, n.Right)
		switch n.Op {
		case "AND":
			return left && right
		case "OR":
			return left || right
		default:
			return false
		}

	case *parse.NotExpr:
		return !e.evaluateNode(notif, repo, n.Expr)

	case *parse.ParenExpr:
		return e.evaluateNode(notif, repo, n.Expr)

	case *parse.Term:
		result := e.evaluateTerm(notif, repo, n)
		if n.Negated {
			return !result
		}
		return result

	case *parse.FreeText:
		return e.evaluateFreeText(notif, repo, n.Text)

	default:
		return false
	}
}

func (e *Evaluator) evaluateTerm(
	notif *db.Notification,
	repo *db.Repository,
	term *parse.Term,
) bool {
	// Check if ANY value matches (OR logic within a term)
	for _, value := range term.Values {
		if e.evaluateFieldValue(notif, repo, term.Field, value) {
			return true
		}
	}
	return false
}

func (e *Evaluator) evaluateFieldValue(
	notif *db.Notification,
	repo *db.Repository,
	field, value string,
) bool {
	switch field {
	case "is":
		return e.evaluateIsCondition(notif, value)
	case "in":
		return e.evaluateInCondition(notif, value)
	case "repo":
		if repo != nil {
			return strings.Contains(strings.ToLower(repo.FullName), strings.ToLower(value))
		}
		return false
	case "author":
		if notif.AuthorLogin.Valid {
			return notif.AuthorLogin.String == value
		}
		return false
	case "reason":
		if notif.Reason.Valid {
			return notif.Reason.String == value
		}
		return false
	case "type":
		return strings.EqualFold(notif.SubjectType, value)
	// Add other fields as needed (participant, label, etc.)
	default:
		return true // Unknown fields don't filter
	}
}

func (e *Evaluator) evaluateIsCondition(notif *db.Notification, value string) bool {
	switch value {
	case "read":
		return notif.IsRead
	case "unread":
		return !notif.IsRead
	case "archived":
		return notif.Archived
	case "inbox":
		return !notif.Archived
	case "muted":
		return notif.Muted
	case "unmuted":
		return !notif.Muted
	case "starred":
		return notif.Starred
	case "unstarred":
		return !notif.Starred
	case "snoozed":
		return notif.SnoozedUntil.Valid && notif.SnoozedUntil.Time.After(time.Now())
	case "unsnoozed", "active":
		return !notif.SnoozedUntil.Valid || notif.SnoozedUntil.Time.Before(time.Now())
	case "filtered":
		return notif.Filtered
	default:
		return true
	}
}

func (e *Evaluator) evaluateInCondition(notif *db.Notification, value string) bool {
	switch value {
	case "inbox":
		// in:inbox - exclude archived, snoozed, muted, filtered
		return !notif.Archived &&
			(!notif.SnoozedUntil.Valid || notif.SnoozedUntil.Time.Before(time.Now())) &&
			!notif.Muted &&
			!notif.Filtered
	case "archive":
		// in:archive - show only archived (exclude muted)
		return notif.Archived && !notif.Muted
	case "snoozed":
		// in:snoozed - show only snoozed (exclude archived, muted)
		return notif.SnoozedUntil.Valid &&
			notif.SnoozedUntil.Time.After(time.Now()) &&
			!notif.Archived &&
			!notif.Muted
	case "filtered":
		// in:filtered - exclude snoozed, archived, muted
		return notif.Filtered &&
			!notif.Archived &&
			(!notif.SnoozedUntil.Valid || notif.SnoozedUntil.Time.Before(time.Now())) &&
			!notif.Muted
	case "anywhere":
		// in:anywhere - show all (no filters)
		return true
	default:
		return true
	}
}

// evaluateFreeText matches free text against multiple fields
// This matches the SQL builder's free text search logic:
// - Subject title
// - Subject type
// - Repository full name
// - Author login
// - subject_state (extracted column, preferred over subject_raw JSON parsing)
// - subject_number (cast to text for search)
//

func (e *Evaluator) evaluateFreeText(
	notif *db.Notification,
	repo *db.Repository,
	text string,
) bool {
	lowerText := strings.ToLower(text)

	// Check subject title
	if strings.Contains(strings.ToLower(notif.SubjectTitle), lowerText) {
		return true
	}

	// Check subject type
	if strings.Contains(strings.ToLower(notif.SubjectType), lowerText) {
		return true
	}

	// Check repository full name
	if repo != nil && strings.Contains(strings.ToLower(repo.FullName), lowerText) {
		return true
	}

	// Check author login
	if notif.AuthorLogin.Valid &&
		strings.Contains(strings.ToLower(notif.AuthorLogin.String), lowerText) {
		return true
	}

	// Check subject_state column (extracted column, preferred over JSON parsing)
	// Note: We use the extracted column instead of subject_raw->>'state' for performance
	// since the JSON path index has been removed in favor of the column index
	if notif.SubjectState.Valid &&
		strings.Contains(strings.ToLower(notif.SubjectState.String), lowerText) {
		return true
	}

	// Fallback to parsing from subject_raw only if subject_state is not available
	// This is a legacy fallback for edge cases where the column might not be populated
	if !notif.SubjectState.Valid && notif.SubjectRaw.Valid {
		var subjectData map[string]interface{}
		if err := json.Unmarshal(notif.SubjectRaw.RawMessage, &subjectData); err == nil {
			if state, ok := subjectData["state"].(string); ok {
				if strings.Contains(strings.ToLower(state), lowerText) {
					return true
				}
			}
		}
	}

	// Check subject_number (cast to text for search)
	if notif.SubjectNumber.Valid {
		numberStr := strconv.Itoa(int(notif.SubjectNumber.Int32))
		if strings.Contains(strings.ToLower(numberStr), lowerText) {
			return true
		}
	}

	// Check subject_state_reason column
	if notif.SubjectStateReason.Valid &&
		strings.Contains(strings.ToLower(notif.SubjectStateReason.String), lowerText) {
		return true
	}

	// Check subject_merged (convert boolean to text for search)
	if notif.SubjectMerged.Valid {
		mergedStr := "false"
		if notif.SubjectMerged.Bool {
			mergedStr = "true"
		}
		if strings.Contains(strings.ToLower(mergedStr), lowerText) ||
			(notif.SubjectMerged.Bool && strings.Contains(lowerText, "merged")) {
			return true
		}
	}

	return false
}
