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

// Package notifications provides the notifications handler.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	notificationcore "github.com/octobud-hq/octobud/backend/internal/core/notification"
)

// Error definitions
// Note: Some errors are defined in notifications.go to avoid duplication
var (
	ErrFailedToMarkNotificationRead   = errors.New("failed to mark notification as read")
	ErrFailedToMarkNotificationUnread = errors.New("failed to mark notification as unread")
	ErrFailedToArchiveNotification    = errors.New("failed to archive notification")
	ErrFailedToUnarchiveNotification  = errors.New("failed to unarchive notification")
	ErrFailedToMuteNotification       = errors.New("failed to mute notification")
	ErrFailedToUnmuteNotification     = errors.New("failed to unmute notification")
	ErrFailedToSnoozeNotification     = errors.New("failed to snooze notification")
	ErrFailedToUnsnoozeNotification   = errors.New("failed to unsnooze notification")
	ErrFailedToStarNotification       = errors.New("failed to star notification")
	ErrFailedToUnstarNotification     = errors.New("failed to unstar notification")
	ErrFailedToUnfilterNotification   = errors.New("failed to unfilter notification")
	ErrFailedToDecodeRequest          = errors.New("failed to decode request")
	ErrFailedToGetNotification        = errors.New("failed to get notification")
	ErrFailedToGetTag                 = errors.New("failed to get tag")
	ErrFailedToAssignTag              = errors.New("failed to assign tag")
	ErrFailedToGetUpdatedNotification = errors.New("failed to get updated notification")
	ErrFailedToBuildResponse          = errors.New("failed to build response")
	ErrFailedToCreateTag              = errors.New("failed to create tag")
	ErrFailedToRemoveTag              = errors.New("failed to remove tag")
)

// NotificationAction represents a type of single notification action
type NotificationAction string

// NotificationAction constants
const (
	ActionMarkRead   NotificationAction = "mark-read"
	ActionMarkUnread NotificationAction = "mark-unread"
	ActionArchive    NotificationAction = "archive"
	ActionUnarchive  NotificationAction = "unarchive"
	ActionMute       NotificationAction = "mute"
	ActionUnmute     NotificationAction = "unmute"
	ActionUnsnooze   NotificationAction = "unsnooze"
	ActionStar       NotificationAction = "star"
	ActionUnstar     NotificationAction = "unstar"
	ActionUnfilter   NotificationAction = "unfilter"
)

type snoozeNotificationRequest struct {
	SnoozedUntil string `json:"snoozedUntil"`
}

type assignTagRequest struct {
	TagID string `json:"tagId"`
}

type assignTagByNameRequest struct {
	TagName string `json:"tagName"`
}

type tagActionResponse struct {
	Notification NotificationResponse `json:"notification"`
}

// handleNotificationAction handles simple notification actions that follow the standard pattern
// (read, unread, archive, unarchive, mute, unmute, star, unstar, unfilter, unsnooze)
func (h *Handler) handleNotificationAction(
	w http.ResponseWriter,
	r *http.Request,
	action NotificationAction,
) {
	ctx := r.Context()

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		h.logger.Error(
			"invalid githubID encoding",
			zap.String("github_id", rawGithubID),
			zap.Error(errors.Join(ErrInvalidGithubIDEncoding, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	// Get userID first
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	// Execute the action
	_, err = h.executeNotificationAction(ctx, userID, action, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug(
				"notification not found",
				zap.String("github_id", githubID),
				zap.String("action", string(action)),
			)
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to execute notification action",
			zap.String("github_id", githubID),
			zap.String("action", string(action)),
			zap.Error(err),
		)
		shared.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to %s notification", action),
		)
		return
	}

	// Get updated notification with details
	queryStr := r.URL.Query().Get("query")
	notification, err := h.notifications.GetNotificationWithDetails(ctx, userID, githubID, queryStr)
	if err != nil {
		h.logger.Error(
			"failed to get notification with details",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToGetNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to get notification")
		return
	}

	shared.WriteJSON(w, http.StatusOK, notificationActionResponse{Notification: notification})
}

// executeNotificationAction executes a notification action using the service
// Note: userID should already be validated by the caller
func (h *Handler) executeNotificationAction(
	ctx context.Context,
	userID string,
	action NotificationAction,
	githubID string,
) (interface{}, error) {
	switch action {
	case ActionMarkRead:
		return h.notifications.MarkNotificationRead(ctx, userID, githubID)
	case ActionMarkUnread:
		return h.notifications.MarkNotificationUnread(ctx, userID, githubID)
	case ActionArchive:
		return h.notifications.ArchiveNotification(ctx, userID, githubID)
	case ActionUnarchive:
		return h.notifications.UnarchiveNotification(ctx, userID, githubID)
	case ActionMute:
		return h.notifications.MuteNotification(ctx, userID, githubID)
	case ActionUnmute:
		return h.notifications.UnmuteNotification(ctx, userID, githubID)
	case ActionUnsnooze:
		return h.notifications.UnsnoozeNotification(ctx, userID, githubID)
	case ActionStar:
		return h.notifications.StarNotification(ctx, userID, githubID)
	case ActionUnstar:
		return h.notifications.UnstarNotification(ctx, userID, githubID)
	case ActionUnfilter:
		return h.notifications.UnfilterNotification(ctx, userID, githubID)
	default:
		return nil, fmt.Errorf("unknown notification action: %s", action)
	}
}

// Individual handler methods that delegate to the unified handler
func (h *Handler) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionMarkRead)
}

func (h *Handler) handleMarkNotificationUnread(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionMarkUnread)
}

