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

// Package timeline provides the timeline service.
package timeline

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"

	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// FilteredEventTypes contains event types that should be filtered out from timeline.
var FilteredEventTypes = map[string]bool{
	// Subscription
	"subscribed":   true,
	"unsubscribed": true,
	// PR
	"auto_squash_enabled":  true,
	"auto_squash_disabled": true,
	// Mentioned
	"mentioned": true,
	// Labels
	"labeled":   true,
	"unlabeled": true,
	// Projects
	"added_to_project_v2":            true,
	"removed_from_project_v2":        true,
	"project_v2_item_status_changed": true,
	// Copilot
	"copilot_work_started":  true,
	"copilot_work_finished": true,
	// Connections
	"connected":                       true,
	"disconnected":                    true,
	"review_requested":                true,
	"review_dismissed":                true,
	"automatic_base_change_succeeded": true,
	"automatic_base_change_failed":    true,
	"cross-referenced":                true,
	"referenced":                      true,
	"milestoned":                      true,
	"demilestoned":                    true,
	"base_ref_changed":                true,
	"base_ref_force_pushed":           true,
	// Issue modifications
	"renamed":            true,
	"parent_issue_added": true,
	"issue_type_added":   true,
	"sub_issue_added":    true,
	"issue_type_removed": true,
	"sub_issue_removed":  true,
	"issue_type_changed": true,
}

// TimelineService is the interface for the timeline service.
//

type TimelineService interface {
	FetchFilteredTimeline(
		ctx context.Context,
		client githubinterfaces.Client,
		subjectInfo *types.SubjectInfo,
		subjectType string,
		perPage, page int,
	) (*models.TimelineResult, error)
}

// Service provides timeline business logic operations.
type Service struct {
	logger *zap.Logger
}

// NewService creates a new timeline service with a logger.
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger: logger,
	}
}

// SupportsTimeline returns true if the notification type supports timeline.
func SupportsTimeline(subjectType string) bool {
	normalizedType := strings.ToLower(strings.ReplaceAll(subjectType, "_", ""))

	switch normalizedType {
	case "pullrequest", "pr":
		return true
	case "issue":
		return true
	case "discussion":
		return true
	default:
		return false
	}
}

// FetchFilteredTimeline fetches and filters timeline events from GitHub,
// intelligently fetching multiple pages if needed to satisfy pagination requirements.
// Note: Callers should check SupportsTimeline() before calling this function.
// subjectType is used to determine if GraphQL should be used (for discussions).
func (s *Service) FetchFilteredTimeline(
	ctx context.Context,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
	subjectType string,
	perPage, page int,
) (*models.TimelineResult, error) {
	s.logger.Info(
		"fetching filtered timeline",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.String("subject_type", subjectType),
		zap.Int("per_page", perPage),
		zap.Int("page", page),
	)

	allItems, err := fetchAndFilterTimelineEvents(
		ctx,
		s.logger,
		client,
		subjectInfo,
		subjectType,
		perPage,
		page,
	)
	if err != nil {
		s.logger.Error(
			"failed to fetch timeline events",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch timeline events: %w", err)
	}

	// Sort by timestamp (newest first)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Timestamp.After(allItems[j].Timestamp)
	})

	// Apply pagination
	total := len(allItems)
	start := (page - 1) * perPage
	end := start + perPage

	result := &models.TimelineResult{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		HasMore: end < total,
	}

	if start >= total {
		result.Items = []models.TimelineItem{}
		return result, nil
	}

	if end > total {
		end = total
	}

	result.Items = allItems[start:end]

	s.logger.Info(
		"timeline fetch completed",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("total_items", total),
		zap.Int("returned_items", len(result.Items)),
		zap.Bool("has_more", result.HasMore),
	)

	return result, nil
}

// fetchAndFilterTimelineEvents intelligently fetches timeline events from GitHub,
// filtering out unwanted event types and requesting more events if needed.
// For discussions, it uses GraphQL API. For issues/PRs, it uses REST API.
func fetchAndFilterTimelineEvents(
	ctx context.Context,
	logger *zap.Logger,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
	subjectType string,
	perPage, page int,
) ([]models.TimelineItem, error) {
	normalizedType := strings.ToLower(strings.ReplaceAll(subjectType, "_", ""))
	isDiscussion := normalizedType == "discussion"

	if isDiscussion {
		return fetchDiscussionTimelineEvents(ctx, logger, client, subjectInfo, perPage, page)
	}

	return fetchRESTTimelineEvents(ctx, logger, client, subjectInfo, perPage, page)
}

