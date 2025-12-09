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
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToParseListOptions          = errors.New("failed to parse notification list options")
	ErrFailedToLoadNotifications         = errors.New("failed to load notifications")
	ErrFailedToLoadRepositories          = errors.New("failed to load repositories")
	ErrFailedToBuildNotificationResponse = errors.New("failed to build notification response")
	ErrInvalidGithubIDEncoding           = errors.New("invalid githubID encoding")
	ErrFailedToLoadNotification          = errors.New("failed to load notification")
	ErrSyncServiceNotAvailable           = errors.New("sync service not available")
	ErrFailedToRefreshSubjectData        = errors.New("failed to refresh subject data")
	ErrFailedToLoadUpdatedNotification   = errors.New("failed to load updated notification")
	ErrFailedToFetchNotification         = errors.New("failed to fetch notification")
	ErrFailedToParseSubjectInfo          = errors.New("failed to parse subject info")
	ErrGitHubClientNotConfigured         = errors.New("GitHub client not configured")
	ErrGitHubClientTypeMismatch          = errors.New("GitHub client type mismatch")
	ErrFailedToFetchTimeline             = errors.New("failed to fetch timeline")
	ErrFailedToFetchTags                 = errors.New("failed to fetch tags")
)

func (h *Handler) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := helpers.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	options := parseNotificationListOptions(r)

	result, err := h.notifications.ListNotifications(ctx, userID, options)
	if err != nil {
		// Check if this is an invalid query error
		if errors.Is(err, notification.ErrInvalidQuery) {
			h.logger.Warn(
				"invalid query in notification list request",
				zap.String("query", options.Query),
				zap.Error(err),
			)
			// Extract the error message for the client
			errorMsg := getQueryErrorMessage(err)
			helpers.WriteError(w, http.StatusBadRequest, errorMsg)
			return
		}

		// For all listing errors, return 400 with inline error message
		// This allows the frontend to show the error inline rather than blocking the view
		h.logger.Error(
			"failed to load notifications",
			zap.Error(errors.Join(ErrFailedToLoadNotifications, err)),
		)
		errorMsg := "Failed to load notifications"
		if err.Error() != "" {
			errorMsg = err.Error()
		}
		helpers.WriteError(w, http.StatusBadRequest, errorMsg)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, listNotificationsResponse{
		Notifications: result.Notifications,
		Total:         result.Total,
		Page:          result.Page,
		PageSize:      result.PageSize,
	})
}

