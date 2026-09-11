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

// Package notification provides the business logic for notifications.
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
	"github.com/octobud-hq/octobud/backend/internal/query/eval"
)

// Error definitions
var (
	ErrInvalidQuery                      = errors.New("invalid query")
	ErrFailedToBuildQuery                = errors.New("failed to build query")
	ErrFailedToListNotifications         = errors.New("failed to list notifications")
	ErrFailedToIndexRepositories         = errors.New("failed to index repositories")
	ErrFailedToCountRepositories         = errors.New("failed to count notifications by repository")
	ErrFailedToGetRepository             = errors.New("failed to get repository")
	ErrFailedToBuildNotificationResponse = errors.New("failed to build notification response")
	ErrFailedToUpsertNotification        = errors.New("failed to upsert notification")
	ErrFailedToUpdateNotificationSubject = errors.New("failed to update notification subject")
	ErrFailedToGetNotification           = errors.New("failed to get notification")
)

// GetByGithubID fetches a notification by its GitHub identifier.
func (s *Service) GetByGithubID(
	ctx context.Context,
	userID, githubID string,
) (db.Notification, error) {
	notification, err := s.queries.GetNotificationByGithubID(ctx, userID, githubID)
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToGetNotification, err)
	}
	return notification, nil
}

// ListNotifications retrieves notifications matching the provided filtering options with enriched data.
func (s *Service) ListNotifications(
	ctx context.Context,
	userID string,
	opts models.ListOptions,
) (models.ListDetailsResult, error) {
	limit, offset, page, pageSize := normalizedPagination(opts)

	// Use unified BuildQuery which applies business rules based on query content
	dbQuery, err := query.BuildQueryWithOptions(opts.Query, limit, offset, opts.IncludeSubject)

	if err != nil {
		// Wrap query errors in a high-level error type
		return models.ListDetailsResult{}, errors.Join(ErrInvalidQuery, err)
	}
	dbQuery = query.ApplyRepositoryFilter(dbQuery, opts.RepositoryIDs)

	// Execute query
	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return models.ListDetailsResult{}, errors.Join(ErrFailedToListNotifications, err)
	}

	// Index repositories for efficient lookup
	repoMap, err := s.IndexRepositories(ctx, userID)
	if err != nil {
		return models.ListDetailsResult{}, errors.Join(ErrFailedToIndexRepositories, err)
	}

	// Create evaluator once for all notifications (optimization)
	// Always create evaluator, even for empty queries, so action hints work correctly
	evaluator, err := query.NewEvaluator(opts.Query)
	if err != nil {
		// Continue with nil evaluator - hints will be conservative (empty)
		evaluator = nil
	}

	// Build responses for each notification
	responses := make([]models.Notification, 0, len(result.Notifications))
	for _, notification := range result.Notifications {
		item, err := s.BuildResponse(ctx, userID, notification, repoMap, evaluator)
		if err != nil {
			return models.ListDetailsResult{}, errors.Join(
				ErrFailedToBuildNotificationResponse,
				err,
			)
		}

		// Conditionally exclude subjectRaw to reduce payload size for list views
		if !opts.IncludeSubject {
			item.SubjectRaw = nil
		}

		responses = append(responses, item)
	}

	return models.ListDetailsResult{
		Notifications: responses,
		Total:         result.Total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// ListPollNotifications retrieves only essential fields for polling (much smaller payload)
func (s *Service) ListPollNotifications(
	ctx context.Context,
	userID string,
	opts models.ListOptions,
) (models.ListPollResult, error) {
	limit, offset, page, pageSize := normalizedPagination(opts)

	// Use unified BuildQuery which applies business rules based on query content
	dbQuery, err := query.BuildQueryWithOptions(
		opts.Query,
		limit,
		offset,
		false,
	) // Never include subject for polling
	if err != nil {
		return models.ListPollResult{}, err
	}
	dbQuery = query.ApplyRepositoryFilter(dbQuery, opts.RepositoryIDs)

	// Execute query
	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return models.ListPollResult{}, err
	}

	// Index repositories for efficient lookup (only need fullName)
	repoMap, err := s.IndexRepositories(ctx, userID)
	if err != nil {
		return models.ListPollResult{}, errors.Join(ErrFailedToIndexRepositories, err)
	}

	// Build poll responses - only include essential fields
	responses := make([]models.PollNotification, 0, len(result.Notifications))
	for _, notification := range result.Notifications {
		item := models.PollNotification{
			ID:                notification.ID,
			GithubID:          notification.GithubID,
			EffectiveSortDate: notification.EffectiveSortDate.Format(time.RFC3339Nano),
			Archived:          notification.Archived,
			Muted:             notification.Muted,
			SubjectTitle:      notification.SubjectTitle,
			SubjectType:       notification.SubjectType,
			Reason:            models.NullStringPtr(notification.Reason),
		}

		// Only include repo fullName if available
		if notification.RepositoryID != 0 {
			if repo, ok := repoMap[notification.RepositoryID]; ok {
				item.RepoFullName = repo.FullName
			}
		}

		responses = append(responses, item)
	}

	return models.ListPollResult{
		Notifications: responses,
		Total:         result.Total,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// ListNotificationsFromQueryString lists notifications from a query string (for bulk operations).
// repositoryIDs optionally narrows the query to those repositories.
func (s *Service) ListNotificationsFromQueryString(
	ctx context.Context,
	userID, queryStr string,
	repositoryIDs []int64,
	limit int32,
) ([]db.Notification, error) {
	dbQuery, err := query.BuildQuery(queryStr, limit, 0)
	if err != nil {
		return nil, errors.Join(ErrFailedToBuildQuery, err)
	}
	dbQuery = query.ApplyRepositoryFilter(dbQuery, repositoryIDs)

	result, err := s.queries.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return nil, errors.Join(ErrFailedToListNotifications, err)
	}

	return result.Notifications, nil
}

// ListRepositoryCounts returns, for every repository with at least one notification matching
// queryStr, the repository's identity plus total and unread counts, ordered by unread, then
// total, then name (the order the repository selector displays them in). Repositories in
// includeIDs that have no matches are appended with zero counts so the caller always has
// an identity for selected and pinned repositories.
func (s *Service) ListRepositoryCounts(
	ctx context.Context,
	userID, queryStr string,
	includeIDs []int64,
) ([]models.RepositoryCount, error) {
	dbQuery, err := query.BuildQueryWithOptions(queryStr, 0, 0, false)
	if err != nil {
		return nil, errors.Join(ErrInvalidQuery, err)
	}

	counts, err := s.queries.CountNotificationsByRepository(ctx, userID, dbQuery)
	if err != nil {
		return nil, errors.Join(ErrFailedToCountRepositories, err)
	}

	result := make([]models.RepositoryCount, 0, len(counts)+len(includeIDs))
	seen := make(map[int64]struct{}, len(counts))
	for _, count := range counts {
		seen[count.RepositoryID] = struct{}{}
		result = append(result, models.RepositoryCount{
			Repository: models.Repository{
				ID:             count.RepositoryID,
				Name:           count.Name,
				FullName:       count.FullName,
				OwnerLogin:     models.NullStringPtr(count.OwnerLogin),
				OwnerAvatarURL: models.NullStringPtr(count.OwnerAvatarURL),
				HTMLURL:        models.NullStringPtr(count.HTMLURL),
			},
			Total:  count.Total,
			Unread: count.Unread,
		})
	}

	extra := make([]models.RepositoryCount, 0, len(includeIDs))
	for _, id := range includeIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		repo, err := s.queries.GetRepositoryByID(ctx, userID, id)
		if errors.Is(err, sql.ErrNoRows) {
			continue // unknown or deleted repository: nothing to identify
		}
		if err != nil {
			return nil, errors.Join(ErrFailedToGetRepository, err)
		}
		repoResponse := models.RepositoryFromDB(repo)
		repoResponse.Raw = nil
		extra = append(extra, models.RepositoryCount{Repository: repoResponse})
	}
	sort.SliceStable(extra, func(i, j int) bool {
		return strings.ToLower(extra[i].Repository.FullName) <
			strings.ToLower(extra[j].Repository.FullName)
	})

	return append(result, extra...), nil
}

