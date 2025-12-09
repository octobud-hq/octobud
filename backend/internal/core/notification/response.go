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
	"github.com/octobud-hq/octobud/backend/internal/query/eval"
)

// BuildResponse builds a notification response with all enriched data
func (s *Service) BuildResponse(
	ctx context.Context,
	userID string,
	notification db.Notification,
	repoMap map[int64]db.Repository,
	evaluator *eval.Evaluator,
) (models.Notification, error) {
	item := models.NotificationFromDB(notification)

	var repo *db.Repository

	if notification.RepositoryID != 0 {
		var (
			repoValue db.Repository
			err       error
			ok        bool
		)

		if repoMap != nil {
			repoValue, ok = repoMap[notification.RepositoryID]
			if ok {
				repo = &repoValue
			}
		} else {
			repoValue, err = s.queries.GetRepositoryByID(ctx, userID, notification.RepositoryID)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return models.Notification{}, err
				}
				// If not found, repo stays nil
			} else {
				repo = &repoValue
			}
		}

		if repo != nil {
			repoResponse := models.RepositoryFromDB(*repo)
			item.Repository = &repoResponse
		}
	}

	// Fetch tags for this notification
	tags, err := s.queries.ListTagsForEntity(ctx, userID, db.ListTagsForEntityParams{
		EntityType: "notification",
		EntityID:   notification.ID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return models.Notification{}, errors.Join(ErrFailedToFetchTags, err)
	}

	if len(tags) > 0 {
		item.Tags = make([]models.Tag, 0, len(tags))
		for _, tag := range tags {
			tagResp := models.Tag{
				ID:   tag.ID, // UUID string
				Name: tag.Name,
			}
			if tag.Color.Valid {
				tagResp.Color = &tag.Color.String
			}
			if tag.Description.Valid {
				tagResp.Description = &tag.Description.String
			}
			item.Tags = append(item.Tags, tagResp)
		}
	}

	// Always compute action hints using unified query semantics
	// Pass nil for repo if not found - evaluator handles this gracefully
	hints := computeActionHintsForNotification(&notification, repo, evaluator)
	item.ActionHints = hints

	return item, nil
}

// IndexRepositories creates a map of repository ID to repository for efficient lookup
func (s *Service) IndexRepositories(
	ctx context.Context,
	userID string,
) (map[int64]db.Repository, error) {
	repositories, err := s.queries.ListRepositories(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]db.Repository, len(repositories))
	for _, repository := range repositories {
		result[repository.ID] = repository
	}

	return result, nil
}
