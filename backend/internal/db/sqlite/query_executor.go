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

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// notificationColumns returns the list of all notification table columns for SQLite.
// Note: SQLite doesn't have tag_ids column - we use a junction table instead.
func notificationColumns(includeSubject bool) string {
	columns := []string{
		"n.id",
		"n.user_id",
		"n.github_id",
		"n.repository_id",
		"n.pull_request_id",
		"n.subject_type",
		"n.subject_title",
		"n.subject_url",
		"n.subject_latest_comment_url",
		"n.reason",
		"n.archived",
		"n.github_unread",
		"n.github_updated_at",
		"n.github_last_read_at",
		"n.github_url",
		"n.github_subscription_url",
		"n.imported_at",
		"n.payload",
		"n.subject_fetched_at",
		"n.author_login",
		"n.author_id",
		"n.is_read",
		"n.muted",
		"n.snoozed_until",
		"n.effective_sort_date",
		"n.snoozed_at",
		"n.starred",
		"n.filtered",
		"n.subject_number",
		"n.subject_state",
		"n.subject_merged",
		"n.subject_state_reason",
		"n.subject_draft",
	}

	if includeSubject {
		// Insert subject_raw after payload
		columns = append(columns[:18], append([]string{"n.subject_raw"}, columns[18:]...)...)
	}

	return "SELECT " + strings.Join(columns, ", ") + " FROM notifications n"
}

// buildQueryClauses assembles the JOIN and WHERE fragments (with args) shared by the
// dynamic notification queries. The user_id condition is always included first.
func buildQueryClauses(
	userID string,
	query db.NotificationQuery,
) (joins, where string, args []interface{}) {
	if len(query.Joins) > 0 {
		joins = " " + strings.Join(query.Joins, " ")
	}

	whereConditions := []string{"n.user_id = ?"}
	args = append([]interface{}{userID}, query.Args...)
	if len(query.Where) > 0 {
		whereConditions = append(whereConditions, query.Where...)
	}
	where = " WHERE " + strings.Join(whereConditions, " AND ")

	return joins, where, args
}

// repositoryJoin is the exact join the query compiler emits for repo:/org:/free-text
// terms, so the counts query can reuse it without producing a duplicate alias.
const repositoryJoin = "LEFT JOIN repositories r ON r.id = n.repository_id"

// countNotificationsByRepository counts the notifications matching a query, grouped by
// repository, joined with the repository's identity and ordered by unread, then total,
// then name. Only repositories with at least one match are returned.
func countNotificationsByRepository(
	ctx context.Context,
	s *Store,
	userID string,
	query db.NotificationQuery,
) ([]db.RepositoryNotificationCount, error) {
	if !slices.Contains(query.Joins, repositoryJoin) {
		query.Joins = append(slices.Clone(query.Joins), repositoryJoin)
	}
	joins, where, args := buildQueryClauses(userID, query)

	//nolint:gosec // G202: joins/where are compiler-controlled fragments; values are bound params
	countQuery := "SELECT n.repository_id, r.name, r.full_name, r.owner_login, " +
		"r.owner_avatar_url, r.html_url, COUNT(*) AS total, " +
		"SUM(CASE WHEN n.is_read = 0 THEN 1 ELSE 0 END) AS unread " +
		"FROM notifications n" + joins + where +
		" GROUP BY n.repository_id" +
		" ORDER BY unread DESC, total DESC, lower(r.full_name) ASC"

	var rows *sql.Rows
	err := db.RetryVoidOnBusy(ctx, func() error {
		var queryErr error
		rows, queryErr = s.dbConn.QueryContext(ctx, countQuery, args...)
		return queryErr
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count notifications by repository: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Nothing useful to do with a close error here; the scan result is what matters
			_ = closeErr
		}
	}()

	counts := make([]db.RepositoryNotificationCount, 0)
	for rows.Next() {
		var c db.RepositoryNotificationCount
		var name, fullName sql.NullString
		if scanErr := rows.Scan(
			&c.RepositoryID, &name, &fullName, &c.OwnerLogin,
			&c.OwnerAvatarURL, &c.HTMLURL, &c.Total, &c.Unread,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan repository count: %w", scanErr)
		}
		c.Name = name.String
		c.FullName = fullName.String
		counts = append(counts, c)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("error iterating repository counts: %w", rowsErr)
	}

	return counts, nil
}

