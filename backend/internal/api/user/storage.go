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

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// StorageStatsResponse represents the response for storage statistics
type StorageStatsResponse struct {
	TotalCount         int64 `json:"totalCount"`
	ArchivedCount      int64 `json:"archivedCount"`
	StarredCount       int64 `json:"starredCount"`
	SnoozedCount       int64 `json:"snoozedCount"`
	UnreadCount        int64 `json:"unreadCount"`
	TaggedCount        int64 `json:"taggedCount"`
	EligibleForCleanup int64 `json:"eligibleForCleanup"`
}

// RetentionSettingsResponse represents the response for retention settings
type RetentionSettingsResponse struct {
	Enabled        bool   `json:"enabled"`
	RetentionDays  int    `json:"retentionDays"`
	ProtectStarred bool   `json:"protectStarred"`
	ProtectTagged  bool   `json:"protectTagged"`
	LastCleanupAt  string `json:"lastCleanupAt,omitempty"`
}

// RetentionSettingsRequest represents the request for updating retention settings
type RetentionSettingsRequest struct {
	Enabled        bool `json:"enabled"`
	RetentionDays  int  `json:"retentionDays"`
	ProtectStarred bool `json:"protectStarred"`
	ProtectTagged  bool `json:"protectTagged"`
}

// CleanupRequest represents the request for manual cleanup
type CleanupRequest struct {
	RetentionDays  int  `json:"retentionDays"`
	ProtectStarred bool `json:"protectStarred"`
	ProtectTagged  bool `json:"protectTagged"`
}

// CleanupResponse represents the response for cleanup operation
type CleanupResponse struct {
	NotificationsDeleted int64 `json:"notificationsDeleted"`
	PullRequestsDeleted  int64 `json:"pullRequestsDeleted"`
}