// fetchRESTTimelineEvents fetches timeline events using REST API (for issues/PRs).
func fetchRESTTimelineEvents(
	ctx context.Context,
	logger *zap.Logger,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
	perPage, page int,
) ([]models.TimelineItem, error) {
	const (
		maxGitHubPerPage = 100
		maxGitHubPages   = 10 // Safety limit to avoid infinite loops
	)

	// Calculate how many events we need in total (for all pages up to and including the requested page)
	totalNeeded := perPage * page

	var allFilteredItems []models.TimelineItem
	githubPage := 1

	// Keep fetching from GitHub until we have enough filtered events or reach the safety limit
	for githubPage <= maxGitHubPages {
		// Fetch a page from GitHub
		logger.Info(
			"fetching timeline page from GitHub",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("github_page", githubPage),
			zap.Int("per_page", maxGitHubPerPage),
		)

		timeline, err := client.FetchTimeline(
			ctx,
			subjectInfo.Owner,
			subjectInfo.Repo,
			subjectInfo.Number,
			maxGitHubPerPage,
			githubPage,
		)
		if err != nil {
			logger.Warn(
				"failed to fetch timeline page from GitHub",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("github_page", githubPage),
				zap.Int("filtered_items_so_far", len(allFilteredItems)),
				zap.Error(err),
			)
			// If this is the first fetch and we have no data, return the error
			// Otherwise, return what we have so far (graceful degradation)
			if githubPage == 1 && len(allFilteredItems) == 0 {
				return nil, fmt.Errorf("failed to fetch timeline events: %w", err)
			}
			// Don't fail completely - maybe the repo is private or deleted
			// Return what we have so far
			break
		}

		logger.Info(
			"received timeline page from GitHub",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("github_page", githubPage),
			zap.Int("events_returned", len(timeline)),
		)

		// If GitHub returned no events, we've reached the end
		if len(timeline) == 0 {
			logger.Info(
				"no more timeline events from GitHub",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("github_page", githubPage),
			)
			break
		}

		// Convert and filter timeline events
		filteredCount := 0
		skippedCount := 0
		for _, event := range timeline {
			// Skip filtered event types
			if FilteredEventTypes[event.Event] {
				skippedCount++
				continue
			}

			item := convertTimelineEvent(event)
			if item == nil {
				// Skip events without timestamp
				skippedCount++
				continue
			}
			allFilteredItems = append(allFilteredItems, *item)
			filteredCount++
		}

		logger.Info(
			"processed timeline page",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("github_page", githubPage),
			zap.Int("events_included", filteredCount),
			zap.Int("events_skipped", skippedCount),
			zap.Int("total_filtered_items", len(allFilteredItems)),
			zap.Int("total_needed", totalNeeded),
		)

		// If we have enough filtered events, we can stop fetching
		if len(allFilteredItems) >= totalNeeded {
			logger.Info(
				"collected enough filtered timeline events",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("total_filtered_items", len(allFilteredItems)),
				zap.Int("total_needed", totalNeeded),
			)
			break
		}

		// If GitHub returned fewer events than requested, we've reached the end
		if len(timeline) < maxGitHubPerPage {
			logger.Info(
				"reached end of timeline (fewer events than requested)",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("events_returned", len(timeline)),
				zap.Int("max_per_page", maxGitHubPerPage),
			)
			break
		}

		githubPage++
	}

	logger.Info(
		"finished fetching timeline events",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("total_filtered_items", len(allFilteredItems)),
		zap.Int("github_pages_fetched", githubPage-1),
	)

	return allFilteredItems, nil
}

