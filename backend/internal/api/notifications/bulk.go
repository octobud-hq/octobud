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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToDecodeBulkRequest = errors.New("failed to decode bulk request")
	ErrFailedToBuildQuery        = errors.New("failed to build query")
	ErrFailedToListNotifications = errors.New("failed to list notifications")
)

// BulkOperation represents a type of bulk operation
type BulkOperation string

// BulkOperation constants
const (
	BulkOpMarkRead   BulkOperation = "mark-read"
	BulkOpMarkUnread BulkOperation = "mark-unread"
	BulkOpArchive    BulkOperation = "archive"
	BulkOpUnarchive  BulkOperation = "unarchive"
	BulkOpMute       BulkOperation = "mute"
	BulkOpUnmute     BulkOperation = "unmute"
	BulkOpStar       BulkOperation = "star"
	BulkOpUnstar     BulkOperation = "unstar"
	BulkOpUnfilter   BulkOperation = "unfilter"
	BulkOpUnsnooze   BulkOperation = "unsnooze"
)

type bulkMarkNotificationsRequest struct {
	GithubIDs []string `json:"githubIDs,omitempty"`
	Query     string   `json:"query,omitempty"`
}

type bulkTagNotificationsRequest struct {
	GithubIDs []string `json:"githubIDs,omitempty"`
	TagID     string   `json:"tagId"`
	Query     string   `json:"query,omitempty"`
}

// handleBulkOperation handles bulk operations that follow the standard pattern
// (read, unread, archive, unarchive, mute, unmute, star, unstar, unfilter, unsnooze)
func (h *Handler) handleBulkOperation(w http.ResponseWriter, r *http.Request, op BulkOperation) {
	ctx := r.Context()

	var req bulkMarkNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error(
			"failed to decode bulk request",
			zap.String("operation", string(op)),
			zap.Error(errors.Join(ErrFailedToDecodeBulkRequest, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate that either GithubIDs or Query is provided (explicitly), but not both
	// Note: An empty query string is valid and represents inbox semantics
	hasQuery := req.Query != "" || (req.Query == "" && len(req.GithubIDs) == 0)
	hasIDs := len(req.GithubIDs) > 0

	// Check if neither was provided
	if !hasQuery && !hasIDs {
		h.logger.Debug(
			"validation error: neither githubIDs nor query provided",
			zap.String("operation", string(op)),
		)
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"either 'query' or 'githubIDs' must be provided",
		)
		return
	}

	// Check if both were provided
	if hasQuery && hasIDs {
		h.logger.Debug(
			"validation error: both githubIDs and query provided",
			zap.String("operation", string(op)),
		)
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"provide either 'query' or 'githubIDs', not both",
		)
		return
	}

	var count int64
	var err error
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	if hasQuery {
		count, err = h.executeBulkOperationByQuery(ctx, userID, op, req.Query)
	} else {
		count, err = h.executeBulkOperationByIDs(ctx, userID, op, req.GithubIDs)
	}

	if err != nil {
		h.logger.Error(
			"failed to execute bulk operation",
			zap.String("operation", string(op)),
			zap.Error(err),
		)
		if errors.Is(err, notification.ErrNoNotificationIDs) {
			shared.WriteError(w, http.StatusBadRequest, "no notification ids provided")
			return
		}
		shared.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("failed to %s notifications", op),
		)
		return
	}

	shared.WriteJSON(w, http.StatusOK, bulkNotificationsResponse{Count: int(count)})
}

// executeBulkOperationByQuery executes a bulk operation using a query string
func (h *Handler) executeBulkOperationByQuery(
	ctx context.Context,
	userID string,
	op BulkOperation,
	queryStr string,
) (int64, error) {
	return h.notifications.BulkUpdate(
		ctx,
		userID,
		models.BulkOperationType(op),
		models.BulkOperationTarget{Query: queryStr},
		models.BulkUpdateParams{},
	)
}

// executeBulkOperationByIDs executes a bulk operation using notification IDs
func (h *Handler) executeBulkOperationByIDs(
	ctx context.Context,
	userID string,
	op BulkOperation,
	githubIDs []string,
) (int64, error) {
	return h.notifications.BulkUpdate(
		ctx,
		userID,
		models.BulkOperationType(op),
		models.BulkOperationTarget{IDs: githubIDs},
		models.BulkUpdateParams{},
	)
}

// Individual handler methods that delegate to the unified handler
func (h *Handler) handleBulkMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpMarkRead)
}

func (h *Handler) handleBulkMarkNotificationsUnread(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpMarkUnread)
}

func (h *Handler) handleBulkArchiveNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpArchive)
}

func (h *Handler) handleBulkUnarchiveNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpUnarchive)
}

func (h *Handler) handleBulkMuteNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpMute)
}

func (h *Handler) handleBulkUnmuteNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpUnmute)
}

func (h *Handler) handleBulkStarNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpStar)
}

func (h *Handler) handleBulkUnstarNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpUnstar)
}

func (h *Handler) handleBulkUnfilterNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpUnfilter)
}

func (h *Handler) handleBulkUnsnoozeNotifications(w http.ResponseWriter, r *http.Request) {
	h.handleBulkOperation(w, r, BulkOpUnsnooze)
}