// HandleGetStorageStats handles GET /api/user/storage-stats
func (h *Handler) HandleGetStorageStats(w http.ResponseWriter, r *http.Request) {
	// Get cleanup handler from scheduler
	cleanupHandler := h.getCleanupHandler()
	if cleanupHandler == nil {
		shared.WriteError(w, http.StatusInternalServerError, "Cleanup handler not configured")
		return
	}

	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	// Get storage stats
	stats, err := cleanupHandler.GetStorageStats(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get storage stats", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to get storage stats")
		return
	}

	// Get current retention settings to calculate eligible count
	settings, err := cleanupHandler.GetRetentionSettings(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get retention settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to get retention settings")
		return
	}

	// Calculate eligible for cleanup count
	var eligibleCount int64
	if settings != nil && settings.RetentionDays > 0 {
		eligibleCount, err = cleanupHandler.CountEligibleForCleanup(
			ctx,
			userID,
			settings.RetentionDays,
			settings.ProtectStarred,
			settings.ProtectTagged,
		)
		if err != nil {
			h.logger.Warn("failed to count eligible for cleanup", zap.Error(err))
			// Don't fail the whole request for this
		}
	}

	response := StorageStatsResponse{
		TotalCount:         stats.TotalCount,
		ArchivedCount:      stats.ArchivedCount,
		StarredCount:       stats.StarredCount,
		SnoozedCount:       stats.SnoozedCount,
		UnreadCount:        stats.UnreadCount,
		TaggedCount:        stats.TaggedCount,
		EligibleForCleanup: eligibleCount,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleGetRetentionSettings handles GET /api/user/retention-settings
func (h *Handler) HandleGetRetentionSettings(w http.ResponseWriter, r *http.Request) {
	cleanupHandler := h.getCleanupHandler()
	if cleanupHandler == nil {
		shared.WriteError(w, http.StatusInternalServerError, "Cleanup handler not configured")
		return
	}

	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	settings, err := cleanupHandler.GetRetentionSettings(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get retention settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to get retention settings")
		return
	}

	if settings == nil {
		settings = models.DefaultRetentionSettings()
	}

	response := RetentionSettingsResponse{
		Enabled:        settings.Enabled,
		RetentionDays:  settings.RetentionDays,
		ProtectStarred: settings.ProtectStarred,
		ProtectTagged:  settings.ProtectTagged,
		LastCleanupAt:  settings.LastCleanupAt,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleUpdateRetentionSettings handles PUT /api/user/retention-settings
func (h *Handler) HandleUpdateRetentionSettings(w http.ResponseWriter, r *http.Request) {
	var req RetentionSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode retention settings request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate retention days
	validDays := []int{1, 30, 60, 90, 180, 365}
	isValid := false
	for _, d := range validDays {
		if req.RetentionDays == d {
			isValid = true
			break
		}
	}
	if !isValid {
		shared.WriteError(
			w,
			http.StatusBadRequest,
			"Invalid retention days. Valid values: 1, 30, 60, 90, 180, 365",
		)
		return
	}

	cleanupHandler := h.getCleanupHandler()
	if cleanupHandler == nil {
		shared.WriteError(w, http.StatusInternalServerError, "Cleanup handler not configured")
		return
	}

	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	// Get existing settings to preserve lastCleanupAt
	existingSettings, _ := cleanupHandler.GetRetentionSettings(ctx, userID)
	var lastCleanupAt string
	if existingSettings != nil {
		lastCleanupAt = existingSettings.LastCleanupAt
	}

	settings := &models.RetentionSettings{
		Enabled:        req.Enabled,
		RetentionDays:  req.RetentionDays,
		ProtectStarred: req.ProtectStarred,
		ProtectTagged:  req.ProtectTagged,
		LastCleanupAt:  lastCleanupAt,
	}

	if err := cleanupHandler.UpdateRetentionSettings(ctx, userID, settings); err != nil {
		h.logger.Error("failed to update retention settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to update retention settings")
		return
	}

	response := RetentionSettingsResponse{
		Enabled:        settings.Enabled,
		RetentionDays:  settings.RetentionDays,
		ProtectStarred: settings.ProtectStarred,
		ProtectTagged:  settings.ProtectTagged,
		LastCleanupAt:  settings.LastCleanupAt,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleRunCleanup handles POST /api/user/cleanup
func (h *Handler) HandleRunCleanup(w http.ResponseWriter, r *http.Request) {
	var req CleanupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode cleanup request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RetentionDays <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Retention days must be greater than 0")
		return
	}

	cleanupHandler := h.getCleanupHandler()
	if cleanupHandler == nil {
		shared.WriteError(w, http.StatusInternalServerError, "Cleanup handler not configured")
		return
	}

	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	result, err := cleanupHandler.RunManualCleanup(
		ctx,
		userID,
		req.RetentionDays,
		req.ProtectStarred,
		req.ProtectTagged,
	)
	if err != nil {
		h.logger.Error("failed to run cleanup", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to run cleanup")
		return
	}

	response := CleanupResponse{
		NotificationsDeleted: result.NotificationsDeleted,
		PullRequestsDeleted:  result.PullRequestsDeleted,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleGetEligibleForCleanup handles GET /api/user/cleanup-preview
func (h *Handler) HandleGetEligibleForCleanup(w http.ResponseWriter, r *http.Request) {
	cleanupHandler := h.getCleanupHandler()
	if cleanupHandler == nil {
		shared.WriteError(w, http.StatusInternalServerError, "Cleanup handler not configured")
		return
	}

	ctx := r.Context()
	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	// Get retention settings
	settings, err := cleanupHandler.GetRetentionSettings(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get retention settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to get retention settings")
		return
	}

	if settings == nil || settings.RetentionDays <= 0 {
		shared.WriteJSON(w, http.StatusOK, map[string]int64{"count": 0})
		return
	}

	count, err := cleanupHandler.CountEligibleForCleanup(
		ctx,
		userID,
		settings.RetentionDays,
		settings.ProtectStarred,
		settings.ProtectTagged,
	)
	if err != nil {
		h.logger.Error("failed to count eligible for cleanup", zap.Error(err))
		shared.WriteError(
			w,
			http.StatusInternalServerError,
			"Failed to count eligible notifications",
		)
		return
	}

	shared.WriteJSON(w, http.StatusOK, map[string]int64{"count": count})
}

// getCleanupHandler retrieves the cleanup handler from the scheduler
func (h *Handler) getCleanupHandler() *jobs.CleanupNotificationsHandler {
	if h.scheduler == nil {
		return nil
	}

	// Type assert to SQLiteScheduler to access GetCleanupHandler
	if sqliteScheduler, ok := h.scheduler.(*jobs.SQLiteScheduler); ok {
		return sqliteScheduler.GetCleanupHandler()
	}

	return nil
}
