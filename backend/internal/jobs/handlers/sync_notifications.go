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
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/sync"
)

// NotificationEnqueuer is an interface for enqueueing notification processing jobs.
type NotificationEnqueuer interface {
	EnqueueProcessNotification(ctx context.Context, userID string, notificationData []byte) error
}

// SyncNotificationsHandler handles syncing notifications from GitHub.
type SyncNotificationsHandler struct {
	syncService sync.SyncOperations
	enqueuer    NotificationEnqueuer
	logger      *zap.Logger
}

// NewSyncNotificationsHandler creates a new SyncNotificationsHandler.
func NewSyncNotificationsHandler(
	syncService sync.SyncOperations,
	enqueuer NotificationEnqueuer,
	logger *zap.Logger,
) *SyncNotificationsHandler {
	return &SyncNotificationsHandler{
		syncService: syncService,
		enqueuer:    enqueuer,
		logger:      logger,
	}
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	UserID             string
	Threads            []types.NotificationThread
	LatestUpdate       time.Time
	OldestNotification time.Time
	IsInitialSync      bool
}

// Handle syncs notifications from GitHub.
// The userID parameter is used to enqueue child jobs with the correct user context.
// Returns the sync result so callers can handle state updates appropriately.
func (h *SyncNotificationsHandler) Handle(ctx context.Context, userID string) (*SyncResult, error) {
	// Get all sync context at the start - single source of truth
	syncCtx, err := h.syncService.GetSyncContext(ctx, userID)
	if err != nil {
		h.logger.Warn("failed to get sync context", zap.Error(err))
		return nil, nil
	}

	// If sync isn't configured yet, skip entirely
	if !syncCtx.IsSyncConfigured {
		h.logger.Debug("sync not configured yet, skipping")
		return nil, nil
	}

	// Log sync status
	if syncCtx.IsInitialSync {
		if syncCtx.OldestNotificationSyncedAt.IsZero() {
			h.logger.Info("initial sync starting")
		} else {
			h.logger.Info("initial sync in progress",
				zap.Time("oldestSoFar", syncCtx.OldestNotificationSyncedAt))
		}
	}

	// Fetch notifications from GitHub
	threads, err := h.syncService.FetchNotificationsToSync(ctx, syncCtx)
	if err != nil {
		return nil, err
	}

	// If this was an initial sync and we found zero notifications, mark as complete
	if syncCtx.IsInitialSync && len(threads) == 0 {
		h.logger.Info("initial sync found 0 notifications, marking as complete")
		now := time.Now().UTC()
		var zeroTime time.Time
		if err := h.syncService.UpdateSyncStateAfterProcessingWithInitialSync(
			ctx, userID, zeroTime, &now, nil,
		); err != nil {
			h.logger.Warn(
				"failed to mark initial sync complete with 0 notifications",
				zap.Error(err),
			)
		}
		return nil, nil
	}

	if len(threads) == 0 {
		h.logger.Debug("no new notifications found")
		return nil, nil
	}

	h.logger.Info("processing notifications", zap.Int("count", len(threads)))

	result := &SyncResult{
		UserID:        userID,
		Threads:       threads,
		IsInitialSync: syncCtx.IsInitialSync,
	}

	// First, marshal all notifications - if any fail, don't proceed
	type preparedJob struct {
		thread types.NotificationThread
		data   []byte
	}
	prepared := make([]preparedJob, 0, len(threads))

	for _, thread := range threads {
		threadData, err := json.Marshal(thread)
		if err != nil {
			h.logger.Error("failed to marshal notification thread - aborting batch",
				zap.String("threadID", thread.ID),
				zap.Error(err))
			return nil, err
		}
		prepared = append(prepared, preparedJob{thread: thread, data: threadData})
	}

	// Now enqueue all notifications - we need ALL to succeed to update sync state
	var enqueuedCount int
	for i, job := range prepared {
		h.logger.Debug("enqueueing notification",
			zap.Int("index", i),
			zap.Int("total", len(prepared)),
			zap.String("threadID", job.thread.ID),
			zap.String("type", job.thread.Subject.Type),
			zap.String("title", job.thread.Subject.Title))

		if err := h.enqueuer.EnqueueProcessNotification(ctx, userID, job.data); err != nil {
			// If any enqueue fails, don't update sync state - we'll retry next sync
			// Processing is idempotent, so re-enqueueing the same notification is safe
			h.logger.Error("failed to enqueue notification - aborting batch to prevent data loss",
				zap.String("threadID", job.thread.ID),
				zap.Int("enqueuedSoFar", enqueuedCount),
				zap.Int("remaining", len(prepared)-i),
				zap.Error(err))
			return nil, err
		}

		enqueuedCount++

		if job.thread.UpdatedAt.After(result.LatestUpdate) {
			result.LatestUpdate = job.thread.UpdatedAt
		}

		// Track oldest notification for initial sync
		if syncCtx.IsInitialSync {
			if result.OldestNotification.IsZero() ||
				job.thread.UpdatedAt.Before(result.OldestNotification) {
				result.OldestNotification = job.thread.UpdatedAt
			}
		}
	}

	h.logger.Info("finished enqueueing notifications",
		zap.Int("enqueued", enqueuedCount),
		zap.Int("total", len(threads)))

	return result, nil
}

// UpdateSyncState updates the sync state after processing.
// This is separated so callers can decide how/when to update state.
func (h *SyncNotificationsHandler) UpdateSyncState(ctx context.Context, result *SyncResult) error {
	if result == nil || result.LatestUpdate.IsZero() {
		return nil
	}

	if result.IsInitialSync {
		now := time.Now().UTC()
		return h.syncService.UpdateSyncStateAfterProcessingWithInitialSync(
			ctx,
			result.UserID,
			result.LatestUpdate,
			&now,
			&result.OldestNotification,
		)
	}

	return h.syncService.UpdateSyncStateAfterProcessing(ctx, result.UserID, result.LatestUpdate)
}
