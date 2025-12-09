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
	"sort"
	"strings"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
)

// Error definitions
var (
	ErrNoNotificationIDs = errors.New("notifications: no notification ids provided")
)

// BulkUpdate performs a unified bulk operation on notifications.
// This method consolidates the many individual bulk operation methods into a single unified interface.
// It handles operations that can target notifications by IDs or by query string.
//
// Operations supported:
//   - mark-read, mark-unread
//   - archive, unarchive
//   - mute, unmute
//   - star, unstar
//   - unfilter
//   - snooze (requires SnoozedUntil in params), unsnooze
//
// Returns the number of notifications affected.
func (s *Service) BulkUpdate(
	ctx context.Context,
	userID string,
	op models.BulkOperationType,
	target models.BulkOperationTarget,
	params models.BulkUpdateParams,
) (int64, error) {
	// Validate target
	if len(target.IDs) == 0 && target.Query == "" {
		return 0, ErrNoNotificationIDs
	}
	if len(target.IDs) > 0 && target.Query != "" {
		return 0, errors.New("cannot specify both IDs and Query")
	}

	// Handle snooze-specific validation
	if op == models.BulkOpSnooze && params.SnoozedUntil == "" {
		return 0, errors.New("SnoozedUntil parameter is required for snooze operations")
	}

	// Execute based on target type
	if len(target.IDs) > 0 {
		return s.executeBulkUpdateByIDs(ctx, userID, op, target.IDs, params)
	}
	return s.executeBulkUpdateByQuery(ctx, userID, op, target.Query, params)
}

// executeBulkUpdateByIDs executes a bulk operation using notification IDs
func (s *Service) executeBulkUpdateByIDs(
	ctx context.Context,
	userID string,
	op models.BulkOperationType,
	githubIDs []string,
	params models.BulkUpdateParams,
) (int64, error) {
	if len(githubIDs) == 0 {
		return 0, ErrNoNotificationIDs
	}
	canonicalIDs := dedupeAndSort(githubIDs)

	switch op {
	case models.BulkOpMarkRead:
		return s.queries.BulkMarkNotificationsRead(ctx, userID, canonicalIDs)
	case models.BulkOpMarkUnread:
		return s.queries.BulkMarkNotificationsUnread(ctx, userID, canonicalIDs)
	case models.BulkOpArchive:
		return s.queries.BulkArchiveNotifications(ctx, userID, canonicalIDs)
	case models.BulkOpUnarchive:
		return s.queries.BulkUnarchiveNotifications(ctx, userID, canonicalIDs)
	case models.BulkOpMute:
		return s.queries.BulkMuteNotifications(ctx, userID, canonicalIDs)
	case models.BulkOpUnmute:
		return s.queries.BulkUnmuteNotifications(ctx, userID, canonicalIDs)
	case models.BulkOpStar:
		return s.queries.BulkStarNotifications(ctx, userID, canonicalIDs)
	case models.BulkOpUnstar:
		return s.queries.BulkUnstarNotifications(ctx, userID, canonicalIDs)
	case models.BulkOpUnfilter:
		return s.queries.BulkMarkNotificationsUnfiltered(ctx, userID, canonicalIDs)
	case models.BulkOpSnooze:
		t, err := time.Parse(time.RFC3339, params.SnoozedUntil)
		if err != nil {
			return 0, errors.Join(ErrInvalidSnoozedUntilFormat, err)
		}
		return s.queries.BulkSnoozeNotifications(ctx, userID, db.BulkSnoozeNotificationsParams{
			GithubIDs:    canonicalIDs,
			SnoozedUntil: sql.NullTime{Time: t, Valid: true},
		})
	case models.BulkOpUnsnooze:
		return s.queries.BulkUnsnoozeNotifications(ctx, userID, canonicalIDs)
	default:
		return 0, errors.New("unknown bulk operation type: " + string(op))
	}
}

