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

package notification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Error definitions
var (
	ErrInvalidSnoozedUntilFormat = errors.New("invalid snoozedUntil format")
)

// MarkNotificationRead marks a notification as read.
func (s *Service) MarkNotificationRead(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.MarkNotificationRead(ctx, userID, githubID)
}

// MarkNotificationUnread marks a notification as unread.
func (s *Service) MarkNotificationUnread(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.MarkNotificationUnread(ctx, userID, githubID)
}

// ArchiveNotification archives a notification.
func (s *Service) ArchiveNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.ArchiveNotification(ctx, userID, githubID)
}

// UnarchiveNotification unarchives a notification.
func (s *Service) UnarchiveNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.UnarchiveNotification(ctx, userID, githubID)
}

// MuteNotification mutes a notification.
func (s *Service) MuteNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.MuteNotification(ctx, userID, githubID)
}

// UnmuteNotification unmutes a notification.
func (s *Service) UnmuteNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.UnmuteNotification(ctx, userID, githubID)
}

// SnoozeNotification snoozes a notification until a specified time.
func (s *Service) SnoozeNotification(
	ctx context.Context,
	userID, githubID, snoozedUntil string,
) (db.Notification, error) {
	// Parse the time string
	t, err := time.Parse(time.RFC3339, snoozedUntil)
	if err != nil {
		return db.Notification{}, errors.Join(ErrInvalidSnoozedUntilFormat, err)
	}

	return s.queries.SnoozeNotification(ctx, userID, db.SnoozeNotificationParams{
		GithubID:     githubID,
		SnoozedUntil: sql.NullTime{Time: t, Valid: true},
	})
}

// UnsnoozeNotification clears the snooze on a notification.
func (s *Service) UnsnoozeNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.UnsnoozeNotification(ctx, userID, githubID)
}

// StarNotification stars a notification.
func (s *Service) StarNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.StarNotification(ctx, userID, githubID)
}

// UnstarNotification unstars a notification.
func (s *Service) UnstarNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.UnstarNotification(ctx, userID, githubID)
}

// UnfilterNotification unfilters a notification (moves it to inbox).
func (s *Service) UnfilterNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	return s.queries.MarkNotificationUnfiltered(ctx, userID, githubID)
}
