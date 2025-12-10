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
)

// ApplyRulesToNotificationHandler handles applying all enabled rules to a single notification.
type ApplyRulesToNotificationHandler struct {
	store  db.Store
	logger *zap.Logger
}

// NewApplyRulesToNotificationHandler creates a new ApplyRulesToNotificationHandler.
func NewApplyRulesToNotificationHandler(
	store db.Store,
	logger *zap.Logger,
) *ApplyRulesToNotificationHandler {
	return &ApplyRulesToNotificationHandler{
		store:  store,
		logger: logger,
	}
}

// ApplyRulesToNotificationJobPayload represents the payload for the job.
type ApplyRulesToNotificationJobPayload struct {
	UserID   string `json:"userID"`
	GithubID string `json:"githubID"`
}

// Handle processes a single notification to apply all enabled rules.
func (h *ApplyRulesToNotificationHandler) Handle(
	ctx context.Context,
	userID string,
	payload []byte,
) error {
	var jobPayload ApplyRulesToNotificationJobPayload
	if err := json.Unmarshal(payload, &jobPayload); err != nil {
		h.logger.Warn("failed to unmarshal apply rules to notification job payload", zap.Error(err))
		return err
	}

	// Use userID from payload if provided, otherwise use the one passed in
	actualUserID := jobPayload.UserID
	if actualUserID == "" {
		actualUserID = userID
	}

	h.logger.Debug("applying rules to notification",
		zap.String("userID", actualUserID),
		zap.String("githubID", jobPayload.GithubID))

	// Get notification by githubID to get notificationID
	notification, err := h.store.GetNotificationByGithubID(ctx, actualUserID, jobPayload.GithubID)
	if err != nil {
		h.logger.Warn("failed to get notification for rule application",
			zap.String("githubID", jobPayload.GithubID),
			zap.Error(err))
		return err
	}

	// Apply all enabled rules to this notification
	matcher := NewRuleMatcher(h.store)
	_, matchErr := matcher.MatchAndApplyRules(ctx, actualUserID, notification.ID)
	if matchErr != nil {
		// Log the error but don't fail the job - rule application is best-effort
		h.logger.Debug("failed to apply rules to notification",
			zap.String("githubID", jobPayload.GithubID),
			zap.Error(matchErr))
		return nil
	}

	h.logger.Debug("rules applied to notification",
		zap.String("githubID", jobPayload.GithubID))

	return nil
}
