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

package jobs

import (
	"context"

	"github.com/octobud-hq/octobud/backend/internal/jobs/handlers"
)

// CleanupNotificationsHandler is re-exported from handlers for API access
type CleanupNotificationsHandler = handlers.CleanupNotificationsHandler

// Scheduler defines the interface for a background job scheduler
type Scheduler interface {
	// Start begins the scheduler's background processing
	Start(ctx context.Context) error

	// Stop gracefully shuts down the scheduler
	Stop(ctx context.Context) error

	// EnqueueSyncNotifications enqueues a job to sync notifications from GitHub
	EnqueueSyncNotifications(ctx context.Context, userID string) error

	// EnqueueProcessNotification enqueues a job to process a single notification
	EnqueueProcessNotification(ctx context.Context, userID string, notificationData []byte) error

	// EnqueueApplyRule enqueues a job to apply a rule to notifications
	EnqueueApplyRule(ctx context.Context, userID string, ruleID string) error

	// EnqueueSyncOlder enqueues a job to sync older notifications
	EnqueueSyncOlder(ctx context.Context, args SyncOlderNotificationsArgs) error
}

// JobHandler defines the interface for handling different job types
type JobHandler interface {
	// HandleSyncNotifications handles the sync notifications job
	HandleSyncNotifications(ctx context.Context, userID string) error

	// HandleProcessNotification handles processing a single notification
	HandleProcessNotification(ctx context.Context, userID string, notificationData []byte) error

	// HandleApplyRule handles applying a rule to notifications
	HandleApplyRule(ctx context.Context, userID string, ruleID string) error
}
