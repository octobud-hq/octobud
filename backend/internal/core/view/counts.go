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

// Package view provides the view service.
package view

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/octobud-hq/octobud/backend/internal/query"
)

// Error definitions
var (
	ErrFailedToBuildQuery        = errors.New("failed to build query")
	ErrFailedToListNotifications = errors.New("failed to list notifications")
)

// calculateViewUnreadCount calculates the count of "new" (unread) notifications for a view with given query.
func (s *Service) calculateViewUnreadCount(
	ctx context.Context,
	userID string,
	viewQuery sql.NullString,
) (int64, error) {
	queryStr := ""
	if viewQuery.Valid {
		queryStr = viewQuery.String
	}

	// Always AND the view's query with is:unread to get the unread count
	if queryStr == "" {
		// If no query specified, just use is:unread (archived/muted/snoozed excluded by default)
		queryStr = "is:unread"
	} else {
		// Combine the view's query with is:unread
		queryStr = fmt.Sprintf("(%s) AND is:unread", queryStr)
	}

	// Use unified BuildQuery which applies business rules based on query content
	dbQuery, err := query.BuildQuery(queryStr, 1, 0)

	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Total, nil
}

// calculateInboxUnreadCount calculates the count of "new" (unread) notifications in the inbox.
func (s *Service) calculateInboxUnreadCount(ctx context.Context, userID string) (int64, error) {
	// Inbox uses explicit in:inbox query, badge count shows only unread items
	queryStr := "in:inbox is:unread"

	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Total, nil
}

// calculateEverythingUnreadCount calculates the count of "new" (unread) notifications in everything view.
func (s *Service) calculateEverythingUnreadCount(
	ctx context.Context,
	userID string,
) (int64, error) {
	// Everything view shows all notifications, badge shows count of unread items including archived/muted/snoozed
	queryStr := "is:unread in:anywhere"

	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Total, nil
}

// calculateArchiveUnreadCount calculates the count of "new" (unread) notifications in the archive view.
func (s *Service) calculateArchiveUnreadCount(ctx context.Context, userID string) (int64, error) {
	// Archive view shows all archived notifications
	queryStr := "in:archive is:unread"

	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Total, nil
}

// calculateSnoozedUnreadCount calculates the count of "new" (unread) notifications in the snoozed view.
func (s *Service) calculateSnoozedUnreadCount(ctx context.Context, userID string) (int64, error) {
	// Snoozed view shows all snoozed notifications
	queryStr := "in:snoozed is:unread"

	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Total, nil
}

// calculateStarredUnreadCount calculates the count of "new" (unread) notifications in the starred view.
func (s *Service) calculateStarredUnreadCount(ctx context.Context, userID string) (int64, error) {
	// Starred view shows all starred notifications including archived/snoozed
	queryStr := "is:starred is:unread in:anywhere"

	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return 0, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Total, nil
}
