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

package tag

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
)

// Error definitions
var (
	ErrFailedToListTags              = errors.New("failed to list tags")
	ErrInvalidTagName                = errors.New("invalid tag name - cannot generate slug")
	ErrFailedToCreateTag             = errors.New("failed to create tag")
	ErrFailedToUpdateTag             = errors.New("failed to update tag")
	ErrTagNotFound                   = errors.New("tag not found")
	ErrFailedToDeleteTag             = errors.New("failed to delete tag")
	ErrTagIDsRequired                = errors.New("tagIDs is required")
	ErrFailedToUpdateTagDisplayOrder = errors.New("failed to update tag display order")
	ErrFailedToGetTag                = errors.New("failed to get tag")
	ErrFailedToGetTagByName          = errors.New("failed to get tag by name")
	ErrFailedToListTagsForEntity     = errors.New("failed to list tags for entity")
	ErrFailedToBuildQuery            = errors.New("failed to build query")
	ErrFailedToListNotifications     = errors.New("failed to list notifications")
)

// ListTagsWithUnreadCounts returns all tags with their unread counts
func (s *Service) ListTagsWithUnreadCounts(
	ctx context.Context,
	userID string,
) ([]models.Tag, error) {
	tags, err := s.queries.ListAllTags(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToListTags, err)
	}

	response := make([]models.Tag, 0, len(tags))
	for _, tag := range tags {
		tagResp := models.TagFromDB(tag)

		// Calculate unread count for this tag
		unreadCount, err := s.calculateTagUnreadCount(ctx, userID, tag.Slug)
		// If there's an error or unread count is 0, don't include it.
		if err == nil && unreadCount > 0 {
			// Only include unreadCount if > 0
			tagResp.UnreadCount = &unreadCount
		}

		response = append(response, tagResp)
	}

	return response, nil
}

// CreateTag creates a new tag
func (s *Service) CreateTag(
	ctx context.Context,
	userID, name string,
	color, description *string,
) (db.Tag, error) {
	slug := models.Slugify(name)
	if slug == "" {
		return db.Tag{}, ErrInvalidTagName
	}

	var colorNull, descriptionNull sql.NullString
	if color != nil && *color != "" {
		colorNull = sql.NullString{String: *color, Valid: true}
	}
	if description != nil && *description != "" {
		descriptionNull = sql.NullString{String: *description, Valid: true}
	}

	tag, err := s.queries.UpsertTag(ctx, userID, db.UpsertTagParams{
		Name:        name,
		Slug:        slug,
		Color:       colorNull,
		Description: descriptionNull,
	})
	if err != nil {
		return db.Tag{}, errors.Join(ErrFailedToCreateTag, err)
	}

	return tag, nil
}

// UpdateTag updates an existing tag
func (s *Service) UpdateTag(
	ctx context.Context,
	userID, tagID, name string,
	color, description *string,
) (db.Tag, error) {
	slug := models.Slugify(name)
	if slug == "" {
		return db.Tag{}, ErrInvalidTagName
	}

	var colorNull, descriptionNull sql.NullString
	if color != nil && *color != "" {
		colorNull = sql.NullString{String: *color, Valid: true}
	}
	if description != nil && *description != "" {
		descriptionNull = sql.NullString{String: *description, Valid: true}
	}

	tag, err := s.queries.UpdateTag(ctx, userID, db.UpdateTagParams{
		ID:          tagID,
		Name:        name,
		Slug:        slug,
		Color:       colorNull,
		Description: descriptionNull,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Tag{}, errors.Join(ErrTagNotFound, err)
		}
		return db.Tag{}, errors.Join(ErrFailedToUpdateTag, err)
	}

	return tag, nil
}

// DeleteTag deletes a tag
func (s *Service) DeleteTag(ctx context.Context, userID, tagID string) error {
	err := s.queries.DeleteTag(ctx, userID, tagID)
	if err != nil {
		return errors.Join(ErrFailedToDeleteTag, err)
	}
	return nil
}

// ReorderTags updates the display order of tags
func (s *Service) ReorderTags(
	ctx context.Context,
	userID string,
	tagIDs []string,
) ([]db.Tag, error) {
	if len(tagIDs) == 0 {
		return nil, ErrTagIDsRequired
	}

	// Update display order for each tag
	for i, tagID := range tagIDs {
		err := s.queries.UpdateTagDisplayOrder(ctx, userID, db.UpdateTagDisplayOrderParams{
			ID:           tagID,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return nil, errors.Join(ErrFailedToUpdateTagDisplayOrder, err)
		}
	}

	// Return the updated tags list
	tags, err := s.queries.ListAllTags(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToListTags, err)
	}

	return tags, nil
}

// GetTag returns a tag by ID
func (s *Service) GetTag(ctx context.Context, userID, tagID string) (db.Tag, error) {
	tag, err := s.queries.GetTag(ctx, userID, tagID)
	if err != nil {
		return db.Tag{}, errors.Join(ErrFailedToGetTag, err)
	}
	return tag, nil
}

// GetTagByName returns a tag by name
func (s *Service) GetTagByName(ctx context.Context, userID, name string) (db.Tag, error) {
	tag, err := s.queries.GetTagByName(ctx, userID, name)
	if err != nil {
		return db.Tag{}, errors.Join(ErrFailedToGetTagByName, err)
	}
	return tag, nil
}

// ListTags returns all tags (without unread counts)
func (s *Service) ListTags(ctx context.Context, userID string) ([]db.Tag, error) {
	tags, err := s.queries.ListAllTags(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToListTags, err)
	}
	return tags, nil
}

// calculateTagUnreadCount calculates the count of unread notifications for a tag
func (s *Service) calculateTagUnreadCount(
	ctx context.Context,
	userID, tagSlug string,
) (int64, error) {
	// Build explicit query: tags:{slug} in:anywhere -is:muted is:unread
	// This shows all unread notifications with the tag, including archived/snoozed/filtered, but excluding muted
	queryStr := fmt.Sprintf("tags:%s in:anywhere -is:muted is:unread", tagSlug)

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