func (h *Handler) handlePollNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := helpers.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	options := parseNotificationListOptions(r)

	result, err := h.notifications.ListPollNotifications(ctx, userID, options)
	if err != nil {
		h.logger.Error(
			"failed to load poll notifications",
			zap.Error(errors.Join(ErrFailedToLoadNotifications, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to load notifications")
		return
	}

	// Convert models.PollNotification to PollNotificationResponse
	notifications := make([]PollNotificationResponse, 0, len(result.Notifications))
	for _, n := range result.Notifications {
		notifications = append(notifications, PollNotificationResponse{
			ID:                n.ID,
			GithubID:          n.GithubID,
			EffectiveSortDate: n.EffectiveSortDate,
			Archived:          n.Archived,
			Muted:             n.Muted,
			RepoFullName:      n.RepoFullName,
			SubjectTitle:      n.SubjectTitle,
			SubjectType:       n.SubjectType,
			Reason:            n.Reason,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, listPollNotificationsResponse{
		Notifications: notifications,
		Total:         result.Total,
		Page:          result.Page,
		PageSize:      result.PageSize,
	})
}

func (h *Handler) handleGetNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := helpers.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		helpers.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		h.logger.Error(
			"invalid githubID encoding",
			zap.String("github_id", rawGithubID),
			zap.Error(errors.Join(ErrInvalidGithubIDEncoding, err)),
		)
		helpers.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	queryStr := r.URL.Query().Get("query")
	notif, err := h.notifications.GetNotificationWithDetails(ctx, userID, githubID, queryStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("notification not found", zap.String("github_id", githubID))
			helpers.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to load notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToLoadNotification, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to load notification")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, notificationDetailResponse{Notification: notif})
}

func (h *Handler) handleRefreshNotificationSubject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := helpers.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	// Check if sync service is available
	if h.syncService == nil {
		h.logger.Warn(
			"sync service not available for refresh",
			zap.Error(ErrSyncServiceNotAvailable),
		)
		helpers.WriteError(w, http.StatusServiceUnavailable, "subject refresh not available")
		return
	}

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		helpers.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		h.logger.Error(
			"invalid githubID encoding",
			zap.String("github_id", rawGithubID),
			zap.Error(errors.Join(ErrInvalidGithubIDEncoding, err)),
		)
		helpers.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	// Verify the notification exists
	_, err = h.notifications.GetByGithubID(ctx, userID, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("notification not found for refresh", zap.String("github_id", githubID))
			helpers.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to load notification for refresh",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToLoadNotification, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to load notification")
		return
	}

	// Refresh the subject by fetching fresh data from GitHub
	err = h.refreshSubjectData(ctx, userID, githubID)
	if err != nil {
		// Check if this is a 403 Forbidden error (likely due to organization restrictions)
		errStr := err.Error()
		if strings.Contains(errStr, "status 403") || strings.Contains(errStr, "403") {
			h.logger.Warn(
				"forbidden when refreshing subject data (likely organization restriction)",
				zap.String("github_id", githubID),
				zap.Error(errors.Join(ErrFailedToRefreshSubjectData, err)),
			)
			helpers.WriteError(
				w,
				http.StatusForbidden,
				"insufficient permissions to access repository details",
			)
			return
		}
		h.logger.Error(
			"failed to refresh subject data",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToRefreshSubjectData, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to refresh subject data")
		return
	}

	// Get the updated notification with details
	queryStr := r.URL.Query().Get("query")
	updatedNotification, err := h.notifications.GetNotificationWithDetails(
		ctx,
		userID,
		githubID,
		queryStr,
	)
	if err != nil {
		h.logger.Error(
			"failed to load updated notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToLoadUpdatedNotification, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to load updated notification")
		return
	}

	helpers.WriteJSON(
		w,
		http.StatusOK,
		refreshNotificationSubjectResponse{Notification: updatedNotification},
	)
}

func parseNotificationListOptions(r *http.Request) models.ListOptions {
	query := r.URL.Query()

	opts := models.ListOptions{
		Query: query.Get(
			"query",
		), // Combined query string with key-value pairs and free text
		Page:     parseIntDefault(query.Get("page")),
		PageSize: parseIntDefault(query.Get("pageSize")),
		IncludeSubject: parseBoolDefault(
			query.Get("includeSubject"),
		), // Default: false to reduce payload size
	}

	return opts
}

func parseIntDefault(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func parseBoolDefault(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return raw == "true" || raw == "1" || raw == "yes"
}

// getQueryErrorMessage extracts a user-friendly error message from a query validation error
func getQueryErrorMessage(err error) string {
	if err == nil {
		return "Invalid query"
	}

	// Try to extract the underlying error message
	errStr := err.Error()

	// Remove common error prefixes to get cleaner messages
	errStr = strings.TrimPrefix(errStr, "invalid query: ")
	errStr = strings.TrimPrefix(errStr, "parse failed: ")
	errStr = strings.TrimPrefix(errStr, "tokenization failed: ")
	errStr = strings.TrimPrefix(errStr, "SQL generation failed: ")
	errStr = strings.TrimPrefix(errStr, "validation failed: ")

	// If the error message is too generic, provide a better one
	if errStr == "invalid query" ||
		errStr == "parse failed" ||
		errStr == "tokenization failed" ||
		errStr == "SQL generation failed" ||
		errStr == "validation failed" {
		return "Invalid query syntax"
	}

	return errStr
}

// refreshSubjectData fetches fresh subject data from GitHub and updates the notification
func (h *Handler) refreshSubjectData(ctx context.Context, userID, githubID string) error {
	if h.syncService == nil {
		return ErrSyncServiceNotAvailable
	}

	err := h.syncService.RefreshSubjectData(ctx, userID, githubID)
	if err != nil {
		return errors.Join(ErrFailedToRefreshSubjectData, err)
	}
	return nil
}
