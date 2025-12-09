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

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// CleanupBatchSize is the number of notifications to delete per batch
const CleanupBatchSize = 100

// CleanupNotificationsHandler handles periodic cleanup of old archived notifications
type CleanupNotificationsHandler struct {
	store  db.Store
	logger *zap.Logger
}

// NewCleanupNotificationsHandler creates a new CleanupNotificationsHandler
func NewCleanupNotificationsHandler(
	store db.Store,
	logger *zap.Logger,
) *CleanupNotificationsHandler {
	return &CleanupNotificationsHandler{
		store:  store,
		logger: logger,
	}
}

// CleanupResult contains the results of a cleanup operation
type CleanupResult struct {
	NotificationsDeleted int64
	PullRequestsDeleted  int64
	Skipped              bool   // True if cleanup was skipped (disabled or not configured)
	SkipReason           string // Reason for skipping
}

// Handle performs the cleanup operation based on user's retention settings
func (h *CleanupNotificationsHandler) Handle(
	ctx context.Context,
	userID string,
) (*CleanupResult, error) {
	result := &CleanupResult{}

	// Get user to check retention settings
	user, err := h.store.GetUser(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			result.Skipped = true
			result.SkipReason = "no user configured"
			return result, nil
		}
		return nil, err
	}

	// Parse retention settings
	settings, err := h.getRetentionSettings(user)
	if err != nil {
		h.logger.Warn("failed to parse retention settings", zap.Error(err))
		result.Skipped = true
		result.SkipReason = "invalid retention settings"
		return result, nil
	}

	// Check if cleanup is enabled
	if settings == nil || !settings.Enabled {
		result.Skipped = true
		result.SkipReason = "cleanup disabled"
		h.logger.Debug("cleanup skipped - disabled in settings")
		return result, nil
	}

	// Check if retention days is configured
	if settings.RetentionDays <= 0 {
		result.Skipped = true
		result.SkipReason = "retention days not set"
		h.logger.Debug("cleanup skipped - retention days not set")
		return result, nil
	}

	// Calculate cutoff date
	cutoffDate := time.Now().UTC().AddDate(0, 0, -settings.RetentionDays)
	cutoffStr := cutoffDate.Format(time.RFC3339)

	h.logger.Info("starting notification cleanup",
		zap.String("userID", userID),
		zap.Int("retentionDays", settings.RetentionDays),
		zap.Time("cutoffDate", cutoffDate),
		zap.Bool("protectStarred", settings.ProtectStarred),
		zap.Bool("protectTagged", settings.ProtectTagged))

	// Delete notifications in batches to avoid long locks
	params := db.CleanupParams{
		ProtectStarred: settings.ProtectStarred,
		ProtectTagged:  settings.ProtectTagged,
		CutoffDate:     cutoffStr,
		BatchSize:      CleanupBatchSize,
	}

	for {
		deleted, err := h.store.DeleteOldArchivedNotifications(ctx, userID, params)
		if err != nil {
			h.logger.Error("failed to delete batch of notifications", zap.Error(err))
			return nil, err
		}

		result.NotificationsDeleted += deleted
		h.logger.Debug("deleted batch of notifications", zap.Int64("count", deleted))

		// If we deleted fewer than batch size, we're done
		if deleted < CleanupBatchSize {
			break
		}

		// Small delay between batches to reduce lock contention
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}

	// Clean up orphaned pull requests
	prDeleted, err := h.store.DeleteOrphanedPullRequests(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to delete orphaned pull requests", zap.Error(err))
		// Don't fail the whole operation for this
	} else {
		result.PullRequestsDeleted = prDeleted
	}

	// Update lastCleanupAt timestamp
	settings.LastCleanupAt = time.Now().UTC().Format(time.RFC3339)
	if err := h.updateRetentionSettings(ctx, settings); err != nil {
		h.logger.Warn("failed to update lastCleanupAt", zap.Error(err))
		// Don't fail the operation for this
	}

	h.logger.Info("notification cleanup completed",
		zap.Int64("notificationsDeleted", result.NotificationsDeleted),
		zap.Int64("pullRequestsDeleted", result.PullRequestsDeleted))

	return result, nil
}