// fetchDiscussionTimelineEvents fetches timeline events using GraphQL API (for discussions).
func fetchDiscussionTimelineEvents(
	ctx context.Context,
	logger *zap.Logger,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
	perPage, page int,
) ([]models.TimelineItem, error) {
	const (
		maxGraphQLPerPage = 100
		maxGraphQLPages   = 10 // Safety limit to avoid infinite loops
	)

	// Calculate how many events we need in total (for all pages up to and including the requested page)
	totalNeeded := perPage * page

	var allFilteredItems []models.TimelineItem
	var after string // GraphQL cursor for pagination
	graphqlPage := 1

	// Keep fetching from GitHub GraphQL until we have enough events or reach the safety limit
	for graphqlPage <= maxGraphQLPages {
		// Calculate how many to fetch this time
		// We need to fetch enough to satisfy totalNeeded, but cap at maxGraphQLPerPage
		remainingNeeded := totalNeeded - len(allFilteredItems)
		first := remainingNeeded
		if first > maxGraphQLPerPage {
			first = maxGraphQLPerPage
		}
		if first <= 0 {
			first = maxGraphQLPerPage
		}

		logger.Info(
			"fetching discussion timeline page from GitHub GraphQL",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("graphql_page", graphqlPage),
			zap.Int("first", first),
			zap.String("after", after),
		)

		timeline, hasNextPage, endCursor, err := client.FetchDiscussionComments(
			ctx,
			subjectInfo.Owner,
			subjectInfo.Repo,
			subjectInfo.Number,
			first,
			after,
		)
		if err != nil {
			logger.Warn(
				"failed to fetch discussion timeline page from GitHub GraphQL",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("graphql_page", graphqlPage),
				zap.Int("filtered_items_so_far", len(allFilteredItems)),
				zap.Error(err),
			)
			// If this is the first fetch and we have no data, return the error
			// Otherwise, return what we have so far (graceful degradation)
			if graphqlPage == 1 && len(allFilteredItems) == 0 {
				return nil, fmt.Errorf("failed to fetch discussion timeline events: %w", err)
			}
			// Don't fail completely - maybe the repo is private or deleted
			// Return what we have so far
			break
		}

		logger.Info(
			"received discussion timeline page from GitHub GraphQL",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("graphql_page", graphqlPage),
			zap.Int("events_returned", len(timeline)),
			zap.Bool("has_next_page", hasNextPage),
		)

		// If GitHub returned no events, we've reached the end
		if len(timeline) == 0 {
			logger.Info(
				"no more discussion timeline events from GitHub GraphQL",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("graphql_page", graphqlPage),
			)
			break
		}

		// Convert timeline events to TimelineItems
		// For discussions, we don't filter out any event types (all comments are relevant)
		for _, event := range timeline {
			item := convertTimelineEvent(event)
			if item == nil {
				// Skip events without timestamp
				continue
			}
			allFilteredItems = append(allFilteredItems, *item)
		}

		logger.Info(
			"processed discussion timeline page",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("graphql_page", graphqlPage),
			zap.Int("events_included", len(timeline)),
			zap.Int("total_filtered_items", len(allFilteredItems)),
			zap.Int("total_needed", totalNeeded),
		)

		// If we have enough filtered events, we can stop fetching
		if len(allFilteredItems) >= totalNeeded {
			logger.Info(
				"collected enough filtered discussion timeline events",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("total_filtered_items", len(allFilteredItems)),
				zap.Int("total_needed", totalNeeded),
			)
			break
		}

		// If there's no next page, we've reached the end
		if !hasNextPage {
			logger.Info(
				"reached end of discussion timeline (no next page)",
				zap.String("owner", subjectInfo.Owner),
				zap.String("repo", subjectInfo.Repo),
				zap.Int("number", subjectInfo.Number),
				zap.Int("events_returned", len(timeline)),
			)
			break
		}

		// Update cursor for next page
		after = endCursor
		graphqlPage++
	}

	logger.Info(
		"finished fetching discussion timeline events",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("total_filtered_items", len(allFilteredItems)),
		zap.Int("graphql_pages_fetched", graphqlPage-1),
	)

	return allFilteredItems, nil
}

// convertTimelineEvent converts a GitHub timeline event to a normalized TimelineItem.
func convertTimelineEvent(event types.TimelineEvent) *models.TimelineItem {
	item := &models.TimelineItem{
		Event:   event.Event,
		ID:      event.ID,
		Body:    event.Body,
		State:   strings.ToUpper(event.State), // Normalize to uppercase
		Message: event.Message,
		SHA:     event.SHA,
		HTMLURL: event.HTMLURL,
	}

	// Set author from either User or Actor field
	if event.User != nil {
		item.AuthorLogin = event.User.Login
		item.AuthorAvatarURL = event.User.AvatarURL
	} else if event.Actor != nil {
		item.AuthorLogin = event.Actor.Login
		item.AuthorAvatarURL = event.Actor.AvatarURL
	}

	// Set timestamp for sorting - if no timestamp exists, return nil

	if event.SubmittedAt != nil {
		item.Timestamp = *event.SubmittedAt
		item.SubmittedAt = event.SubmittedAt
	} else if event.CreatedAt != nil {
		item.Timestamp = *event.CreatedAt
		item.CreatedAt = event.CreatedAt
	} else if event.UpdatedAt != nil {
		item.Timestamp = *event.UpdatedAt
		item.UpdatedAt = event.UpdatedAt
	} else {
		// No timestamp available - skip this event
		return nil
	}

	if event.UpdatedAt != nil {
		item.UpdatedAt = event.UpdatedAt
	}

	return item
}
