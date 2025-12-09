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

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions for tag operations
var (
	ErrNotificationNotFound           = errors.New("notification not found")
	ErrTagNotFound                    = errors.New("tag not found")
	ErrFailedToGetTag                 = errors.New("failed to get tag")
	ErrFailedToAssignTag              = errors.New("failed to assign tag")
	ErrFailedToGetUpdatedNotification = errors.New("failed to get updated notification")
	ErrInvalidTagName                 = errors.New("invalid tag name - cannot generate slug")
	ErrFailedToCreateTag              = errors.New("failed to create tag")
	ErrFailedToRemoveTag              = errors.New("failed to remove tag")
)

// AssignTag assigns a tag to a notification by tag ID
func (s *Service) AssignTag(
	ctx context.Context,
	userID, githubID, tagID string,
) (db.Notification, error) {
	// Get the notification
	notification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Notification{}, errors.Join(ErrNotificationNotFound, err)
		}
		return db.Notification{}, errors.Join(ErrFailedToGetNotification, err)
	}

	// Verify the tag exists
	tag, err := s.queries.GetTag(ctx, userID, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Notification{}, errors.Join(ErrTagNotFound, err)
		}
		return db.Notification{}, errors.Join(ErrFailedToGetTag, err)
	}

	// Assign the tag
	_, err = s.queries.AssignTagToEntity(ctx, userID, db.AssignTagToEntityParams{
		TagID:      tag.ID,
		EntityType: "notification",
		EntityID:   notification.ID,
	})
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToAssignTag, err)
	}

	// Return the updated notification
	updatedNotification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToGetUpdatedNotification, err)
	}

	return updatedNotification, nil
}

// AssignTagByName assigns a tag to a notification by name, creating it if it doesn't exist
func (s *Service) AssignTagByName(
	ctx context.Context,
	userID, githubID, tagName string,
) (db.Notification, error) {
	// Get the notification
	notification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Notification{}, errors.Join(ErrNotificationNotFound, err)
		}
		return db.Notification{}, errors.Join(ErrFailedToGetNotification, err)
	}

	// Try to get existing tag by name
	tag, err := s.queries.GetTagByName(ctx, userID, tagName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tag doesn't exist, create it with auto-generated slug and default color
			slug := models.Slugify(tagName)
			if slug == "" {
				return db.Notification{}, ErrInvalidTagName
			}

			// Use indigo as default color
			defaultColor := sql.NullString{String: "#6366f1", Valid: true}

			tag, err = s.queries.UpsertTag(ctx, userID, db.UpsertTagParams{
				Name:        tagName,
				Slug:        slug,
				Color:       defaultColor,
				Description: sql.NullString{}, // Empty description
			})
			if err != nil {
				return db.Notification{}, errors.Join(ErrFailedToCreateTag, err)
			}
		} else {
			return db.Notification{}, errors.Join(ErrFailedToGetTag, err)
		}
	}

	// Assign the tag
	_, err = s.queries.AssignTagToEntity(ctx, userID, db.AssignTagToEntityParams{
		TagID:      tag.ID,
		EntityType: "notification",
		EntityID:   notification.ID,
	})
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToAssignTag, err)
	}

	// Return the updated notification
	updatedNotification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToGetUpdatedNotification, err)
	}

	return updatedNotification, nil
}

// RemoveTag removes a tag from a notification
func (s *Service) RemoveTag(
	ctx context.Context,
	userID, githubID, tagID string,
) (db.Notification, error) {
	// Get the notification
	notification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Notification{}, errors.Join(ErrNotificationNotFound, err)
		}
		return db.Notification{}, errors.Join(ErrFailedToGetNotification, err)
	}

	// Remove the tag
	err = s.queries.RemoveTagAssignment(ctx, userID, db.RemoveTagAssignmentParams{
		TagID:      tagID,
		EntityType: "notification",
		EntityID:   notification.ID,
	})
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToRemoveTag, err)
	}

	// Return the updated notification
	updatedNotification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToGetUpdatedNotification, err)
	}

	return updatedNotification, nil
}
