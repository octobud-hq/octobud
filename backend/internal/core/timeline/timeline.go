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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"

	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

const (
	// restMaxPerPage is the maximum number of events per REST API page.
	restMaxPerPage = 30
	// restMaxPages is the safety limit on total pages fetched.
	restMaxPages = 10
)

// FilteredEventTypes contains event types that should be filtered out from timeline.
var FilteredEventTypes = map[string]bool{
	// Subscription noise
	"subscribed":   true,
	"unsubscribed": true,
	// Auto-squash
	"auto_squash_enabled":  true,
	"auto_squash_disabled": true,
	// Project board events
	"added_to_project_v2":            true,
	"removed_from_project_v2":        true,
	"project_v2_item_status_changed": true,
	// Copilot events
	"copilot_work_started":  true,
	"copilot_work_finished": true,
	// Low-signal auto-change events
	"automatic_base_change_succeeded": true,
	"automatic_base_change_failed":    true,
	// Referenced (noisy commit references, distinct from cross-referenced)
	"referenced": true,
	// Mentioned (redundant — the comment containing the @mention is already shown)
	"mentioned": true,
	// Issue sub-structure events
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

// FetchFilteredTimeline fetches timeline events from GitHub and applies pagination.
// For REST (issues/PRs), fetches pages forward from page 1, stopping as soon as
// enough filtered items are collected. For discussions, fetches all via GraphQL.
// Note: Callers should check SupportsTimeline() before calling this function.
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

	normalizedType := strings.ToLower(strings.ReplaceAll(subjectType, "_", ""))

	var allItems []models.TimelineItem
	var hasMorePages bool
	var err error

	if normalizedType == "discussion" {
		// Discussions: fetch all via GraphQL, sort, paginate client-side
		allItems, err = fetchDiscussionTimelineEvents(ctx, s.logger, client, subjectInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch timeline events: %w", err)
		}
		// Discussion events need sorting (GraphQL doesn't guarantee order)
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[i].Timestamp.After(allItems[j].Timestamp)
		})
	} else {
		// Issues/PRs: fetch REST pages with early stop
		needed := page * perPage
		allItems, hasMorePages, err = fetchRESTTimelineEvents(
			ctx, s.logger, client, subjectInfo, needed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch timeline events: %w", err)
		}
	}

	// Apply pagination
	total := len(allItems)
	start := (page - 1) * perPage
	end := start + perPage

	result := &models.TimelineResult{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		HasMore: end < total || hasMorePages,
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

// fetchRESTTimelineEvents fetches timeline events using the REST API.
// The API returns events oldest-first (ascending), so we discover the last page
// via a probe request, then fetch backwards from lastPage to collect newest events first.
// Stops as soon as `needed` filtered items are collected.
// Returns (items, hasMorePages, error) where hasMorePages indicates unfetched REST pages remain.
func fetchRESTTimelineEvents(
	ctx context.Context,
	logger *zap.Logger,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
	needed int,
) ([]models.TimelineItem, bool, error) {
	// Fetch page 1 to discover lastPage from Link header.
	// For single-page timelines, this is the only request needed.
	firstPageEvents, lastPage, err := client.FetchTimeline(
		ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number,
		restMaxPerPage, 1,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch timeline page 1: %w", err)
	}

	logger.Debug(
		"fetched timeline page 1",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("events", len(firstPageEvents)),
		zap.Int("last_page", lastPage),
	)

	// Single-page: reverse and return
	if lastPage <= 1 {
		var items []models.TimelineItem
		for i := len(firstPageEvents) - 1; i >= 0; i-- {
			if FilteredEventTypes[firstPageEvents[i].Event] {
				continue
			}
			item := convertTimelineEvent(firstPageEvents[i])
			if item != nil {
				items = append(items, *item)
			}
		}
		return items, false, nil
	}

	// Multi-page: fetch from lastPage backwards to collect newest events first
	var allItems []models.TimelineItem
	hasMorePages := false
	startPage := lastPage

	// Cap to safety limit
	if startPage > restMaxPages {
		startPage = restMaxPages
		hasMorePages = true // We're capping, so there are definitely more pages
	}

	for p := startPage; p >= 1; p-- {
		events, _, err := client.FetchTimeline(
			ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number,
			restMaxPerPage, p,
		)
		if err != nil {
			logger.Warn("failed to fetch timeline page",
				zap.Int("page", p), zap.Error(err))
			continue
		}

		logger.Debug(
			"fetched timeline page",
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Int("page", p),
			zap.Int("events", len(events)),
		)

		// Reverse iterate within each page to get newest-first
		for i := len(events) - 1; i >= 0; i-- {
			if FilteredEventTypes[events[i].Event] {
				continue
			}
			item := convertTimelineEvent(events[i])
			if item != nil {
				allItems = append(allItems, *item)
			}
		}

		// Stop if we have enough items
		if len(allItems) >= needed && p > 1 {
			hasMorePages = true
			break
		}
	}

	logger.Debug(
		"finished fetching timeline events",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("total_items", len(allItems)),
		zap.Bool("has_more_pages", hasMorePages),
	)

	return allItems, hasMorePages, nil
}

// fetchDiscussionTimelineEvents fetches timeline events using GraphQL API (for discussions).
// We use `last` to fetch the most recent comments first, then paginate from that sorted list.
// This ensures users see the most recent activity first.
func fetchDiscussionTimelineEvents(
	ctx context.Context,
	logger *zap.Logger,
	client githubinterfaces.Client,
	subjectInfo *types.SubjectInfo,
) ([]models.TimelineItem, error) {
	const maxCommentsToFetch = 100 // GitHub GraphQL caps `last` at 100

	logger.Debug(
		"fetching discussion timeline from GitHub GraphQL",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("fetch_count", maxCommentsToFetch),
	)

	timeline, hasNextPage, _, err := client.FetchDiscussionComments(
		ctx,
		subjectInfo.Owner,
		subjectInfo.Repo,
		subjectInfo.Number,
		maxCommentsToFetch,
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

	var allItems []models.TimelineItem
	for _, event := range timeline {
		item := convertTimelineEvent(event)
		if item == nil {
			continue
		}
		allItems = append(allItems, *item)
	}

	logger.Debug(
		"finished fetching discussion timeline events",
		zap.String("owner", subjectInfo.Owner),
		zap.String("repo", subjectInfo.Repo),
		zap.Int("number", subjectInfo.Number),
		zap.Int("total_items", len(allItems)),
	)

	return allItems, nil
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
	if event.Assignee != nil {
		item.AssigneeLogin = event.Assignee.Login
	}
	if event.RequestedReviewer != nil {
		item.RequestedReviewerLogin = event.RequestedReviewer.Login
	} else if event.RequestedTeam != nil {
		item.RequestedReviewerLogin = event.RequestedTeam.Name
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

	// Cross-reference source: parse from REST JSON (SourceRaw) or use pre-parsed Source
	if event.SourceRaw != nil {
		var restSource struct {
			Type  string `json:"type"`
			Issue *struct {
				Number  int    `json:"number"`
				Title   string `json:"title"`
				HTMLURL string `json:"html_url"`
				State   string `json:"state"`
			} `json:"issue"`
		}
		if json.Unmarshal(event.SourceRaw, &restSource) == nil && restSource.Issue != nil {
			sourceType := "Issue"
			if strings.Contains(restSource.Issue.HTMLURL, "/pull/") {
				sourceType = "PullRequest"
			}
			item.SourceType = sourceType
			item.SourceNumber = restSource.Issue.Number
			item.SourceTitle = restSource.Issue.Title
			item.SourceHTMLURL = restSource.Issue.HTMLURL
		}
	} else if event.Source != nil {
		item.SourceType = event.Source.Type
		item.SourceNumber = event.Source.Number
		item.SourceTitle = event.Source.Title
		item.SourceHTMLURL = event.Source.HTMLURL
	}

	// Discussion replies
	if len(event.Replies) > 0 {
		item.ReplyCount = event.ReplyCount
		item.HasMoreReplies = event.HasMoreReplies
		for _, r := range event.Replies {
			item.Replies = append(item.Replies, models.TimelineReply{
				ID:              r.ID,
				Body:            r.Body,
				AuthorLogin:     r.Author.Login,
				AuthorAvatarURL: r.Author.AvatarURL,
				CreatedAt:       r.CreatedAt,
				UpdatedAt:       r.UpdatedAt,
				HTMLURL:         r.HTMLURL,
			})
		}
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
