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
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToDecodeBulkSnoozeRequest = errors.New("failed to decode bulk snooze request")
	ErrFailedToBulkSnooze              = errors.New("failed to bulk snooze")
)

type bulkSnoozeNotificationsRequest struct {
	GithubIDs    []string `json:"githubIDs,omitempty"`
	SnoozedUntil string   `json:"snoozedUntil"`
	Query        string   `json:"query,omitempty"`
}

// handleBulkSnoozeNotifications snoozes multiple notifications
func (h *Handler) handleBulkSnoozeNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req bulkSnoozeNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error(
			"failed to decode request",
			zap.Error(errors.Join(ErrFailedToDecodeBulkSnoozeRequest, err)),
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
			zap.String("operation", "bulk snooze"),
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
			zap.String("operation", "bulk snooze"),
		)
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"provide either 'query' or 'githubIDs', not both",
		)
		return
	}

	if req.SnoozedUntil == "" {
		// snoozedUntil validation handled by WriteError
		shared.WriteError(w, http.StatusBadRequest, "snoozedUntil is required")
		return
	}

	var count int64
	var err error
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	if hasQuery {
		count, err = h.notifications.BulkUpdate(
			ctx,
			userID,
			models.BulkOpSnooze,
			models.BulkOperationTarget{Query: req.Query},
			models.BulkUpdateParams{SnoozedUntil: req.SnoozedUntil},
		)
	} else {
		count, err = h.notifications.BulkUpdate(
			ctx,
			userID,
			models.BulkOpSnooze,
			models.BulkOperationTarget{IDs: req.GithubIDs},
			models.BulkUpdateParams{SnoozedUntil: req.SnoozedUntil},
		)
	}

	if err != nil {
		h.logger.Error("failed to bulk snooze", zap.Error(errors.Join(ErrFailedToBulkSnooze, err)))
		if errors.Is(err, notification.ErrNoNotificationIDs) {
			shared.WriteError(w, http.StatusBadRequest, "no notification ids provided")
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to snooze notifications")
		return
	}

	shared.WriteJSON(w, http.StatusOK, bulkNotificationsResponse{Count: int(count)})
}

// handleBulkUnsnoozeNotifications is now handled by the unified bulk handler in bulk.go
