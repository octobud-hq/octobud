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

	"github.com/octobud-hq/octobud/backend/internal/sync"
)

// SyncOlderArgs contains arguments for syncing older notifications.
type SyncOlderArgs struct {
	// UserID is the GitHub user ID for scoping
	UserID string
	// Days is the number of days to sync back from UntilTime
	Days int
	// UntilTime is the cutoff - only sync notifications older than this
	UntilTime time.Time
	// MaxCount is an optional limit on the number of notifications to sync
	MaxCount *int
	// UnreadOnly filters to only sync unread notifications
	UnreadOnly bool
}

// SyncOlderHandler handles syncing older notifications from GitHub.
type SyncOlderHandler struct {
	syncService sync.SyncOperations
	enqueuer    NotificationEnqueuer
	logger      *zap.Logger
}

// NewSyncOlderHandler creates a new SyncOlderHandler.
func NewSyncOlderHandler(
	syncService sync.SyncOperations,
	enqueuer NotificationEnqueuer,
	logger *zap.Logger,
) *SyncOlderHandler {
	return &SyncOlderHandler{
		syncService: syncService,
		enqueuer:    enqueuer,
		logger:      logger,
	}
}

// Handle syncs older notifications from GitHub.
func (h *SyncOlderHandler) Handle(ctx context.Context, args SyncOlderArgs) error {
	// Compute the time range
	since := args.UntilTime.AddDate(0, 0, -args.Days)

	h.logger.Info("syncing older notifications",
		zap.String("userID", args.UserID),
		zap.Time("since", since),
		zap.Time("until", args.UntilTime),
		zap.Int("days", args.Days),
		zap.Any("maxCount", args.MaxCount),
		zap.Bool("unreadOnly", args.UnreadOnly))

	// Fetch older notifications using the sync service
	threads, err := h.syncService.FetchOlderNotificationsToSync(
		ctx,
		since,
		args.UntilTime,
		args.MaxCount,
		args.UnreadOnly,
	)
	if err != nil {
		h.logger.Error("failed to fetch older notifications", zap.Error(err))
		return err
	}

	if len(threads) == 0 {
		h.logger.Info("no older notifications found in time range")
		return nil
	}

	h.logger.Info("processing older notifications", zap.Int("count", len(threads)))

	// Track the oldest notification for updating sync state
	var oldestNotification time.Time

	// Enqueue individual processing jobs for each notification
	for _, thread := range threads {
		threadData, err := json.Marshal(thread)
		if err != nil {
			h.logger.Warn("failed to marshal notification thread",
				zap.String("threadID", thread.ID),
				zap.Error(err))
			continue
		}

		if err := h.enqueuer.EnqueueProcessNotification(ctx, args.UserID, threadData); err != nil {
			h.logger.Warn("failed to enqueue notification processing",
				zap.String("threadID", thread.ID),
				zap.Error(err))
			continue
		}

		// Track oldest notification
		if oldestNotification.IsZero() || thread.UpdatedAt.Before(oldestNotification) {
			oldestNotification = thread.UpdatedAt
		}
	}

	// Update oldest_notification_synced_at if we found older notifications
	if !oldestNotification.IsZero() {
		h.logger.Info("updating oldest notification synced timestamp",
			zap.Time("oldestNotification", oldestNotification))

		if err := h.syncService.UpdateSyncStateAfterProcessingWithInitialSync(
			ctx,
			args.UserID,
			time.Time{}, // Don't update latest_notification_at
			nil,         // Don't change initial_sync_completed_at
			&oldestNotification,
		); err != nil {
			h.logger.Warn("failed to update oldest notification timestamp", zap.Error(err))
		}
	}

	h.logger.Info("completed sync older notifications",
		zap.Int("processed", len(threads)),
		zap.Time("oldestNotification", oldestNotification))

	return nil
}