// handleBulkAssignTag assigns a tag to multiple notifications
// This is kept separate because it has different request structure and logic
func (h *Handler) handleBulkAssignTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req bulkTagNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error(
			"failed to decode request",
			zap.Error(errors.Join(ErrFailedToDecodeBulkRequest, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TagID == "" {
		h.logger.Debug(
			"validation error: tagId required",
			zap.String("operation", "bulk assign tag"),
		)
		shared.WriteError(w, http.StatusBadRequest, "tagId is required")
		return
	}

	// Validate that either GithubIDs or Query is provided
	// Note: Empty query string is valid (for "select all" semantics)
	hasQuery := req.Query != "" || (req.Query == "" && len(req.GithubIDs) == 0)
	hasIDs := len(req.GithubIDs) > 0

	// Check if both were provided (explicitly)
	if hasQuery && hasIDs && req.Query != "" {
		h.logger.Debug(
			"validation error: both githubIDs and query provided",
			zap.String("operation", "bulk assign tag"),
		)
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"provide either 'query' or 'githubIDs', not both",
		)
		return
	}

	// Verify the tag exists
	_, err := h.tagSvc.GetTag(ctx, userID, req.TagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Debug("tag not found", zap.String("tag_id", req.TagID))
			shared.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("tag with id %s not found", req.TagID),
			)
			return
		}
		h.logger.Error(
			"failed to get tag",
			zap.String("tag_id", req.TagID),
			zap.Error(errors.Join(ErrFailedToGetTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to get tag")
		return
	}

	var notifications []db.Notification
	if hasQuery {
		// Use service to list notifications from query string
		notifications, err = h.notifications.ListNotificationsFromQueryString(
			ctx,
			userID,
			req.Query,
			999999,
		)
		if err != nil {
			h.logger.Error(
				"failed to list notifications",
				zap.Error(errors.Join(ErrFailedToListNotifications, err)),
			)
			shared.WriteError(w, http.StatusInternalServerError, "failed to list notifications")
			return
		}
	} else {
		// Get notifications by IDs
		notifications = make([]db.Notification, 0, len(req.GithubIDs))
		for _, githubID := range req.GithubIDs {
			notification, getErr := h.notifications.GetByGithubID(ctx, userID, githubID)
			if getErr != nil {
				if errors.Is(getErr, sql.ErrNoRows) {
					continue // Skip missing notifications
				}
				h.logger.Warn(
					"failed to get notification",
					zap.String("github_id", githubID),
					zap.Error(errors.Join(ErrFailedToGetNotification, err)),
				)
				continue
			}
			notifications = append(notifications, notification)
		}
	}

	if len(notifications) == 0 {
		shared.WriteJSON(w, http.StatusOK, bulkNotificationsResponse{Count: 0})
		return
	}

	// Assign tag to all notifications using service
	count, err := h.notifications.BulkAssignTag(ctx, userID, notifications, req.TagID)
	if err != nil {
		h.logger.Error(
			"failed to bulk assign tag",
			zap.String("tag_id", req.TagID),
			zap.Error(errors.Join(ErrFailedToAssignTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to assign tag")
		return
	}

	shared.WriteJSON(w, http.StatusOK, bulkNotificationsResponse{Count: count})
}

// handleBulkRemoveTag removes a tag from multiple notifications
// This is kept separate because it has different request structure and logic
func (h *Handler) handleBulkRemoveTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req bulkTagNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error(
			"failed to decode request",
			zap.Error(errors.Join(ErrFailedToDecodeBulkRequest, err)),
		)
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TagID == "" {
		h.logger.Debug(
			"validation error: tagId required",
			zap.String("operation", "bulk remove tag"),
		)
		shared.WriteError(w, http.StatusBadRequest, "tagId is required")
		return
	}

	// Validate that either GithubIDs or Query is provided
	// Note: Empty query string is valid (for "select all" semantics)
	hasQuery := req.Query != "" || (req.Query == "" && len(req.GithubIDs) == 0)
	hasIDs := len(req.GithubIDs) > 0

	// Check if both were provided (explicitly)
	if hasQuery && hasIDs && req.Query != "" {
		h.logger.Debug(
			"validation error: both githubIDs and query provided",
			zap.String("operation", "bulk remove tag"),
		)
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"provide either 'query' or 'githubIDs', not both",
		)
		return
	}

	var notifications []db.Notification
	var err error
	if hasQuery {
		// Use service to list notifications from query string
		notifications, err = h.notifications.ListNotificationsFromQueryString(
			ctx,
			userID,
			req.Query,
			999999,
		)
		if err != nil {
			h.logger.Error(
				"failed to list notifications",
				zap.Error(errors.Join(ErrFailedToListNotifications, err)),
			)
			shared.WriteError(w, http.StatusInternalServerError, "failed to list notifications")
			return
		}
	} else {
		// Get notifications by IDs
		notifications = make([]db.Notification, 0, len(req.GithubIDs))
		for _, githubID := range req.GithubIDs {
			notification, getErr := h.notifications.GetByGithubID(ctx, userID, githubID)
			if getErr != nil {
				if errors.Is(getErr, sql.ErrNoRows) {
					continue // Skip missing notifications
				}
				h.logger.Warn(
					"failed to get notification",
					zap.String("github_id", githubID),
					zap.Error(errors.Join(ErrFailedToGetNotification, err)),
				)
				continue
			}
			notifications = append(notifications, notification)
		}
	}

	if len(notifications) == 0 {
		shared.WriteJSON(w, http.StatusOK, bulkNotificationsResponse{Count: 0})
		return
	}

	// Remove tag from all notifications using service
	count, err := h.notifications.BulkRemoveTag(ctx, userID, notifications, req.TagID)
	if err != nil {
		h.logger.Error(
			"failed to bulk remove tag",
			zap.String("tag_id", req.TagID),
			zap.Error(errors.Join(ErrFailedToRemoveTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to remove tag")
		return
	}

	shared.WriteJSON(w, http.StatusOK, bulkNotificationsResponse{Count: count})
}
