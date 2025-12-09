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

package user

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// HandleGetSyncSettings handles GET /api/user/sync-settings
func (h *Handler) HandleGetSyncSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, err := h.authSvc.GetUserSyncSettings(ctx)
	if err != nil {
		h.logger.Error("failed to get sync settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := SyncSettingsResponse{}
	if settings != nil {
		response = SyncSettingsResponse{
			InitialSyncDays:       settings.InitialSyncDays,
			InitialSyncMaxCount:   settings.InitialSyncMaxCount,
			InitialSyncUnreadOnly: settings.InitialSyncUnreadOnly,
			SetupCompleted:        settings.SetupCompleted,
		}
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleUpdateSyncSettings handles PUT /api/user/sync-settings
func (h *Handler) HandleUpdateSyncSettings(w http.ResponseWriter, r *http.Request) {
	var req SyncSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode sync settings request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate sync settings
	// Upper bounds prevent abuse and unreasonable values
	const maxSyncDays = 3650 // 10 years
	const maxNotificationCount = 100000

	if req.InitialSyncDays != nil {
		if *req.InitialSyncDays < 1 {
			shared.WriteError(w, http.StatusBadRequest, "initialSyncDays must be at least 1")
			return
		}
		if *req.InitialSyncDays > maxSyncDays {
			shared.WriteError(
				w,
				http.StatusBadRequest,
				"initialSyncDays cannot exceed 3650 (10 years)",
			)
			return
		}
	}

	if req.InitialSyncMaxCount != nil {
		if *req.InitialSyncMaxCount < 1 {
			shared.WriteError(w, http.StatusBadRequest, "initialSyncMaxCount must be at least 1")
			return
		}
		if *req.InitialSyncMaxCount > maxNotificationCount {
			shared.WriteError(w, http.StatusBadRequest, "initialSyncMaxCount cannot exceed 100000")
			return
		}
	}

	ctx := r.Context()

	// Get userID first for scheduler calls
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	// Check current sync settings to see if setup was already completed
	// We only want to trigger sync on initial setup completion, not on subsequent updates
	var wasAlreadyCompleted bool
	currentSettings, err := h.authSvc.GetUserSyncSettings(ctx)
	if err == nil && currentSettings != nil {
		wasAlreadyCompleted = currentSettings.SetupCompleted
	}

	settings := &models.SyncSettings{
		InitialSyncDays:       req.InitialSyncDays,
		InitialSyncMaxCount:   req.InitialSyncMaxCount,
		InitialSyncUnreadOnly: req.InitialSyncUnreadOnly,
		SetupCompleted:        req.SetupCompleted,
	}

	if err := h.authSvc.UpdateUserSyncSettings(ctx, settings); err != nil {
		h.logger.Error("failed to update sync settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Queue a sync job if setup was just completed (first time) and scheduler is available
	if req.SetupCompleted && !wasAlreadyCompleted && h.scheduler != nil {
		if err := h.scheduler.EnqueueSyncNotifications(ctx, userID); err != nil {
			h.logger.Warn("failed to queue sync job after completing setup", zap.Error(err))
		} else {
			h.logger.Info("queued sync job after initial setup completion")
		}
	}

	response := SyncSettingsResponse{
		InitialSyncDays:       settings.InitialSyncDays,
		InitialSyncMaxCount:   settings.InitialSyncMaxCount,
		InitialSyncUnreadOnly: settings.InitialSyncUnreadOnly,
		SetupCompleted:        settings.SetupCompleted,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleGetSyncState handles GET /api/user/sync-state
// Returns the current sync state including oldest_notification_synced_at
func (h *Handler) HandleGetSyncState(w http.ResponseWriter, r *http.Request) {
	if h.syncStateSvc == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, "Sync state service not available")
		return
	}

	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	state, err := h.syncStateSvc.GetSyncState(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get sync state", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := SyncStateResponse{}
	if state.OldestNotificationSyncedAt.Valid {
		formatted := state.OldestNotificationSyncedAt.Time.Format(time.RFC3339)
		response.OldestNotificationSyncedAt = &formatted
	}
	if state.InitialSyncCompletedAt.Valid {
		formatted := state.InitialSyncCompletedAt.Time.Format(time.RFC3339)
		response.InitialSyncCompletedAt = &formatted
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleSyncOlder handles POST /api/user/sync-older
// Queues a job to sync notifications older than the current oldest synced notification
func (h *Handler) HandleSyncOlder(w http.ResponseWriter, r *http.Request) {
	var req SyncOlderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode sync older request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	const maxSyncDays = 3650 // 10 years
	const maxNotificationCount = 100000

	if req.Days < 1 {
		shared.WriteError(w, http.StatusBadRequest, "days must be at least 1")
		return
	}
	if req.Days > maxSyncDays {
		shared.WriteError(w, http.StatusBadRequest, "days cannot exceed 3650 (10 years)")
		return
	}

	if req.MaxCount != nil {
		if *req.MaxCount < 1 {
			shared.WriteError(w, http.StatusBadRequest, "maxCount must be at least 1")
			return
		}
		if *req.MaxCount > maxNotificationCount {
			shared.WriteError(w, http.StatusBadRequest, "maxCount cannot exceed 100000")
			return
		}
	}

	// Check if sync state service is available
	if h.syncStateSvc == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, "Sync state service not available")
		return
	}

	// Check if scheduler is available
	if h.scheduler == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, "Job queue not available")
		return
	}

	ctx := r.Context()

	var untilTime time.Time

	// If BeforeDate is provided, use it as the until time (override)
	if req.BeforeDate != nil && *req.BeforeDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.BeforeDate)
		if err != nil {
			h.logger.Debug(
				"failed to parse beforeDate",
				zap.String("beforeDate", *req.BeforeDate),
				zap.Error(err),
			)
			shared.WriteError(
				w,
				http.StatusBadRequest,
				"beforeDate must be in RFC3339 format (e.g., 2024-01-15T00:00:00Z)",
			)
			return
		}
		untilTime = parsedTime
		h.logger.Info("using provided beforeDate override",
			zap.Time("beforeDate", untilTime))
	} else {
		// Fall back to oldest_notification_synced_at from sync state
		userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
		if !ok {
			return
		}
		state, err := h.syncStateSvc.GetSyncState(ctx, userID)
		if err != nil {
			h.logger.Error("failed to get sync state", zap.Error(err))
			shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		// Check if we have an oldest notification timestamp
		if !state.OldestNotificationSyncedAt.Valid {
			shared.WriteError(
				w,
				http.StatusBadRequest,
				"No notifications have been synced yet. Complete initial setup first, or provide a beforeDate.",
			)
			return
		}

		untilTime = state.OldestNotificationSyncedAt.Time
	}

	// Queue the sync older job
	err := h.scheduler.EnqueueSyncOlder(ctx, jobs.SyncOlderNotificationsArgs{
		Days:       req.Days,
		UntilTime:  untilTime,
		MaxCount:   req.MaxCount,
		UnreadOnly: req.UnreadOnly,
	})
	if err != nil {
		h.logger.Error("failed to queue sync older job", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to queue sync job")
		return
	}

	h.logger.Info("queued sync older notifications job",
		zap.Int("days", req.Days),
		zap.Time("untilTime", untilTime),
		zap.Any("maxCount", req.MaxCount),
		zap.Bool("unreadOnly", req.UnreadOnly),
		zap.Bool("usedBeforeDateOverride", req.BeforeDate != nil && *req.BeforeDate != ""))

	w.WriteHeader(http.StatusAccepted)
}
