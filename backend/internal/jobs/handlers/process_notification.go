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

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/sync"
)

// ProcessNotificationHandler handles processing a single notification.
type ProcessNotificationHandler struct {
	store       db.Store
	syncService sync.SyncOperations
	logger      *zap.Logger
}

// NewProcessNotificationHandler creates a new ProcessNotificationHandler.
func NewProcessNotificationHandler(
	store db.Store,
	syncService sync.SyncOperations,
	logger *zap.Logger,
) *ProcessNotificationHandler {
	return &ProcessNotificationHandler{
		store:       store,
		syncService: syncService,
		logger:      logger,
	}
}

// Handle processes a single notification.
func (h *ProcessNotificationHandler) Handle(
	ctx context.Context,
	userID string,
	notificationData []byte,
) error {
	var thread types.NotificationThread
	if err := json.Unmarshal(notificationData, &thread); err != nil {
		h.logger.Warn("failed to unmarshal notification data", zap.Error(err))
		return err
	}

	h.logger.Debug("processing notification",
		zap.String("userID", userID),
		zap.String("githubID", thread.ID),
		zap.String("type", thread.Subject.Type),
		zap.String("title", thread.Subject.Title),
		zap.String("reason", thread.Reason))

	// Check if notification already exists (to determine if this is INSERT or UPDATE)
	var isNewNotification bool
	if h.store != nil {
		_, err := h.store.GetNotificationByGithubID(ctx, userID, thread.ID)
		isNewNotification = err != nil // If we got an error, notification doesn't exist yet
	} else {
		// In tests or when store is not available, assume it's a new notification
		isNewNotification = true
	}

	h.logger.Debug("notification existence check",
		zap.String("githubID", thread.ID),
		zap.Bool("isNew", isNewNotification))

	if err := h.syncService.ProcessNotification(ctx, userID, thread); err != nil {
		h.logger.Warn("failed to process notification via sync service",
			zap.String("githubID", thread.ID),
			zap.Error(err))
		return err
	}

	h.logger.Debug("notification synced successfully",
		zap.String("githubID", thread.ID))

	// Only apply rules to newly created notifications (INSERT), not updates
	if isNewNotification && h.store != nil {
		notification, err := h.store.GetNotificationByGithubID(ctx, userID, thread.ID)
		if err == nil {
			matcher := NewRuleMatcher(h.store)
			_, matchErr := matcher.MatchAndApplyRules(ctx, userID, notification.ID)
			if matchErr != nil {
				// Log the error but don't fail the job - rule application is best-effort
				h.logger.Debug("failed to apply rules to notification",
					zap.String("githubID", thread.ID),
					zap.Error(matchErr))
				return nil
			}
			h.logger.Debug("rules applied to new notification",
				zap.String("githubID", thread.ID))
		}
	}

	return nil
}
