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

package handlers

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/update"
)

// CheckUpdatesHandler handles checking for application updates.
type CheckUpdatesHandler struct {
	authSvc       auth.AuthService
	updateService *update.Service
	logger        *zap.Logger
}

// NewCheckUpdatesHandler creates a new CheckUpdatesHandler.
func NewCheckUpdatesHandler(
	authSvc auth.AuthService,
	updateService *update.Service,
	logger *zap.Logger,
) *CheckUpdatesHandler {
	return &CheckUpdatesHandler{
		authSvc:       authSvc,
		updateService: updateService,
		logger:        logger,
	}
}

// Handle checks for updates based on user settings.
// Returns true if an update is available, false otherwise.
// If update checking is disabled or should be skipped, returns false without error.
func (h *CheckUpdatesHandler) Handle(ctx context.Context) (bool, error) {
	// Get user's update settings
	settings, err := h.authSvc.GetUserUpdateSettings(ctx)
	if err != nil {
		h.logger.Warn("failed to get update settings", zap.Error(err))
		return false, nil // Don't fail the job, just skip
	}

	// If updates are disabled, skip
	if !settings.Enabled {
		h.logger.Debug("update checking is disabled")
		return false, nil
	}

	// Check if we should skip based on dismissed until
	if settings.DismissedUntil != "" {
		dismissedUntil, err := time.Parse(time.RFC3339, settings.DismissedUntil)
		if err == nil && time.Now().Before(dismissedUntil) {
			h.logger.Debug("update check dismissed until", zap.String("until", settings.DismissedUntil))
			return false, nil
		}
	}

	// Check if enough time has passed since last check (respect check frequency)
	if settings.LastCheckedAt != "" {
		lastChecked, err := time.Parse(time.RFC3339, settings.LastCheckedAt)
		if err == nil {
			var minInterval time.Duration
			switch settings.CheckFrequency {
			case "daily":
				minInterval = 24 * time.Hour
			case "weekly":
				minInterval = 7 * 24 * time.Hour
			case "on_startup":
				// Only check on startup, not periodically
				return false, nil
			case "never":
				return false, nil
			default:
				minInterval = 24 * time.Hour // Default to daily
			}

			if time.Since(lastChecked) < minInterval {
				h.logger.Debug("update check skipped - too soon since last check",
					zap.String("frequency", settings.CheckFrequency),
					zap.Time("lastChecked", lastChecked),
				)
				return false, nil
			}
		}
	}

	// Check for updates
	updateInfo, err := h.updateService.CheckForUpdates(ctx, settings.IncludePrereleases)
	if err != nil {
		h.logger.Warn("failed to check for updates", zap.Error(err))
		return false, nil // Don't fail the job, just log and skip
	}

	// Update last checked time
	settings.LastCheckedAt = time.Now().Format(time.RFC3339)
	if err := h.authSvc.UpdateUserUpdateSettings(ctx, settings); err != nil {
		h.logger.Warn("failed to update last checked time", zap.Error(err))
	}

	if updateInfo != nil {
		h.logger.Info("update available",
			zap.String("current", updateInfo.CurrentVersion),
			zap.String("latest", updateInfo.LatestVersion),
		)
		return true, nil
	}

	h.logger.Debug("no update available")
	return false, nil
}
