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
	"errors"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/query/parse"
	"github.com/octobud-hq/octobud/backend/internal/query/sql"
)

// Error definitions
var (
	ErrTokenizationFailed  = errors.New("tokenization failed")
	ErrParseFailed         = errors.New("parse failed")
	ErrSQLGenerationFailed = errors.New("SQL generation failed")
)

// Node is an alias for parse.Node for convenience
type Node = parse.Node

// ParseAndValidate parses a query string and validates it
// Returns the AST node if successful
func ParseAndValidate(queryStr string) (Node, error) {
	if queryStr == "" {
		return nil, nil
	}

	lexer := parse.NewLexer(queryStr)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, errors.Join(ErrTokenizationFailed, err)
	}

	parser := parse.NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, errors.Join(ErrParseFailed, err)
	}

	validator := parse.NewValidator()
	if err := validator.Validate(ast); err != nil {
		return nil, errors.Join(parse.ErrValidationFailed, err)
	}

	return ast, nil
}

// BuildQuery parses a query string and generates SQL with explicit query context:
// - Empty query "" → Default inbox: exclude archived, snoozed (active), muted, filtered (backward compatibility)
// - Query with in: operator (any value) → No defaults (in: operator explicitly handles lifecycle)
// - Query without in: operator (non-empty) → Apply muted-only default (exclude muted unless explicitly requested)
func BuildQuery(queryStr string, limit, offset int32) (db.NotificationQuery, error) {
	return BuildQueryWithOptions(queryStr, limit, offset, true)
}

// BuildQueryWithOptions parses a query string and generates SQL with unified business rules
func BuildQueryWithOptions(
	queryStr string,
	limit, offset int32,
	includeSubject bool,
) (db.NotificationQuery, error) {
	ast, err := ParseAndValidate(queryStr)
	if err != nil {
		return db.NotificationQuery{}, err
	}

	builder := sql.NewBuilder()
	query, err := builder.Build(ast)
	if err != nil {
		return db.NotificationQuery{}, errors.Join(ErrSQLGenerationFailed, err)
	}

	// Apply unified business rules
	query = applyUnifiedDefaults(query, ast, queryStr)

	query.Limit = limit
	query.Offset = offset
	query.IncludeSubject = includeSubject

	return query, nil
}

// applyUnifiedDefaults applies default filters based on explicit query context:
// - Empty query → Default inbox: exclude archived, snoozed (active), muted, filtered (backward compatibility)
// - Query with in: operator (any value) → No defaults (in: operator explicitly handles lifecycle)
// - Query without in: operator (non-empty) → Apply muted-only default (exclude muted unless explicitly requested)
func applyUnifiedDefaults(
	query db.NotificationQuery,
	ast Node,
	queryStr string,
) db.NotificationQuery {
	// 1. Empty query → Default inbox: exclude archived, snoozed (active), muted, filtered (backward compatibility)
	if queryStr == "" || ast == nil {
		return ApplyInboxDefaults(query)
	}

	// 2. Query with in: operator (any value) → No defaults (in: operator explicitly handles lifecycle)
	// This includes in:inbox, in:archive, in:snoozed, in:filtered, in:anywhere, etc.
	if parse.HasInOperator(ast) {
		return query
	}

	// 3. Query without in: operator (non-empty) → Apply muted-only default
	// This allows archived, snoozed, and filtered to show unless explicitly excluded
	// BUT: if query explicitly asks for muted (is:muted or muted:true), don't apply the default
	if parse.HasExplicitMuted(ast) {
		return query
	}
	return ApplyMutedOnlyDefaults(query)
}