// RunManualCleanup runs cleanup with explicit parameters (for manual cleanup via API)
func (h *CleanupNotificationsHandler) RunManualCleanup(
	ctx context.Context,
	userID string,
	retentionDays int,
	protectStarred, protectTagged bool,
) (*CleanupResult, error) {
	result := &CleanupResult{}

	if retentionDays <= 0 {
		result.Skipped = true
		result.SkipReason = "invalid retention days"
		return result, nil
	}

	// Calculate cutoff date
	cutoffDate := time.Now().UTC().AddDate(0, 0, -retentionDays)
	cutoffStr := cutoffDate.Format(time.RFC3339)

	h.logger.Info("starting manual notification cleanup",
		zap.String("userID", userID),
		zap.Int("retentionDays", retentionDays),
		zap.Time("cutoffDate", cutoffDate),
		zap.Bool("protectStarred", protectStarred),
		zap.Bool("protectTagged", protectTagged))

	// Delete notifications in batches
	params := db.CleanupParams{
		ProtectStarred: protectStarred,
		ProtectTagged:  protectTagged,
		CutoffDate:     cutoffStr,
		BatchSize:      CleanupBatchSize,
	}

	for {
		deleted, err := h.store.DeleteOldArchivedNotifications(ctx, userID, params)
		if err != nil {
			return nil, err
		}

		result.NotificationsDeleted += deleted

		if deleted < CleanupBatchSize {
			break
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}

	// Clean up orphaned pull requests
	prDeleted, err := h.store.DeleteOrphanedPullRequests(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to delete orphaned pull requests", zap.Error(err))
	} else {
		result.PullRequestsDeleted = prDeleted
	}

	h.logger.Info("manual cleanup completed",
		zap.Int64("notificationsDeleted", result.NotificationsDeleted),
		zap.Int64("pullRequestsDeleted", result.PullRequestsDeleted))

	return result, nil
}

// getRetentionSettings parses retention settings from user
func (h *CleanupNotificationsHandler) getRetentionSettings(
	user db.User,
) (*models.RetentionSettings, error) {
	if !user.RetentionSettings.Valid || len(user.RetentionSettings.RawMessage) == 0 {
		return nil, nil
	}
	return models.RetentionSettingsFromJSON(user.RetentionSettings.RawMessage)
}

// updateRetentionSettings saves updated retention settings
func (h *CleanupNotificationsHandler) updateRetentionSettings(
	ctx context.Context,
	settings *models.RetentionSettings,
) error {
	data, err := settings.ToJSON()
	if err != nil {
		return err
	}

	_, err = h.store.UpdateUserRetentionSettings(ctx, sql.NullString{
		String: string(data),
		Valid:  true,
	})
	return err
}

// GetStorageStats returns current storage statistics
func (h *CleanupNotificationsHandler) GetStorageStats(
	ctx context.Context,
	userID string,
) (db.StorageStats, error) {
	return h.store.GetStorageStats(ctx, userID)
}

// CountEligibleForCleanup returns the number of notifications that would be deleted
func (h *CleanupNotificationsHandler) CountEligibleForCleanup(
	ctx context.Context,
	userID string,
	retentionDays int,
	protectStarred, protectTagged bool,
) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	cutoffDate := time.Now().UTC().AddDate(0, 0, -retentionDays)
	cutoffStr := cutoffDate.Format(time.RFC3339)

	return h.store.CountEligibleForCleanup(ctx, userID, db.CleanupParams{
		ProtectStarred: protectStarred,
		ProtectTagged:  protectTagged,
		CutoffDate:     cutoffStr,
	})
}

// GetRetentionSettings returns current retention settings for the user
func (h *CleanupNotificationsHandler) GetRetentionSettings(
	ctx context.Context,
	userID string,
) (*models.RetentionSettings, error) {
	// Note: userID is currently unused as we have single-user mode
	// It will be used when we have multi-user support
	_ = userID
	user, err := h.store.GetUser(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.DefaultRetentionSettings(), nil
		}
		return nil, err
	}

	settings, err := h.getRetentionSettings(user)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return models.DefaultRetentionSettings(), nil
	}

	return settings, nil
}

// UpdateRetentionSettings updates the user's retention settings
func (h *CleanupNotificationsHandler) UpdateRetentionSettings(
	ctx context.Context,
	userID string,
	settings *models.RetentionSettings,
) error {
	// Note: userID is currently unused as we have single-user mode
	// It will be used when we have multi-user support
	_ = userID
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, err = h.store.UpdateUserRetentionSettings(ctx, sql.NullString{
		String: string(data),
		Valid:  true,
	})
	return err
}