func (h *Handler) handleArchiveNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionArchive)
}

func (h *Handler) handleUnarchiveNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionUnarchive)
}

func (h *Handler) handleMuteNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionMute)
}

func (h *Handler) handleUnmuteNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionUnmute)
}

func (h *Handler) handleUnsnoozeNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionUnsnooze)
}

func (h *Handler) handleStarNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionStar)
}

func (h *Handler) handleUnstarNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionUnstar)
}

func (h *Handler) handleUnfilterNotification(w http.ResponseWriter, r *http.Request) {
	h.handleNotificationAction(w, r, ActionUnfilter)
}

// handleSnoozeNotification snoozes a notification
// This is kept separate because it requires a request body with snoozedUntil
func (h *Handler) handleSnoozeNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		h.logger.Error(
			"invalid githubID encoding",
			zap.String("github_id", rawGithubID),
			zap.Error(errors.Join(ErrInvalidGithubIDEncoding, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	var req snoozeNotificationRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		h.logger.Error(
			"invalid request body",
			zap.Error(errors.Join(ErrFailedToDecodeRequest, decodeErr)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SnoozedUntil == "" {
		shared.WriteError(w, http.StatusBadRequest, "snoozedUntil is required")
		return
	}

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	_, err = h.notifications.SnoozeNotification(ctx, userID, githubID, req.SnoozedUntil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("notification not found", zap.String("github_id", githubID))
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to snooze notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToSnoozeNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to snooze notification")
		return
	}

	queryStr := r.URL.Query().Get("query")
	notification, err := h.notifications.GetNotificationWithDetails(ctx, userID, githubID, queryStr)
	if err != nil {
		h.logger.Error(
			"failed to get notification with details",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToGetNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to get notification")
		return
	}

	shared.WriteJSON(w, http.StatusOK, notificationActionResponse{Notification: notification})
}

// handleAssignTagToNotification assigns a tag to a notification
// This is kept separate because it requires a request body with tagID
func (h *Handler) handleAssignTagToNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	var req assignTagRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TagID == "" {
		shared.WriteError(w, http.StatusBadRequest, "tagId is required")
		return
	}

	// Assign the tag using service
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	_, err = h.notifications.AssignTag(ctx, userID, githubID, req.TagID)
	if err != nil {
		if errors.Is(err, notificationcore.ErrNotificationNotFound) {
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		if errors.Is(err, notificationcore.ErrTagNotFound) {
			shared.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("tag with id %s not found", req.TagID),
			)
			return
		}
		h.logger.Error(
			"failed to assign tag",
			zap.String("tag_id", req.TagID),
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToAssignTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to assign tag")
		return
	}

	queryStr := r.URL.Query().Get("query")
	notification, err := h.notifications.GetNotificationWithDetails(ctx, userID, githubID, queryStr)
	if err != nil {
		h.logger.Error(
			"failed to get updated notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToGetUpdatedNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to get updated notification")
		return
	}

	shared.WriteJSON(w, http.StatusOK, tagActionResponse{Notification: notification})
}

// handleAssignTagByName assigns a tag to a notification by name, creating it if it doesn't exist
// This is kept separate because it requires a request body with tagName and has different logic
func (h *Handler) handleAssignTagByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	var req assignTagByNameRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TagName == "" {
		shared.WriteError(w, http.StatusBadRequest, "tagName is required")
		return
	}

	// Assign tag by name using service (creates tag if needed)
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	_, err = h.notifications.AssignTagByName(ctx, userID, githubID, req.TagName)
	if err != nil {
		if errors.Is(err, notificationcore.ErrNotificationNotFound) {
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		if errors.Is(err, notificationcore.ErrInvalidTagName) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error(
			"failed to assign tag",
			zap.String("tag_name", req.TagName),
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToAssignTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to assign tag")
		return
	}

	queryStr := r.URL.Query().Get("query")
	notification, err := h.notifications.GetNotificationWithDetails(ctx, userID, githubID, queryStr)
	if err != nil {
		h.logger.Error(
			"failed to get updated notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToGetUpdatedNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to get updated notification")
		return
	}

	shared.WriteJSON(w, http.StatusOK, tagActionResponse{Notification: notification})
}

// handleRemoveTagFromNotification removes a tag from a notification
// This is kept separate because it gets tagID from URL path instead of request body
func (h *Handler) handleRemoveTagFromNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawGithubID := chi.URLParam(r, "githubID")
	if rawGithubID == "" {
		shared.WriteError(w, http.StatusBadRequest, "githubID is required")
		return
	}

	githubID, err := url.PathUnescape(rawGithubID)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid githubID encoding")
		return
	}

	tagIDStr := chi.URLParam(r, "tagId")
	if tagIDStr == "" {
		shared.WriteError(w, http.StatusBadRequest, "tagId is required")
		return
	}

	// Remove tag using service
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	_, err = h.notifications.RemoveTag(ctx, userID, githubID, tagIDStr)
	if err != nil {
		if errors.Is(err, notificationcore.ErrNotificationNotFound) {
			shared.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to remove tag",
			zap.String("tag_id", tagIDStr),
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToRemoveTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to remove tag")
		return
	}

	queryStr := r.URL.Query().Get("query")
	notification, err := h.notifications.GetNotificationWithDetails(ctx, userID, githubID, queryStr)
	if err != nil {
		h.logger.Error(
			"failed to get updated notification",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToGetUpdatedNotification, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to get updated notification")
		return
	}

	shared.WriteJSON(w, http.StatusOK, tagActionResponse{Notification: notification})
}
