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
	"automatic_base_change_succeeded": true,
	"automatic_base_change_failed":    true,
	"cross-referenced":                true,
	"referenced":                      true,
	"base_ref_changed":                true,
	"base_ref_force_pushed":           true,
	// Issue modifications
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
	s.logger.Debug(
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

	s.logger.Debug(
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
		logger.Debug(
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

		logger.Debug(
			"received timeline page from GitHub",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("github_page", githubPage),
			zap.Int("events_returned", len(timeline)),
		)

		// If GitHub returned no events, we've reached the end
		if len(timeline) == 0 {
			logger.Debug(
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

		logger.Debug(
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
			logger.Debug(
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
			logger.Debug(
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

	logger.Debug(
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
// We use `last` to fetch the most recent comments first, then paginate from that sorted list.
// This ensures users see the most recent activity first.
func fetchDiscussionTimelineEvents(
	ctx context.Context,
	logger *zap.Logger,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
	perPage, page int,
) ([]models.TimelineItem, error) {
	const maxCommentsToFetch = 200 // Safety limit to avoid fetching too many comments

	// Calculate how many events we need in total (for all pages up to and including the requested page)
	totalNeeded := perPage * page

	// Cap the fetch count to avoid fetching too many comments
	fetchCount := totalNeeded
	if fetchCount > maxCommentsToFetch {
		fetchCount = maxCommentsToFetch
	}

	logger.Debug(
		"fetching discussion timeline from GitHub GraphQL",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("fetch_count", fetchCount),
		zap.Int("requested_page", page),
		zap.Int("per_page", perPage),
	)

	// Use 'last' to fetch the most recent comments
	// The FetchDiscussionComments function will reverse them so newest come first
	timeline, hasNextPage, _, err := client.FetchDiscussionComments(
		ctx,
		subjectInfo.Owner,
		subjectInfo.Repo,
		subjectInfo.Number,
		fetchCount,
		"", // No cursor for the initial fetch
	)
	if err != nil {
		logger.Warn(
			"failed to fetch discussion timeline from GitHub GraphQL",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch discussion timeline events: %w", err)
	}

	logger.Debug(
		"received discussion timeline from GitHub GraphQL",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("events_returned", len(timeline)),
		zap.Bool("has_next_page", hasNextPage),
	)

	// Convert timeline events to TimelineItems
	// For discussions, we don't filter out any event types (all comments are relevant)
	var allFilteredItems []models.TimelineItem
	for _, event := range timeline {
		item := convertTimelineEvent(event)
		if item == nil {
			// Skip events without timestamp
			continue
		}
		allFilteredItems = append(allFilteredItems, *item)
	}

	logger.Debug(
		"finished fetching discussion timeline events",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("total_filtered_items", len(allFilteredItems)),
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

	// Set event-specific metadata
	if event.RequestedReviewer != nil {
		item.RequestedReviewerLogin = event.RequestedReviewer.Login
	}
	if event.Label != nil {
		item.LabelName = event.Label.Name
		item.LabelColor = event.Label.Color
	}
	if event.Milestone != nil {
		item.MilestoneTitle = event.Milestone.Title
	}
	if event.Rename != nil {
		item.RenameFrom = event.Rename.From
		item.RenameTo = event.Rename.To
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