// GetTagsForNotification returns tags for a notification
func (s *Service) GetTagsForNotification(
	ctx context.Context,
	userID string,
	notificationID int64,
) ([]db.Tag, error) {
	tags, err := s.queries.ListTagsForEntity(ctx, userID, db.ListTagsForEntityParams{
		EntityType: "notification",
		EntityID:   notificationID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.Join(ErrFailedToFetchTags, err)
	}
	return tags, nil
}

// NewEvaluator creates a query evaluator for the given query string
func (s *Service) NewEvaluator(queryStr string) (*eval.Evaluator, error) {
	return query.NewEvaluator(queryStr)
}

// UpsertNotification creates or updates a notification
func (s *Service) UpsertNotification(
	ctx context.Context,
	userID string,
	params db.UpsertNotificationParams,
) (db.Notification, error) {
	notification, err := s.queries.UpsertNotification(ctx, userID, params)
	if err != nil {
		return db.Notification{}, errors.Join(ErrFailedToUpsertNotification, err)
	}
	return notification, nil
}

// UpdateNotificationSubject updates the subject data for a notification
func (s *Service) UpdateNotificationSubject(
	ctx context.Context,
	userID string,
	params db.UpdateNotificationSubjectParams,
) error {
	if err := s.queries.UpdateNotificationSubject(ctx, userID, params); err != nil {
		return errors.Join(ErrFailedToUpdateNotificationSubject, err)
	}
	return nil
}

// GetNotificationWithDetails retrieves a single notification with all enriched data
func (s *Service) GetNotificationWithDetails(
	ctx context.Context, userID, githubID, queryStr string,
) (models.Notification, error) {
	notification, err := s.GetByGithubID(ctx, userID, githubID)
	if err != nil {
		return models.Notification{}, err
	}

	// Always create evaluator, even for empty queries, so action hints work correctly
	evaluator, err := query.NewEvaluator(queryStr)
	if err != nil {
		// Continue with nil evaluator - hints will be conservative (empty)
		evaluator = nil
	}

	// Build response (no repoMap needed for single notification)
	return s.BuildResponse(ctx, userID, notification, nil, evaluator)
}
