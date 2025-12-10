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

// Package sync provides the notification sync orchestration service.
package sync

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/core/pullrequest"
	"github.com/octobud-hq/octobud/backend/internal/core/repository"
	"github.com/octobud-hq/octobud/backend/internal/core/syncstate"
	"github.com/octobud-hq/octobud/backend/internal/db"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

// SyncContext contains ALL state needed for a sync operation.
// It's computed once at the start of a sync job via GetSyncContext().
// This eliminates duplicate state fetching and ensures consistency.
type SyncContext struct {
	// UserID is the GitHub user ID for scoping all operations
	UserID string

	// IsSyncConfigured indicates whether sync settings are configured (setup completed)
	// If false, sync should be skipped entirely.
	IsSyncConfigured bool

	// IsInitialSync indicates whether this is the initial sync (before InitialSyncCompletedAt is set)
	// Affects which limits are applied and how state is updated afterward.
	IsInitialSync bool

	// SinceTimestamp is the pre-computed 'since' parameter for the GitHub API.
	// For initial sync: based on InitialSyncDays setting (or nil for all time)
	// For regular sync: based on LatestNotificationAt (or nil if none)
	SinceTimestamp *time.Time

	// Initial sync limits (only applied when IsInitialSync is true)
	MaxCount   *int // Maximum notifications to sync (nil = no limit)
	UnreadOnly bool // Only sync unread notifications

	// OldestNotificationSyncedAt is for tracking purposes (used by job)
	// Existing oldest notification timestamp from previous partial sync (zero if none)
	OldestNotificationSyncedAt time.Time
}

// SyncOperations interface defines operations for syncing notifications
type SyncOperations interface {
	// GetSyncContext gathers ALL necessary state for a sync operation.
	// This should be called once at the start of a sync job.
	// All state fetching happens here - no other method should fetch state.
	GetSyncContext(ctx context.Context, userID string) (SyncContext, error)

	// FetchNotificationsToSync fetches notifications from GitHub using the provided context.
	// This method ONLY fetches and filters - it does NOT fetch or update any state.
	// The syncCtx parameter MUST come from GetSyncContext().
	FetchNotificationsToSync(
		ctx context.Context,
		syncCtx SyncContext,
	) ([]types.NotificationThread, error)

	// FetchOlderNotificationsToSync fetches notifications older than the specified until time.
	// This is used for backfilling older notifications that weren't included in initial sync.
	// - since: fetch notifications updated after this time
	// - until: only include notifications updated before this time (filters out newer ones)
	// - maxCount: optional limit on number of notifications to return
	// - unreadOnly: when true, only fetch unread notifications from GitHub API
	FetchOlderNotificationsToSync(
		ctx context.Context,
		since time.Time,
		until time.Time,
		maxCount *int,
		unreadOnly bool,
	) ([]types.NotificationThread, error)

	// UpdateSyncStateAfterProcessing updates sync state after notifications are processed.
	UpdateSyncStateAfterProcessing(ctx context.Context, userID string, latestUpdate time.Time) error

	// UpdateSyncStateAfterProcessingWithInitialSync updates sync state including initial sync markers.
	UpdateSyncStateAfterProcessingWithInitialSync(
		ctx context.Context,
		userID string,
		latestUpdate time.Time,
		initialSyncCompletedAt *time.Time,
		oldestNotificationSyncedAt *time.Time,
	) error

	// ProcessNotification processes a single notification (upserts repo, fetches subject, etc.)
	ProcessNotification(ctx context.Context, userID string, thread types.NotificationThread) error

	// ProcessNotificationData processes a notification from JSON data
	ProcessNotificationData(ctx context.Context, userID string, data []byte) error

	// RefreshSubjectData fetches fresh subject data from GitHub and updates the notification.
	// Returns (wasMissing, error) where wasMissing indicates if subject data was previously missing.
	RefreshSubjectData(ctx context.Context, userID string, githubID string) (bool, error)

	// IsInitialSyncComplete checks if the initial sync has been completed
	IsInitialSyncComplete(ctx context.Context, userID string) (bool, error)

	// WithClient returns a copy with a different GitHub client (for multi-tenant)
	WithClient(client githubinterfaces.Client) SyncOperations
}

// Service coordinates fetching notifications from GitHub and persisting them.
type Service struct {
	logger              *zap.Logger
	clock               func() time.Time
	client              githubinterfaces.Client
	syncStateService    syncstate.SyncStateService
	repositoryService   repository.RepositoryService
	pullRequestService  pullrequest.PullRequestService
	notificationService notification.NotificationService
	userStore           db.Store // Used only for GetUser (sync settings)
}

// NewService assembles a Service with the provided dependencies.
func NewService(
	logger *zap.Logger,
	clock func() time.Time,
	client githubinterfaces.Client,
	syncStateService syncstate.SyncStateService,
	repositoryService repository.RepositoryService,
	pullRequestService pullrequest.PullRequestService,
	notificationService notification.NotificationService,
	userStore db.Store,
) *Service {
	return &Service{
		logger:              logger,
		clock:               clock,
		client:              client,
		syncStateService:    syncStateService,
		repositoryService:   repositoryService,
		pullRequestService:  pullRequestService,
		notificationService: notificationService,
		userStore:           userStore,
	}
}

// WithClient returns a copy of the service with a different GitHub client.
// This is used for multi-tenant scenarios where each user has their own token.
func (s *Service) WithClient(client githubinterfaces.Client) SyncOperations {
	return &Service{
		logger:              s.logger,
		clock:               s.clock,
		client:              client,
		syncStateService:    s.syncStateService,
		repositoryService:   s.repositoryService,
		pullRequestService:  s.pullRequestService,
		notificationService: s.notificationService,
		userStore:           s.userStore,
	}
}

// GitHubClient returns the GitHub client used by this service.
func (s *Service) GitHubClient() githubinterfaces.Client {
	return s.client
}