// listNotificationsFromQuery executes a dynamic notification query for SQLite.
func listNotificationsFromQuery(
	ctx context.Context,
	s *Store,
	userID string,
	query db.NotificationQuery,
) (db.ListNotificationsFromQueryResult, error) {
	// Build the SELECT query - conditionally exclude subject_raw to reduce data transfer
	baseSelect := notificationColumns(query.IncludeSubject)

	joins, where, args := buildQueryClauses(userID, query)

	// Add ORDER BY
	orderBy := " ORDER BY n.effective_sort_date DESC, n.imported_at DESC"

	// Add LIMIT and OFFSET
	limitOffset := fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	// Combine everything
	selectQuery := baseSelect + joins + where + orderBy + limitOffset

	// Execute query
	var rows *sql.Rows
	err := db.RetryVoidOnBusy(ctx, func() error {
		var queryErr error
		rows, queryErr = s.dbConn.QueryContext(ctx, selectQuery, args...)
		return queryErr
	})
	if err != nil {
		return db.ListNotificationsFromQueryResult{}, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			// Error closing rows - log if we had a logger, but can't return it
			_ = closeErr
		}
	}()

	var notifications []db.Notification
	for rows.Next() {
		var n Notification
		// Column order must match notificationColumns() function above
		scanColumns := []any{
			&n.ID,
			&n.UserID,
			&n.GithubID,
			&n.RepositoryID,
			&n.PullRequestID,
			&n.SubjectType,
			&n.SubjectTitle,
			&n.SubjectUrl,
			&n.SubjectLatestCommentUrl,
			&n.Reason,
			&n.Archived,
			&n.GithubUnread,
			&n.GithubUpdatedAt,
			&n.GithubLastReadAt,
			&n.GithubUrl,
			&n.GithubSubscriptionUrl,
			&n.ImportedAt,
			&n.Payload,
			&n.SubjectFetchedAt,
			&n.AuthorLogin,
			&n.AuthorID,
			&n.IsRead,
			&n.Muted,
			&n.SnoozedUntil,
			&n.EffectiveSortDate,
			&n.SnoozedAt,
			&n.Starred,
			&n.Filtered,
			&n.SubjectNumber,
			&n.SubjectState,
			&n.SubjectMerged,
			&n.SubjectStateReason,
			&n.SubjectDraft,
		}

		// For convenience, add subject_raw if requested
		if query.IncludeSubject {
			scanColumns = append(
				scanColumns[:18],
				append([]any{&n.SubjectRaw}, scanColumns[18:]...)...)
		}

		if scanErr := rows.Scan(scanColumns...); scanErr != nil {
			return db.ListNotificationsFromQueryResult{}, fmt.Errorf(
				"failed to scan notification: %w",
				scanErr,
			)
		}
		notifications = append(notifications, s.toDBNotification(ctx, userID, n))
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return db.ListNotificationsFromQueryResult{},
			fmt.Errorf("error iterating rows: %w", rowsErr)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM notifications n" + joins + where
	var total int64
	err = db.RetryVoidOnBusy(ctx, func() error {
		return s.dbConn.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	})
	if err != nil {
		return db.ListNotificationsFromQueryResult{}, fmt.Errorf("failed to get count: %w", err)
	}

	return db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         total,
	}, nil
}

// bulkUpdateByQuery updates notifications matching a query.
func bulkUpdateByQuery(
	ctx context.Context,
	s *Store,
	userID string,
	setClause string,
	query db.NotificationQuery,
) (int64, error) {
	joins, where, args := buildQueryClauses(userID, query)

	// Use subquery to handle the alias properly
	//nolint:gosec // G201: SQL string formatting is safe - setClause, joins, and where are controlled
	sqlQuery := fmt.Sprintf(
		"UPDATE notifications SET %s WHERE id IN (SELECT n.id FROM notifications n%s%s)",
		setClause, joins, where,
	)
	var result sql.Result
	err := db.RetryVoidOnBusy(ctx, func() error {
		var execErr error
		result, execErr = s.dbConn.ExecContext(ctx, sqlQuery, args...)
		return execErr
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// bulkSnoozeByQuery snoozes notifications matching a query.
func bulkSnoozeByQuery(
	ctx context.Context,
	s *Store,
	userID string,
	arg db.BulkSnoozeNotificationsByQueryParams,
) (int64, error) {
	joins, where, whereArgs := buildQueryClauses(userID, arg.Query)

	// Prepend snoozed_until (used twice: for snoozed_until and effective_sort_date) to the
	// WHERE args, which already start with userID.
	snoozedUntil := formatNullTime(arg.SnoozedUntil)
	args := make([]interface{}, 0, len(whereArgs)+2)
	args = append(args, snoozedUntil, snoozedUntil)
	args = append(args, whereArgs...)

	// Use subquery to handle the alias properly
	//nolint:gosec // G201: SQL string formatting is safe - joins and where are controlled
	sqlQuery := fmt.Sprintf(
		"UPDATE notifications SET snoozed_until = ?, "+
			"snoozed_at = strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', 'now'), "+
			"effective_sort_date = ? WHERE id IN (SELECT n.id FROM notifications n%s%s)",
		joins,
		where,
	)
	var result sql.Result
	err := db.RetryVoidOnBusy(ctx, func() error {
		var execErr error
		result, execErr = s.dbConn.ExecContext(ctx, sqlQuery, args...)
		return execErr
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
