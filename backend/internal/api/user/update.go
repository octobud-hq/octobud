// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
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
)

// HandleGetUpdateSettings handles GET /api/user/update-settings
func (h *Handler) HandleGetUpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, err := h.authSvc.GetUserUpdateSettings(ctx)
	if err != nil {
		h.logger.Error("failed to get update settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := UpdateSettingsResponse{
		Enabled:            settings.Enabled,
		CheckFrequency:     settings.CheckFrequency,
		IncludePrereleases: settings.IncludePrereleases,
		LastCheckedAt:      settings.LastCheckedAt,
		DismissedUntil:     settings.DismissedUntil,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleUpdateUpdateSettings handles PUT /api/user/update-settings
func (h *Handler) HandleUpdateUpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current settings
	currentSettings, err := h.authSvc.GetUserUpdateSettings(ctx)
	if err != nil {
		h.logger.Error("failed to get current update settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode update settings request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update only provided fields
	if req.Enabled != nil {
		currentSettings.Enabled = *req.Enabled
	}
	if req.CheckFrequency != nil {
		validFrequencies := map[string]bool{
			"on_startup": true,
			"daily":      true,
			"weekly":     true,
			"never":      true,
		}
		if !validFrequencies[*req.CheckFrequency] {
			shared.WriteError(w, http.StatusBadRequest, "checkFrequency must be one of: on_startup, daily, weekly, never")
			return
		}
		currentSettings.CheckFrequency = *req.CheckFrequency
	}
	if req.IncludePrereleases != nil {
		currentSettings.IncludePrereleases = *req.IncludePrereleases
	}
	if req.DismissedUntil != nil {
		currentSettings.DismissedUntil = *req.DismissedUntil
	}

	// Save updated settings
	if err := h.authSvc.UpdateUserUpdateSettings(ctx, currentSettings); err != nil {
		h.logger.Error("failed to update update settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to update settings")
		return
	}

	response := UpdateSettingsResponse{
		Enabled:            currentSettings.Enabled,
		CheckFrequency:     currentSettings.CheckFrequency,
		IncludePrereleases: currentSettings.IncludePrereleases,
		LastCheckedAt:      currentSettings.LastCheckedAt,
		DismissedUntil:     currentSettings.DismissedUntil,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// HandleCheckForUpdates handles GET /api/user/update-check
func (h *Handler) HandleCheckForUpdates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user's update settings
	settings, err := h.authSvc.GetUserUpdateSettings(ctx)
	if err != nil {
		h.logger.Error("failed to get update settings", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// If updates are disabled, return no update available
	if !settings.Enabled {
		shared.WriteJSON(w, http.StatusOK, UpdateCheckResponse{
			UpdateAvailable: false,
		})
		return
	}

	// Check if we should skip based on dismissed until
	if settings.DismissedUntil != "" {
		dismissedUntil, err := time.Parse(time.RFC3339, settings.DismissedUntil)
		if err == nil && time.Now().Before(dismissedUntil) {
			shared.WriteJSON(w, http.StatusOK, UpdateCheckResponse{
				UpdateAvailable: false,
			})
			return
		}
	}

	// Check for updates using the update service
	if h.updateService == nil {
		h.logger.Error("update service not configured")
		shared.WriteError(w, http.StatusServiceUnavailable, "Update service not available")
		return
	}

	updateInfo, err := h.updateService.CheckForUpdates(ctx, settings.IncludePrereleases)
	if err != nil {
		h.logger.Error("failed to check for updates", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to check for updates")
		return
	}

	if updateInfo == nil {
		// Update last checked time
		settings.LastCheckedAt = time.Now().Format(time.RFC3339)
		if err := h.authSvc.UpdateUserUpdateSettings(ctx, settings); err != nil {
			h.logger.Warn("failed to update last checked time", zap.Error(err))
		}

		shared.WriteJSON(w, http.StatusOK, UpdateCheckResponse{
			UpdateAvailable: false,
		})
		return
	}

	// Update last checked time
	settings.LastCheckedAt = time.Now().Format(time.RFC3339)
	if err := h.authSvc.UpdateUserUpdateSettings(ctx, settings); err != nil {
		h.logger.Warn("failed to update last checked time", zap.Error(err))
	}

	response := UpdateCheckResponse{
		UpdateAvailable: true,
		CurrentVersion:  updateInfo.CurrentVersion,
		LatestVersion:   updateInfo.LatestVersion,
		ReleaseName:     updateInfo.ReleaseName,
		ReleaseNotes:    updateInfo.ReleaseNotes,
		PublishedAt:     updateInfo.PublishedAt,
		DownloadURL:     updateInfo.DownloadURL,
		AssetName:       updateInfo.AssetName,
		AssetSize:       updateInfo.AssetSize,
		IsPrerelease:    updateInfo.IsPrerelease,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}