// executeBulkUpdateByQuery executes a bulk operation using a query string
func (s *Service) executeBulkUpdateByQuery(
	ctx context.Context,
	userID string,
	op models.BulkOperationType,
	queryStr string,
	params models.BulkUpdateParams,
) (int64, error) {
	dbQuery, err := query.BuildQuery(queryStr, 0, 0)
	if err != nil {
		return 0, errors.Join(ErrFailedToBuildQuery, err)
	}

	switch op {
	case models.BulkOpMarkRead:
		return s.queries.BulkMarkNotificationsReadByQuery(ctx, userID, dbQuery)
	case models.BulkOpMarkUnread:
		return s.queries.BulkMarkNotificationsUnreadByQuery(ctx, userID, dbQuery)
	case models.BulkOpArchive:
		return s.queries.BulkArchiveNotificationsByQuery(ctx, userID, dbQuery)
	case models.BulkOpUnarchive:
		return s.queries.BulkUnarchiveNotificationsByQuery(ctx, userID, dbQuery)
	case models.BulkOpMute:
		return s.queries.BulkMuteNotificationsByQuery(ctx, userID, dbQuery)
	case models.BulkOpUnmute:
		return s.queries.BulkUnmuteNotificationsByQuery(ctx, userID, dbQuery)
	case models.BulkOpStar:
		return s.queries.BulkStarNotificationsByQuery(ctx, userID, dbQuery)
	case models.BulkOpUnstar:
		return s.queries.BulkUnstarNotificationsByQuery(ctx, userID, dbQuery)
	case models.BulkOpUnfilter:
		// Special case: unfilter needs to fetch notifications first, then call the ID-based method
		result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
		if err != nil {
			return 0, errors.Join(ErrFailedToListNotifications, err)
		}
		if len(result.Notifications) == 0 {
			return 0, nil
		}
		githubIDs := make([]string, 0, len(result.Notifications))
		for _, n := range result.Notifications {
			githubIDs = append(githubIDs, n.GithubID)
		}
		return s.executeBulkUpdateByIDs(
			ctx,
			userID,
			models.BulkOpUnfilter,
			githubIDs,
			models.BulkUpdateParams{},
		)
	case models.BulkOpSnooze:
		t, err := time.Parse(time.RFC3339, params.SnoozedUntil)
		if err != nil {
			return 0, errors.Join(ErrInvalidSnoozedUntilFormat, err)
		}
		return s.queries.BulkSnoozeNotificationsByQuery(
			ctx,
			userID,
			db.BulkSnoozeNotificationsByQueryParams{
				Query:        dbQuery,
				SnoozedUntil: sql.NullTime{Time: t, Valid: true},
			},
		)
	case models.BulkOpUnsnooze:
		return s.queries.BulkUnsnoozeNotificationsByQuery(ctx, userID, dbQuery)
	default:
		return 0, errors.New("unknown bulk operation type: " + string(op))
	}
}

// BulkAssignTag assigns a tag to multiple notifications
func (s *Service) BulkAssignTag(
	ctx context.Context,
	userID string,
	notifications []db.Notification,
	tagID string,
) (int, error) {
	count := 0
	for _, notification := range notifications {
		_, err := s.queries.AssignTagToEntity(ctx, userID, db.AssignTagToEntityParams{
			TagID:      tagID,
			EntityType: "notification",
			EntityID:   notification.ID,
		})
		if err != nil {
			// Check if it's a duplicate key error (tag already assigned)
			if models.IsUniqueViolation(err) {
				// Tag already assigned, count as success
				count++
			}
			continue
		}

		count++
	}
	return count, nil
}

// BulkRemoveTag removes a tag from multiple notifications
func (s *Service) BulkRemoveTag(
	ctx context.Context,
	userID string,
	notifications []db.Notification,
	tagID string,
) (int, error) {
	count := 0
	for _, notification := range notifications {
		err := s.queries.RemoveTagAssignment(ctx, userID, db.RemoveTagAssignmentParams{
			TagID:      tagID,
			EntityType: "notification",
			EntityID:   notification.ID,
		})
		if err != nil {
			continue
		}

		count++
	}
	return count, nil
}

// dedupeAndSort removes duplicates and sorts notification IDs
func dedupeAndSort(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
