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

package notifications

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	"github.com/octobud-hq/octobud/backend/internal/core/timeline"
	"github.com/octobud-hq/octobud/backend/internal/github"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func (h *Handler) handleGetNotificationTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	githubID := chi.URLParam(r, "githubID")

	// Parse query parameters
	perPageStr := r.URL.Query().Get("per_page")
	pageStr := r.URL.Query().Get("page")

	perPage := 5
	page := 1

	if perPageStr != "" {
		if val, err := strconv.Atoi(perPageStr); err == nil && val > 0 {
			perPage = val
		}
	}

	if pageStr != "" {
		if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
			page = val
		}
	}

	// Fetch notification from database
	notification, err := h.notifications.GetByGithubID(ctx, userID, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("notification not found for timeline", zap.String("github_id", githubID))
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}

		h.logger.Error(
			"failed to fetch notification for timeline",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToFetchNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to fetch notification")
		return
	}

	// Check if this notification type supports timeline before attempting to fetch
	// This is important for Discussions which don't have a REST API timeline endpoint
	if !timeline.SupportsTimeline(notification.SubjectType) {
		h.logger.Debug(
			"timeline not supported for notification type",
			zap.String("github_id", githubID),
			zap.String("subject_type", notification.SubjectType),
		)
		shared.WriteJSON(w, http.StatusOK, TimelineResponse{
			Items:   []ThreadItem{},
			Total:   0,
			Page:    page,
			PerPage: perPage,
			HasMore: false,
		})
		return
	}

	// Extract subject info
	subjectURL := ""
	if notification.SubjectURL.Valid {
		subjectURL = notification.SubjectURL.String
	}

	var subjectRaw json.RawMessage
	if notification.SubjectRaw.Valid {
		subjectRaw = notification.SubjectRaw.RawMessage
	}

	subjectInfo, err := github.ExtractSubjectInfo(subjectURL, subjectRaw)
	if err != nil {
		h.logger.Error(
			"failed to parse subject info",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToParseSubjectInfo, err)),
		)
		shared.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf("could not parse subject info: %v", err),
		)
		return
	}

	// Ensure we have a GitHub client and timeline service
	if h.githubClient == nil || h.timelineSvc == nil {
		h.logger.Error(
			"GitHub client or timeline service not configured",
			zap.String("github_id", githubID),
			zap.Error(ErrGitHubClientNotConfigured),
		)
		shared.WriteError(w, http.StatusInternalServerError, "GitHub client not configured")
		return
	}

	// Fetch filtered timeline using the core timeline service
	result, err := h.timelineSvc.FetchFilteredTimeline(
		ctx,
		h.githubClient,
		subjectInfo,
		perPage,
		page,
	)
	if err != nil {
		h.logger.Error(
			"failed to fetch timeline",
			zap.String("github_id", githubID),
			zap.Error(err),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to fetch timeline")
		return
	}

	// Convert timeline items to API response format
	items := make([]ThreadItem, len(result.Items))
	for i, item := range result.Items {
		items[i] = convertTimelineItemToThreadItem(item)
	}

	shared.WriteJSON(w, http.StatusOK, TimelineResponse{
		Items:   items,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
		HasMore: result.HasMore,
	})
}

// convertTimelineItemToThreadItem converts a core timeline item to an API thread item
func convertTimelineItemToThreadItem(item models.TimelineItem) ThreadItem {
	threadItem := ThreadItem{
		Type: item.Event,
		ID:   item.ID,
		Body: item.Body,
		Author: ThreadAuthor{
			Login:     item.AuthorLogin,
			AvatarURL: item.AuthorAvatarURL,
		},
		HTMLURL: item.HTMLURL,
	}

	// Set timestamps
	if item.CreatedAt != nil {
		createdAt := item.CreatedAt.Format(time.RFC3339)
		threadItem.CreatedAt = &createdAt
	}
	if item.UpdatedAt != nil {
		updatedAt := item.UpdatedAt.Format(time.RFC3339)
		threadItem.UpdatedAt = &updatedAt
	}
	if item.SubmittedAt != nil {
		submittedAt := item.SubmittedAt.Format(time.RFC3339)
		threadItem.SubmittedAt = &submittedAt
	}

	// Set type-specific fields
	if item.State != "" {
		threadItem.State = &item.State
	}
	if item.Message != "" {
		threadItem.Message = &item.Message
	}
	if item.SHA != "" {
		threadItem.SHA = &item.SHA
	}

	return threadItem
}
