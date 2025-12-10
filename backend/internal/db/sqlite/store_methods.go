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
	"strings"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// --- Notification methods ---

// GetNotificationByGithubID gets a notification by GitHub ID
func (s *Store) GetNotificationByGithubID(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.GetNotificationByGithubID(ctx, GetNotificationByGithubIDParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// GetNotificationByID gets a notification by ID
func (s *Store) GetNotificationByID(
	ctx context.Context,
	userID string,
	id int64,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.GetNotificationByID(ctx, GetNotificationByIDParams{
			UserID: userID,
			ID:     id,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// ListNotificationsFromQuery lists notifications from a query
func (s *Store) ListNotificationsFromQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (db.ListNotificationsFromQueryResult, error) {
	return listNotificationsFromQuery(ctx, s, userID, query)
}

// MarkNotificationRead marks a notification as read
func (s *Store) MarkNotificationRead(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.MarkNotificationRead(ctx, MarkNotificationReadParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// MarkNotificationUnread marks a notification as unread
func (s *Store) MarkNotificationUnread(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.MarkNotificationUnread(ctx, MarkNotificationUnreadParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// ArchiveNotification archives a notification
func (s *Store) ArchiveNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.ArchiveNotification(ctx, ArchiveNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// UnarchiveNotification unarchives a notification
func (s *Store) UnarchiveNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.UnarchiveNotification(ctx, UnarchiveNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// MuteNotification mutes a notification
func (s *Store) MuteNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.MuteNotification(ctx, MuteNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// UnmuteNotification unmutes a notification
func (s *Store) UnmuteNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.UnmuteNotification(ctx, UnmuteNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// SnoozeNotification snoozes a notification
func (s *Store) SnoozeNotification(
	ctx context.Context,
	userID string,
	arg db.SnoozeNotificationParams,
) (db.Notification, error) {
	snoozedUntil := formatNullTime(arg.SnoozedUntil)
	effectiveSortDate := ""
	if snoozedUntil.Valid {
		effectiveSortDate = snoozedUntil.String
	}
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.SnoozeNotification(ctx, SnoozeNotificationParams{
			SnoozedUntil:      snoozedUntil,
			EffectiveSortDate: effectiveSortDate,
			UserID:            userID,
			GithubID:          arg.GithubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// UnsnoozeNotification marks notifications as unsnoozed.
func (s *Store) UnsnoozeNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.UnsnoozeNotification(ctx, UnsnoozeNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// StarNotification marks notifications as starred.
func (s *Store) StarNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.StarNotification(ctx, StarNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// UnstarNotification marks notifications as unstarred.
func (s *Store) UnstarNotification(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.UnstarNotification(ctx, UnstarNotificationParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// MarkNotificationFiltered marks notifications as filtered.
func (s *Store) MarkNotificationFiltered(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.MarkNotificationFiltered(ctx, MarkNotificationFilteredParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// MarkNotificationUnfiltered marks notifications as unfiltered.
func (s *Store) MarkNotificationUnfiltered(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.MarkNotificationUnfiltered(ctx, MarkNotificationUnfilteredParams{
			UserID:   userID,
			GithubID: githubID,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}
	return s.toDBNotification(ctx, userID, n), nil
}

// --- Bulk notification methods ---

// BulkSnoozeNotifications marks notifications as snoozed.
func (s *Store) BulkSnoozeNotifications(
	ctx context.Context,
	userID string,
	arg db.BulkSnoozeNotificationsParams,
) (int64, error) {
	if len(arg.GithubIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(arg.GithubIDs))
	snoozedUntil := formatNullTime(arg.SnoozedUntil)
	// Args: snoozed_until, effective_sort_date (same value), user_id, then github_ids
	args := make([]interface{}, 0, len(arg.GithubIDs)+3)
	args = append(args, snoozedUntil, snoozedUntil, userID)
	for i, id := range arg.GithubIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	//nolint:gosec // G201: SQL string formatting is safe - placeholders are generated from slice length
	query := fmt.Sprintf(
		`UPDATE notifications SET snoozed_until = ?, `+
			`snoozed_at = strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', 'now'), `+
			`effective_sort_date = ? WHERE user_id = ? AND github_id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	var result sql.Result
	err := db.RetryVoidOnBusy(ctx, func() error {
		var execErr error
		result, execErr = s.dbConn.ExecContext(ctx, query, args...)
		return execErr
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// BulkMarkNotificationsRead marks notifications as read.
func (s *Store) BulkMarkNotificationsRead(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "is_read = 1", githubIDs)
}

// BulkMarkNotificationsUnread marks notifications as unread.
func (s *Store) BulkMarkNotificationsUnread(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "is_read = 0", githubIDs)
}

// BulkArchiveNotifications marks notifications as archived.
func (s *Store) BulkArchiveNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(
		ctx,
		userID,
		"archived = 1, snoozed_until = NULL, snoozed_at = NULL, "+
			"effective_sort_date = COALESCE(github_updated_at, imported_at)",
		githubIDs,
	)
}

// BulkUnarchiveNotifications marks notifications as unarchived.
func (s *Store) BulkUnarchiveNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "archived = 0", githubIDs)
}

// BulkUnsnoozeNotifications marks notifications as unsnoozed.
func (s *Store) BulkUnsnoozeNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(
		ctx,
		userID,
		"snoozed_until = NULL, snoozed_at = NULL, effective_sort_date = COALESCE(github_updated_at, imported_at)",
		githubIDs,
	)
}

// BulkMuteNotifications marks notifications as muted.
func (s *Store) BulkMuteNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(
		ctx,
		userID,
		"muted = 1, snoozed_until = NULL, snoozed_at = NULL, effective_sort_date = COALESCE(github_updated_at, imported_at)",
		githubIDs,
	)
}

// BulkUnmuteNotifications marks notifications as unmuted.
func (s *Store) BulkUnmuteNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "muted = 0", githubIDs)
}

// BulkStarNotifications marks notifications as starred.
func (s *Store) BulkStarNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "starred = 1", githubIDs)
}

// BulkUnstarNotifications marks notifications as unstarred.
func (s *Store) BulkUnstarNotifications(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "starred = 0", githubIDs)
}

// BulkMarkNotificationsUnfiltered marks notifications as unfiltered.
func (s *Store) BulkMarkNotificationsUnfiltered(
	ctx context.Context,
	userID string,
	githubIDs []string,
) (int64, error) {
	return s.bulkUpdate(ctx, userID, "filtered = 0", githubIDs)
}

// bulkUpdate performs a bulk update on notifications
func (s *Store) bulkUpdate(
	ctx context.Context,
	userID string,
	setClause string,
	githubIDs []string,
) (int64, error) {
	if len(githubIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(githubIDs))
	args := make([]interface{}, 0, len(githubIDs)+1)
	args = append(args, userID)
	for i, id := range githubIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	//nolint:gosec // G201: SQL string formatting is safe - setClause and placeholders are controlled
	query := fmt.Sprintf(
		`UPDATE notifications SET %s WHERE user_id = ? AND github_id IN (%s)`,
		setClause,
		strings.Join(placeholders, ","),
	)
	var result sql.Result
	err := db.RetryVoidOnBusy(ctx, func() error {
		var execErr error
		result, execErr = s.dbConn.ExecContext(ctx, query, args...)
		return execErr
	})
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Bulk by query methods (implemented in query_executor.go) ---

// BulkMarkNotificationsReadByQuery marks notifications as read by query
func (s *Store) BulkMarkNotificationsReadByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(ctx, s, userID, "is_read = 1", query)
}

// BulkMarkNotificationsUnreadByQuery marks notifications as unread by query
func (s *Store) BulkMarkNotificationsUnreadByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(ctx, s, userID, "is_read = 0", query)
}

// BulkArchiveNotificationsByQuery marks notifications as archived by query
func (s *Store) BulkArchiveNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(
		ctx,
		s,
		userID,
		"archived = 1, snoozed_until = NULL, snoozed_at = NULL, "+
			"effective_sort_date = COALESCE(github_updated_at, imported_at)",
		query,
	)
}

// BulkUnarchiveNotificationsByQuery marks notifications as unarchived by query
func (s *Store) BulkUnarchiveNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(ctx, s, userID, "archived = 0", query)
}

// BulkSnoozeNotificationsByQuery marks notifications as snoozed by query
func (s *Store) BulkSnoozeNotificationsByQuery(
	ctx context.Context,
	userID string,
	arg db.BulkSnoozeNotificationsByQueryParams,
) (int64, error) {
	return bulkSnoozeByQuery(ctx, s, userID, arg)
}

// BulkUnsnoozeNotificationsByQuery marks notifications as unsnoozed by query
func (s *Store) BulkUnsnoozeNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(
		ctx,
		s,
		userID,
		"snoozed_until = NULL, snoozed_at = NULL, effective_sort_date = COALESCE(github_updated_at, imported_at)",
		query,
	)
}

// BulkMuteNotificationsByQuery marks notifications as muted by query
func (s *Store) BulkMuteNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(
		ctx,
		s,
		userID,
		"muted = 1, snoozed_until = NULL, snoozed_at = NULL, effective_sort_date = COALESCE(github_updated_at, imported_at)",
		query,
	)
}

// BulkUnmuteNotificationsByQuery marks notifications as unmuted by query
func (s *Store) BulkUnmuteNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(ctx, s, userID, "muted = 0", query)
}

// BulkStarNotificationsByQuery marks notifications as starred by query
func (s *Store) BulkStarNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(ctx, s, userID, "starred = 1", query)
}

// BulkUnstarNotificationsByQuery marks notifications as unstarred by query
func (s *Store) BulkUnstarNotificationsByQuery(
	ctx context.Context,
	userID string,
	query db.NotificationQuery,
) (int64, error) {
	return bulkUpdateByQuery(ctx, s, userID, "starred = 0", query)
}

// GetTag gets a tag by ID
func (s *Store) GetTag(ctx context.Context, userID, id string) (db.Tag, error) {
	t, err := db.RetryOnBusy(ctx, func() (Tag, error) {
		return s.q.GetTag(ctx, GetTagParams{
			UserID: userID,
			ID:     id,
		})
	})
	if err != nil {
		return db.Tag{}, err
	}
	return toDBTag(t), nil
}

// GetTagByName gets a tag by name
func (s *Store) GetTagByName(ctx context.Context, userID, name string) (db.Tag, error) {
	t, err := db.RetryOnBusy(ctx, func() (Tag, error) {
		return s.q.GetTagByName(ctx, GetTagByNameParams{
			UserID: userID,
			Name:   name,
		})
	})
	if err != nil {
		return db.Tag{}, err
	}
	return toDBTag(t), nil
}

// ListAllTags lists all tags
func (s *Store) ListAllTags(ctx context.Context, userID string) ([]db.Tag, error) {
	tags, err := db.RetryOnBusy(ctx, func() ([]Tag, error) {
		return s.q.ListAllTags(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.Tag, len(tags))
	for i, t := range tags {
		result[i] = toDBTag(t)
	}
	return result, nil
}

// UpsertTag upserts a tag
func (s *Store) UpsertTag(
	ctx context.Context,
	userID string,
	arg db.UpsertTagParams,
) (db.Tag, error) {
	t, err := db.RetryOnBusy(ctx, func() (Tag, error) {
		return s.q.UpsertTag(ctx, UpsertTagParams{
			UserID:      userID,
			Name:        arg.Name,
			Slug:        arg.Slug,
			Color:       arg.Color,
			Description: arg.Description,
		})
	})
	if err != nil {
		return db.Tag{}, err
	}
	return toDBTag(t), nil
}

// UpdateTag updates a tag
func (s *Store) UpdateTag(
	ctx context.Context,
	userID string,
	arg db.UpdateTagParams,
) (db.Tag, error) {
	t, err := db.RetryOnBusy(ctx, func() (Tag, error) {
		return s.q.UpdateTag(ctx, UpdateTagParams{
			UserID:      userID,
			ID:          arg.ID,
			Name:        arg.Name,
			Slug:        arg.Slug,
			Color:       arg.Color,
			Description: arg.Description,
		})
	})
	if err != nil {
		return db.Tag{}, err
	}
	return toDBTag(t), nil
}

// DeleteTag deletes a tag
func (s *Store) DeleteTag(ctx context.Context, userID, id string) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.DeleteTag(ctx, DeleteTagParams{
			UserID: userID,
			ID:     id,
		})
	})
}

// UpdateTagDisplayOrder updates the display order of a tag
func (s *Store) UpdateTagDisplayOrder(
	ctx context.Context,
	userID string,
	arg db.UpdateTagDisplayOrderParams,
) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.UpdateTagDisplayOrder(ctx, UpdateTagDisplayOrderParams{
			UserID:       userID,
			ID:           arg.ID,
			DisplayOrder: int64(arg.DisplayOrder),
		})
	})
}

// ListTagsForEntity lists tags for an entity
func (s *Store) ListTagsForEntity(
	ctx context.Context,
	userID string,
	arg db.ListTagsForEntityParams,
) ([]db.Tag, error) {
	tags, err := db.RetryOnBusy(ctx, func() ([]Tag, error) {
		return s.q.ListTagsForEntity(ctx, ListTagsForEntityParams{
			UserID:     userID,
			EntityType: arg.EntityType,
			EntityID:   arg.EntityID,
		})
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.Tag, len(tags))
	for i, t := range tags {
		result[i] = toDBTag(t)
	}
	return result, nil
}

// AssignTagToEntity assigns a tag to an entity
func (s *Store) AssignTagToEntity(
	ctx context.Context,
	userID string,
	arg db.AssignTagToEntityParams,
) (db.TagAssignment, error) {
	ta, err := db.RetryOnBusy(ctx, func() (TagAssignment, error) {
		return s.q.AssignTagToEntity(ctx, AssignTagToEntityParams{
			UserID:     userID,
			TagID:      arg.TagID,
			EntityType: arg.EntityType,
			EntityID:   arg.EntityID,
		})
	})
	if err != nil {
		return db.TagAssignment{}, err
	}
	return toDBTagAssignment(ta), nil
}

// RemoveTagAssignment removes a tag assignment
func (s *Store) RemoveTagAssignment(
	ctx context.Context,
	userID string,
	arg db.RemoveTagAssignmentParams,
) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.RemoveTagAssignment(ctx, RemoveTagAssignmentParams{
			UserID:     userID,
			TagID:      arg.TagID,
			EntityType: arg.EntityType,
			EntityID:   arg.EntityID,
		})
	})
}

// GetView gets a view by ID
func (s *Store) GetView(ctx context.Context, userID, id string) (db.View, error) {
	v, err := db.RetryOnBusy(ctx, func() (View, error) {
		return s.q.GetView(ctx, GetViewParams{
			UserID: userID,
			ID:     id,
		})
	})
	if err != nil {
		return db.View{}, err
	}
	return toDBView(v), nil
}

// ListViews lists all views
func (s *Store) ListViews(ctx context.Context, userID string) ([]db.View, error) {
	views, err := db.RetryOnBusy(ctx, func() ([]View, error) {
		return s.q.ListViews(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.View, len(views))
	for i, v := range views {
		result[i] = toDBView(v)
	}
	return result, nil
}

// CreateView creates a new view
func (s *Store) CreateView(
	ctx context.Context,
	userID string,
	arg db.CreateViewParams,
) (db.View, error) {
	// Handle IsDefault which can be interface{} (for compatibility with db types)
	isDefault := int64(0)
	switch v := arg.IsDefault.(type) {
	case bool:
		if v {
			isDefault = 1
		}
	case int:
		isDefault = int64(v)
	case int64:
		isDefault = v
	}

	v, err := db.RetryOnBusy(ctx, func() (View, error) {
		return s.q.CreateView(ctx, CreateViewParams{
			UserID:       userID,
			Name:         arg.Name,
			Slug:         arg.Slug,
			Description:  arg.Description,
			IsDefault:    isDefault,
			Icon:         arg.Icon,
			Query:        arg.Query,
			DisplayOrder: 0,
		})
	})
	if err != nil {
		return db.View{}, err
	}
	return toDBView(v), nil
}

// UpdateView updates a view
func (s *Store) UpdateView(
	ctx context.Context,
	userID string,
	arg db.UpdateViewParams,
) (db.View, error) {
	// SQLite UpdateView expects non-nullable strings for name and slug
	name := ""
	if arg.Name.Valid {
		name = arg.Name.String
	}
	slug := ""
	if arg.Slug.Valid {
		slug = arg.Slug.String
	}

	var isDefault int64
	if arg.IsDefault.Valid {
		if arg.IsDefault.Bool {
			isDefault = 1
		}
	}

	v, err := db.RetryOnBusy(ctx, func() (View, error) {
		return s.q.UpdateView(ctx, UpdateViewParams{
			UserID:      userID,
			ID:          arg.ID,
			Name:        name,
			Slug:        slug,
			Description: arg.Description,
			Icon:        arg.Icon,
			Query:       arg.Query,
			IsDefault:   isDefault,
		})
	})
	if err != nil {
		return db.View{}, err
	}
	return toDBView(v), nil
}

// DeleteView deletes a view
func (s *Store) DeleteView(ctx context.Context, userID, id string) (int64, error) {
	return db.RetryOnBusy(ctx, func() (int64, error) {
		return s.q.DeleteView(ctx, DeleteViewParams{
			UserID: userID,
			ID:     id,
		})
	})
}

// UpdateViewOrder updates the display order of a view
func (s *Store) UpdateViewOrder(
	ctx context.Context,
	userID string,
	arg db.UpdateViewOrderParams,
) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.UpdateViewOrder(ctx, UpdateViewOrderParams{
			UserID:       userID,
			ID:           arg.ID,
			DisplayOrder: int64(arg.DisplayOrder),
		})
	})
}

// GetRulesByViewID gets rules by view ID
func (s *Store) GetRulesByViewID(
	ctx context.Context,
	userID string,
	viewID sql.NullString,
) ([]db.Rule, error) {
	rules, err := db.RetryOnBusy(ctx, func() ([]Rule, error) {
		return s.q.GetRulesByViewID(ctx, GetRulesByViewIDParams{
			UserID: userID,
			ViewID: viewID,
		})
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.Rule, len(rules))
	for i, r := range rules {
		result[i] = toDBRule(r)
	}
	return result, nil
}

// --- Rule methods ---

// GetRule gets a rule by ID
func (s *Store) GetRule(ctx context.Context, userID, id string) (db.Rule, error) {
	r, err := db.RetryOnBusy(ctx, func() (Rule, error) {
		return s.q.GetRule(ctx, GetRuleParams{
			UserID: userID,
			ID:     id,
		})
	})
	if err != nil {
		return db.Rule{}, err
	}
	return toDBRule(r), nil
}

// ListRules lists all rules
func (s *Store) ListRules(ctx context.Context, userID string) ([]db.Rule, error) {
	rules, err := db.RetryOnBusy(ctx, func() ([]Rule, error) {
		return s.q.ListRules(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.Rule, len(rules))
	for i, r := range rules {
		result[i] = toDBRule(r)
	}
	return result, nil
}

// ListEnabledRulesOrdered lists all enabled rules ordered by display order
func (s *Store) ListEnabledRulesOrdered(ctx context.Context, userID string) ([]db.Rule, error) {
	rules, err := db.RetryOnBusy(ctx, func() ([]Rule, error) {
		return s.q.ListEnabledRulesOrdered(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.Rule, len(rules))
	for i, r := range rules {
		result[i] = toDBRule(r)
	}
	return result, nil
}

// CreateRule creates a new rule
func (s *Store) CreateRule(
	ctx context.Context,
	userID string,
	arg db.CreateRuleParams,
) (db.Rule, error) {
	r, err := db.RetryOnBusy(ctx, func() (Rule, error) {
		return s.q.CreateRule(ctx, CreateRuleParams{
			UserID:       userID,
			Name:         arg.Name,
			Description:  arg.Description,
			Query:        arg.Query,
			ViewID:       arg.ViewID,
			Enabled:      boolToInt64(arg.Enabled),
			Actions:      string(arg.Actions),
			DisplayOrder: int64(arg.DisplayOrder),
		})
	})
	if err != nil {
		return db.Rule{}, err
	}
	return toDBRule(r), nil
}

// UpdateRule updates a rule
func (s *Store) UpdateRule(
	ctx context.Context,
	userID string,
	arg db.UpdateRuleParams,
) (db.Rule, error) {
	// Handle type conversions
	name := ""
	if arg.Name.Valid {
		name = arg.Name.String
	}

	var enabled int64
	if arg.Enabled.Valid && arg.Enabled.Bool {
		enabled = 1
	}

	actions := ""
	if arg.Actions.Valid {
		actions = string(arg.Actions.RawMessage)
	}

	// Handle clearing fields
	query := arg.Query
	if arg.ClearQuery.Valid && arg.ClearQuery.Bool {
		query = sql.NullString{}
	}

	viewID := arg.ViewID
	if arg.ClearViewID.Valid && arg.ClearViewID.Bool {
		viewID = sql.NullString{}
	}

	r, err := db.RetryOnBusy(ctx, func() (Rule, error) {
		return s.q.UpdateRule(ctx, UpdateRuleParams{
			UserID:      userID,
			ID:          arg.ID,
			Name:        name,
			Description: arg.Description,
			Query:       query,
			ViewID:      viewID,
			Enabled:     enabled,
			Actions:     actions,
		})
	})
	if err != nil {
		return db.Rule{}, err
	}
	return toDBRule(r), nil
}

// DeleteRule deletes a rule
func (s *Store) DeleteRule(ctx context.Context, userID, id string) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.DeleteRule(ctx, DeleteRuleParams{
			UserID: userID,
			ID:     id,
		})
	})
}

// UpdateRuleOrder updates the order of a rule
func (s *Store) UpdateRuleOrder(
	ctx context.Context,
	userID string,
	arg db.UpdateRuleOrderParams,
) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.UpdateRuleOrder(ctx, UpdateRuleOrderParams{
			UserID:       userID,
			ID:           arg.ID,
			DisplayOrder: int64(arg.DisplayOrder),
		})
	})
}

// --- Repository methods ---

// GetRepositoryByID gets a repository by ID
func (s *Store) GetRepositoryByID(
	ctx context.Context,
	userID string,
	id int64,
) (db.Repository, error) {
	r, err := db.RetryOnBusy(ctx, func() (Repository, error) {
		return s.q.GetRepositoryByID(ctx, GetRepositoryByIDParams{
			UserID: userID,
			ID:     id,
		})
	})
	if err != nil {
		return db.Repository{}, err
	}
	return toDBRepository(r), nil
}

// ListRepositories lists all repositories
func (s *Store) ListRepositories(ctx context.Context, userID string) ([]db.Repository, error) {
	repos, err := db.RetryOnBusy(ctx, func() ([]Repository, error) {
		return s.q.ListRepositories(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	result := make([]db.Repository, len(repos))
	for i, r := range repos {
		result[i] = toDBRepository(r)
	}
	return result, nil
}

// UpsertRepository upserts a repository
func (s *Store) UpsertRepository(
	ctx context.Context,
	userID string,
	arg db.UpsertRepositoryParams,
) (db.Repository, error) {
	// Handle Archived which can be interface{} or sql.NullBool
	var archived sql.NullInt64
	switch v := arg.Archived.(type) {
	case sql.NullBool:
		archived = fromNullBool(v)
	case bool:
		if v {
			archived = sql.NullInt64{Int64: 1, Valid: true}
		} else {
			archived = sql.NullInt64{Int64: 0, Valid: true}
		}
	}

	r, err := db.RetryOnBusy(ctx, func() (Repository, error) {
		return s.q.UpsertRepository(ctx, UpsertRepositoryParams{
			UserID:         userID,
			GithubID:       arg.GithubID,
			NodeID:         arg.NodeID,
			Name:           arg.Name,
			FullName:       arg.FullName,
			OwnerLogin:     arg.OwnerLogin,
			OwnerID:        arg.OwnerID,
			Private:        fromNullBool(arg.Private),
			Description:    arg.Description,
			HtmlUrl:        arg.HTMLURL,
			Fork:           fromNullBool(arg.Fork),
			Visibility:     arg.Visibility,
			DefaultBranch:  arg.DefaultBranch,
			Archived:       archived,
			Disabled:       fromNullBool(arg.Disabled),
			PushedAt:       formatNullTime(arg.PushedAt),
			CreatedAt:      formatNullTime(arg.CreatedAt),
			UpdatedAt:      formatNullTime(arg.UpdatedAt),
			Raw:            fromNullRawMessage(arg.Raw),
			OwnerAvatarUrl: arg.OwnerAvatarURL,
			OwnerHtmlUrl:   arg.OwnerHTMLURL,
		})
	})
	if err != nil {
		return db.Repository{}, err
	}
	return toDBRepository(r), nil
}

// --- Pull Request methods ---

// UpsertPullRequest upserts a pull request
func (s *Store) UpsertPullRequest(
	ctx context.Context,
	userID string,
	arg db.UpsertPullRequestParams,
) (db.PullRequest, error) {
	pr, err := db.RetryOnBusy(ctx, func() (PullRequest, error) {
		return s.q.UpsertPullRequest(ctx, UpsertPullRequestParams{
			UserID:       userID,
			RepositoryID: arg.RepositoryID,
			GithubID:     arg.GithubID,
			NodeID:       arg.NodeID,
			Number:       int64(arg.Number),
			Title:        arg.Title,
			State:        arg.State,
			Draft:        fromNullBool(arg.Draft),
			Merged:       fromNullBool(arg.Merged),
			AuthorLogin:  arg.AuthorLogin,
			AuthorID:     arg.AuthorID,
			CreatedAt:    formatNullTime(arg.CreatedAt),
			UpdatedAt:    formatNullTime(arg.UpdatedAt),
			ClosedAt:     formatNullTime(arg.ClosedAt),
			MergedAt:     formatNullTime(arg.MergedAt),
			Raw:          fromNullRawMessage(arg.Raw),
		})
	})
	if err != nil {
		return db.PullRequest{}, err
	}
	return toDBPullRequest(pr), nil
}

// --- Sync State methods ---

// GetSyncState gets a sync state
func (s *Store) GetSyncState(ctx context.Context, userID string) (db.GetSyncStateRow, error) {
	ss, err := db.RetryOnBusy(ctx, func() (SyncState, error) {
		return s.q.GetSyncState(ctx, userID)
	})
	if err != nil {
		return db.GetSyncStateRow{}, err
	}
	return toDBGetSyncStateRow(ss), nil
}

// UpsertSyncState upserts a sync state
func (s *Store) UpsertSyncState(
	ctx context.Context,
	userID string,
	arg db.UpsertSyncStateParams,
) (db.UpsertSyncStateRow, error) {
	ss, err := db.RetryOnBusy(ctx, func() (SyncState, error) {
		return s.q.UpsertSyncState(ctx, UpsertSyncStateParams{
			UserID:                     userID,
			LastSuccessfulPoll:         formatNullTime(arg.LastSuccessfulPoll),
			LastNotificationEtag:       arg.LastNotificationEtag,
			LatestNotificationAt:       formatNullTime(arg.LatestNotificationAt),
			InitialSyncCompletedAt:     formatNullTime(arg.InitialSyncCompletedAt),
			OldestNotificationSyncedAt: formatNullTime(arg.OldestNotificationSyncedAt),
		})
	})
	if err != nil {
		return db.UpsertSyncStateRow{}, err
	}
	return toDBUpsertSyncStateRow(ss), nil
}

// --- Notification upsert/update methods ---

// UpsertNotification upserts a notification
func (s *Store) UpsertNotification(
	ctx context.Context,
	userID string,
	arg db.UpsertNotificationParams,
) (db.Notification, error) {
	// Check if notification already exists to implement smart status updates
	var existingNotif *Notification
	existing, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.GetNotificationByGithubID(ctx, GetNotificationByGithubIDParams{
			UserID:   userID,
			GithubID: arg.GithubID,
		})
	})
	if err == nil {
		existingNotif = &existing
	}

	var effectiveSortDate interface{}
	if arg.GithubUpdatedAt.Valid {
		effectiveSortDate = formatTime(arg.GithubUpdatedAt.Time)
	}

	n, err := db.RetryOnBusy(ctx, func() (Notification, error) {
		return s.q.UpsertNotification(ctx, UpsertNotificationParams{
			UserID:                  userID,
			GithubID:                arg.GithubID,
			RepositoryID:            arg.RepositoryID,
			PullRequestID:           arg.PullRequestID,
			SubjectType:             arg.SubjectType,
			SubjectTitle:            arg.SubjectTitle,
			SubjectUrl:              arg.SubjectURL,
			SubjectLatestCommentUrl: arg.SubjectLatestCommentURL,
			Reason:                  arg.Reason,
			GithubUnread:            fromNullBool(arg.GithubUnread),
			GithubUpdatedAt:         formatNullTime(arg.GithubUpdatedAt),
			GithubLastReadAt:        formatNullTime(arg.GithubLastReadAt),
			GithubUrl:               arg.GithubURL,
			GithubSubscriptionUrl:   arg.GithubSubscriptionURL,
			Payload:                 fromNullRawMessage(arg.Payload),
			SubjectRaw:              fromNullRawMessage(arg.SubjectRaw),
			SubjectFetchedAt:        formatNullTime(arg.SubjectFetchedAt),
			AuthorLogin:             arg.AuthorLogin,
			AuthorID:                arg.AuthorID,
			SubjectNumber:           fromNullInt32(arg.SubjectNumber),
			SubjectState:            arg.SubjectState,
			SubjectMerged:           fromNullBool(arg.SubjectMerged),
			SubjectStateReason:      arg.SubjectStateReason,
			EffectiveSortDate:       effectiveSortDate,
		})
	})
	if err != nil {
		return db.Notification{}, err
	}

	// Apply smart status updates
	if existingNotif != nil && s.shouldResetStatusOnSync(existingNotif, arg) {
		if resetErr := db.RetryVoidOnBusy(ctx, func() error {
			return s.q.ResetNotificationStatusOnSync(ctx, ResetNotificationStatusOnSyncParams{
				UserID:   userID,
				GithubID: arg.GithubID,
			})
		}); resetErr != nil {
			return db.Notification{},
				fmt.Errorf("failed to reset notification status: %w", resetErr)
		}
		n, err = db.RetryOnBusy(ctx, func() (Notification, error) {
			return s.q.GetNotificationByGithubID(ctx, GetNotificationByGithubIDParams{
				UserID:   userID,
				GithubID: arg.GithubID,
			})
		})
		if err != nil {
			return db.Notification{}, fmt.Errorf("failed to fetch updated notification: %w", err)
		}
	}

	return s.toDBNotification(ctx, userID, n), nil
}

// shouldResetStatusOnSync checks if the status of a notification should be reset on sync
func (s *Store) shouldResetStatusOnSync(
	existing *Notification,
	arg db.UpsertNotificationParams,
) bool {
	if existing.Muted != 0 {
		return false
	}
	if !arg.GithubUpdatedAt.Valid {
		return false
	}
	if !existing.GithubUpdatedAt.Valid {
		return true
	}
	newTime := formatTime(arg.GithubUpdatedAt.Time)
	return existing.GithubUpdatedAt.String != newTime
}

// UpdateNotificationSubject updates the subject of a notification
func (s *Store) UpdateNotificationSubject(
	ctx context.Context,
	userID string,
	arg db.UpdateNotificationSubjectParams,
) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		return s.q.UpdateNotificationSubject(ctx, UpdateNotificationSubjectParams{
			UserID:             userID,
			GithubID:           arg.GithubID,
			SubjectRaw:         fromNullRawMessage(arg.SubjectRaw),
			SubjectFetchedAt:   formatNullTime(arg.SubjectFetchedAt),
			PullRequestID:      arg.PullRequestID,
			SubjectNumber:      fromNullInt32(arg.SubjectNumber),
			SubjectState:       arg.SubjectState,
			SubjectMerged:      fromNullBool(arg.SubjectMerged),
			SubjectStateReason: arg.SubjectStateReason,
			AuthorLogin:        arg.AuthorLogin,
			AuthorID:           arg.AuthorID,
		})
	})
}

// --- User methods ---

// GetUser gets a user by ID
func (s *Store) GetUser(ctx context.Context) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.GetUser(ctx)
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// CreateUser creates a new user
func (s *Store) CreateUser(ctx context.Context) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.CreateUser(ctx)
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// UpdateUserGitHubIdentity updates the GitHub identity for a user
func (s *Store) UpdateUserGitHubIdentity(
	ctx context.Context,
	arg db.UpdateUserGitHubIdentityParams,
) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.UpdateUserGitHubIdentity(ctx, UpdateUserGitHubIdentityParams{
			GithubUserID:   arg.GithubUserID,
			GithubUsername: arg.GithubUsername,
		})
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// UpdateUserSyncSettings updates the sync settings for a user
func (s *Store) UpdateUserSyncSettings(
	ctx context.Context,
	syncSettings db.NullRawMessage,
) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.UpdateUserSyncSettings(ctx, fromNullRawMessage(syncSettings))
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// UpdateUserGitHubToken updates the GitHub token for a user
func (s *Store) UpdateUserGitHubToken(
	ctx context.Context,
	githubTokenEncrypted sql.NullString,
) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.UpdateUserGitHubToken(ctx, githubTokenEncrypted)
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// ClearUserGitHubToken clears the GitHub token for a user
func (s *Store) ClearUserGitHubToken(ctx context.Context) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.ClearUserGitHubToken(ctx)
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// UpdateUserRetentionSettings updates the retention settings for a user
func (s *Store) UpdateUserRetentionSettings(
	ctx context.Context,
	retentionSettings sql.NullString,
) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.UpdateUserRetentionSettings(ctx, retentionSettings)
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// UpdateUserUpdateSettings updates the update settings for a user
func (s *Store) UpdateUserUpdateSettings(
	ctx context.Context,
	updateSettings db.NullRawMessage,
) (db.User, error) {
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.UpdateUserUpdateSettings(ctx, fromNullRawMessage(updateSettings))
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// UpdateUserMutedUntil updates the muted until time for a user
func (s *Store) UpdateUserMutedUntil(
	ctx context.Context,
	mutedUntil sql.NullTime,
) (db.User, error) {
	mutedUntilStr := formatNullTime(mutedUntil)
	u, err := db.RetryOnBusy(ctx, func() (User, error) {
		return s.q.UpdateUserMutedUntil(ctx, mutedUntilStr)
	})
	if err != nil {
		return db.User{}, err
	}
	return toDBUser(u), nil
}

// --- Storage management methods ---

// GetStorageStats gets storage stats
func (s *Store) GetStorageStats(ctx context.Context, userID string) (db.StorageStats, error) {
	stats, err := db.RetryOnBusy(ctx, func() (GetStorageStatsRow, error) {
		return s.q.GetStorageStats(ctx, GetStorageStatsParams{
			UserID:   userID,
			UserID_2: userID,
		})
	})
	if err != nil {
		return db.StorageStats{}, err
	}
	return db.StorageStats{
		TotalCount:    stats.TotalCount,
		ArchivedCount: stats.ArchivedCount,
		StarredCount:  stats.StarredCount,
		SnoozedCount:  stats.SnoozedCount,
		UnreadCount:   stats.UnreadCount,
		TaggedCount:   stats.TaggedCount,
	}, nil
}

// CountEligibleForCleanup counts eligible notifications for cleanup
func (s *Store) CountEligibleForCleanup(
	ctx context.Context,
	userID string,
	params db.CleanupParams,
) (int64, error) {
	return db.RetryOnBusy(ctx, func() (int64, error) {
		return s.q.CountEligibleForCleanup(ctx, CountEligibleForCleanupParams{
			UserID:         userID,
			ProtectStarred: boolToInt64(params.ProtectStarred),
			ProtectTagged:  boolToInt64(params.ProtectTagged),
			CutoffDate:     params.CutoffDate,
		})
	})
}

// DeleteOldArchivedNotifications deletes old archived notifications
func (s *Store) DeleteOldArchivedNotifications(
	ctx context.Context,
	userID string,
	params db.CleanupParams,
) (int64, error) {
	return db.RetryOnBusy(ctx, func() (int64, error) {
		return s.q.DeleteOldArchivedNotifications(ctx, DeleteOldArchivedNotificationsParams{
			UserID:         userID,
			ProtectStarred: boolToInt64(params.ProtectStarred),
			ProtectTagged:  boolToInt64(params.ProtectTagged),
			CutoffDate:     params.CutoffDate,
			BatchSize:      params.BatchSize,
		})
	})
}

// DeleteOrphanedPullRequests deletes orphaned pull requests
func (s *Store) DeleteOrphanedPullRequests(ctx context.Context, userID string) (int64, error) {
	return db.RetryOnBusy(ctx, func() (int64, error) {
		return s.q.DeleteOrphanedPullRequests(ctx, DeleteOrphanedPullRequestsParams{
			UserID:   userID,
			UserID_2: userID,
		})
	})
}

// DeleteAllGitHubData deletes all GitHub data for a user
// (notifications, pull requests, repositories, sync state, tag assignments)
// Also clears sync settings so the user must set up sync again
// Preserves: tags, views, rules, users
func (s *Store) DeleteAllGitHubData(ctx context.Context, userID string) error {
	return db.RetryVoidOnBusy(ctx, func() error {
		// Use a transaction to ensure all deletes happen atomically
		tx, err := s.dbConn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			// Rollback will return an error if the transaction was already committed,
			// which is expected and safe to ignore
			if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
				// Only log if it's not the expected "transaction done" error
				_ = rollbackErr
			}
		}()

		// Delete in order to respect foreign key constraints
		// 1. Tag assignments (references notifications)
		_, err = tx.ExecContext(ctx, "DELETE FROM tag_assignments WHERE user_id = ?", userID)
		if err != nil {
			return err
		}

		// 2. Notifications
		_, err = tx.ExecContext(ctx, "DELETE FROM notifications WHERE user_id = ?", userID)
		if err != nil {
			return err
		}

		// 3. Pull requests
		_, err = tx.ExecContext(ctx, "DELETE FROM pull_requests WHERE user_id = ?", userID)
		if err != nil {
			return err
		}

		// 4. Repositories
		_, err = tx.ExecContext(ctx, "DELETE FROM repositories WHERE user_id = ?", userID)
		if err != nil {
			return err
		}

		// 5. Sync state
		_, err = tx.ExecContext(ctx, "DELETE FROM sync_state WHERE user_id = ?", userID)
		if err != nil {
			return err
		}

		// 6. Clear sync settings (reset to NULL so user has to set up sync again)
		// Note: This is a single-user app, so we update the user with id = 1
		_, err = tx.ExecContext(
			ctx,
			"UPDATE users SET sync_settings = NULL, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1",
		)
		if err != nil {
			return err
		}

		return tx.Commit()
	})
}

// Helper function
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
