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

package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/core/pullrequest"
	"github.com/octobud-hq/octobud/backend/internal/core/repository"
	"github.com/octobud-hq/octobud/backend/internal/core/syncstate"
	"github.com/octobud-hq/octobud/backend/internal/db"
	_ "github.com/octobud-hq/octobud/backend/internal/db/sqlite" // Register SQLite store factory
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

// mockClock returns a fixed time for testing
func mockClock() time.Time {
	return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
}

// setupSyncService creates a Service with mocked dependencies for testing
func setupSyncService(
	_ *testing.T,
	dbConn *sql.DB,
	mockClient githubinterfaces.Client,
) *Service {
	queries := db.NewStore(dbConn)
	syncStateService := syncstate.NewSyncStateService(queries)
	repositoryService := repository.NewService(queries)
	pullRequestService := pullrequest.NewService(queries)
	notificationService := notification.NewService(queries)

	return NewService(
		zap.NewNop(),
		mockClock,
		mockClient,
		syncStateService,
		repositoryService,
		pullRequestService,
		notificationService,
		queries, // userStore for sync settings
	)
}

// TestFetchNotificationsToSync_InitialSync tests fetching notifications when no sync state exists
func TestFetchNotificationsToSync_InitialSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo",
			},
			Subject: types.NotificationSubject{
				Title: "Test PR",
				Type:  "PullRequest",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(notifications, nil)

	// FetchNotificationsToSync is a pure function - no DB calls expected
	service := setupSyncService(t, dbConn, mockClient)

	// Pass sync context indicating this is an initial sync
	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    true,
		SinceTimestamp:   nil, // All time
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.NoError(t, err)
	require.Len(t, threads, 1)
	require.Equal(t, "notif-1", threads[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFetchNotificationsToSync_WithExistingState tests fetching with existing sync state
// (initial sync already completed, using pre-computed SinceTimestamp)
func TestFetchNotificationsToSync_WithExistingState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	latestNotification := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)
	notifications := []types.NotificationThread{
		{
			ID: "notif-2",
			Repository: types.RepositorySnapshot{
				ID:       456,
				FullName: "test/repo2",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	var capturedSince *time.Time
	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, since *time.Time, _ *time.Time, _ bool) ([]types.NotificationThread, error) {
			capturedSince = since
			return notifications, nil
		})

	// FetchNotificationsToSync is a pure function - no DB calls expected
	service := setupSyncService(t, dbConn, mockClient)

	// Pass sync context with pre-computed SinceTimestamp (as would come from GetSyncContext)
	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    false, // Initial sync already completed
		SinceTimestamp:   &latestNotification,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.NoError(t, err)
	require.Len(t, threads, 1)
	require.Equal(t, "notif-2", threads[0].ID)
	require.NotNil(t, capturedSince)
	require.Equal(t, latestNotification, *capturedSince)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFetchNotificationsToSync_EmptyResults tests when GitHub returns no new notifications
// FetchNotificationsToSync is now a pure function - it doesn't update sync state.
func TestFetchNotificationsToSync_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]types.NotificationThread{}, nil)

	// FetchNotificationsToSync is a pure function - no DB calls expected
	service := setupSyncService(t, dbConn, mockClient)

	// Non-initial sync with empty results - no state updates expected
	// (state management is now the job's responsibility)
	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    false,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.NoError(t, err)
	require.Len(t, threads, 0)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestFetchNotificationsToSync_ClientError tests handling of GitHub client errors
func TestFetchNotificationsToSync_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("API error"))

	// FetchNotificationsToSync is a pure function - no DB calls expected
	service := setupSyncService(t, dbConn, mockClient)

	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    true,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch notifications")
	require.Nil(t, threads)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSyncStateAfterProcessing_Success tests successful sync state update
func TestUpdateSyncStateAfterProcessing_Success(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	latestUpdate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// UpsertSyncState expects 6 args: userID + 5 params
	mock.ExpectQuery(`INSERT INTO sync_state`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "last_successful_poll", "last_notification_etag", "created_at", "updated_at",
			"latest_notification_at", "initial_sync_completed_at", "oldest_notification_synced_at",
		}).AddRow(1, "test-user-id", time.Now(), sql.NullString{}, time.Now(), time.Now(),
			latestUpdate, sql.NullTime{}, sql.NullTime{}))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	service := setupSyncService(t, dbConn, mockClient)

	err = service.UpdateSyncStateAfterProcessing(context.Background(), "test-user-id", latestUpdate)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSyncStateAfterProcessing_ZeroTime tests update with zero time
func TestUpdateSyncStateAfterProcessing_ZeroTime(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	// UpsertSyncState expects 6 args: userID + 5 params
	mock.ExpectQuery(`INSERT INTO sync_state`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "last_successful_poll", "last_notification_etag", "created_at", "updated_at",
			"latest_notification_at", "initial_sync_completed_at", "oldest_notification_synced_at",
		}).AddRow(1, "test-user-id", time.Now(), sql.NullString{}, time.Now(), time.Now(),
			sql.NullTime{}, sql.NullTime{}, sql.NullTime{}))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	service := setupSyncService(t, dbConn, mockClient)

	err = service.UpdateSyncStateAfterProcessing(context.Background(), "test-user-id", time.Time{})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessNotification_RepositoryError tests repository upsert failure
// Note: Success cases for ProcessNotification are tested via handler-level tests
// in backend/internal/jobs/handlers/process_notification_test.go which test
// the full integration through MockSyncOperations.
func TestProcessNotification_RepositoryError(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Test",
			Type:  "Issue",
		},
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)

	// Repository upsert fails
	mock.ExpectQuery(`INSERT INTO repositories`).
		WillReturnError(errors.New("database error"))

	service := setupSyncService(t, dbConn, mockClient)

	err = service.ProcessNotification(context.Background(), "test-user-id", thread)

	require.Error(t, err)
	require.Contains(t, err.Error(), "upsert repository")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestRefreshSubjectData_NotificationNotFound tests error when notification doesn't exist
// Note: Success cases for RefreshSubjectData are tested via the API handler
// which calls the sync service through the HTTP endpoint.
func TestRefreshSubjectData_NotificationNotFound(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer dbConn.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)

	mock.ExpectQuery(`SELECT (.+) FROM notifications WHERE user_id = (.+) AND github_id = (.+)`).
		WithArgs("test-user-id", "nonexistent").
		WillReturnError(sql.ErrNoRows)

	service := setupSyncService(t, dbConn, mockClient)

	err = service.RefreshSubjectData(context.Background(), "test-user-id", "nonexistent")

	require.Error(t, err)
	require.Contains(t, err.Error(), "get notification")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ======================================
// Tests for sync_settings.go helpers
// ======================================

// intPtr is a helper function to create a pointer to an int
func intPtr(i int) *int {
	return &i
}

// TestCalculateSyncSinceDate tests the calculateSyncSinceDate helper
func TestCalculateSyncSinceDate(t *testing.T) {
	// Get current time for comparison
	now := time.Now().UTC()

	tests := []struct {
		name         string
		days         int
		expectedDiff time.Duration
	}{
		{
			name:         "1 day ago",
			days:         1,
			expectedDiff: 24 * time.Hour,
		},
		{
			name:         "7 days ago",
			days:         7,
			expectedDiff: 7 * 24 * time.Hour,
		},
		{
			name:         "30 days ago",
			days:         30,
			expectedDiff: 30 * 24 * time.Hour,
		},
		{
			name:         "365 days ago",
			days:         365,
			expectedDiff: 365 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSyncSinceDate(tt.days)

			// Allow 1 second tolerance for test execution time
			diff := now.Sub(result)
			require.InDelta(t, tt.expectedDiff.Seconds(), diff.Seconds(), 1.0,
				"expected %v, got diff of %v", tt.expectedDiff, diff)

			// Verify result is in UTC
			require.Equal(t, time.UTC, result.Location())
		})
	}
}

// TestApplyInitialSyncLimitsFromContext tests the applyInitialSyncLimitsFromContext helper
func TestApplyInitialSyncLimitsFromContext(t *testing.T) {
	// Create test notification threads
	createThreads := func(count int, allUnread bool) []types.NotificationThread {
		threads := make([]types.NotificationThread, count)
		for i := 0; i < count; i++ {
			threads[i] = types.NotificationThread{
				ID:     string(rune('A' + i)),
				Unread: allUnread || i%2 == 0, // Alternate unread if allUnread=false
			}
		}
		return threads
	}

	tests := []struct {
		name          string
		threads       []types.NotificationThread
		syncCtx       SyncContext
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:    "no limits returns all threads",
			threads: createThreads(5, true),
			syncCtx: SyncContext{
				MaxCount:   nil,
				UnreadOnly: false,
			},
			expectedCount: 5,
			expectedIDs:   []string{"A", "B", "C", "D", "E"},
		},
		{
			name:    "maxCount limits results",
			threads: createThreads(10, true),
			syncCtx: SyncContext{
				MaxCount:   intPtr(3),
				UnreadOnly: false,
			},
			expectedCount: 3,
			expectedIDs:   []string{"A", "B", "C"},
		},
		{
			name: "unreadOnly filters read notifications",
			threads: []types.NotificationThread{
				{ID: "A", Unread: true},
				{ID: "B", Unread: false},
				{ID: "C", Unread: true},
				{ID: "D", Unread: false},
				{ID: "E", Unread: true},
			},
			syncCtx: SyncContext{
				MaxCount:   nil,
				UnreadOnly: true,
			},
			expectedCount: 3,
			expectedIDs:   []string{"A", "C", "E"},
		},
		{
			name: "maxCount and unreadOnly combined",
			threads: []types.NotificationThread{
				{ID: "A", Unread: true},
				{ID: "B", Unread: false},
				{ID: "C", Unread: true},
				{ID: "D", Unread: true},
				{ID: "E", Unread: false},
				{ID: "F", Unread: true},
			},
			syncCtx: SyncContext{
				MaxCount:   intPtr(2),
				UnreadOnly: true,
			},
			expectedCount: 2,
			expectedIDs:   []string{"A", "C"},
		},
		{
			name:    "maxCount larger than thread count",
			threads: createThreads(3, true),
			syncCtx: SyncContext{
				MaxCount:   intPtr(100),
				UnreadOnly: false,
			},
			expectedCount: 3,
			expectedIDs:   []string{"A", "B", "C"},
		},
		{
			name:    "maxCount of 1",
			threads: createThreads(5, true),
			syncCtx: SyncContext{
				MaxCount:   intPtr(1),
				UnreadOnly: false,
			},
			expectedCount: 1,
			expectedIDs:   []string{"A"},
		},
		{
			name:    "empty threads returns empty",
			threads: []types.NotificationThread{},
			syncCtx: SyncContext{
				MaxCount:   intPtr(10),
				UnreadOnly: true,
			},
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name: "unreadOnly with all read returns empty",
			threads: []types.NotificationThread{
				{ID: "A", Unread: false},
				{ID: "B", Unread: false},
				{ID: "C", Unread: false},
			},
			syncCtx: SyncContext{
				MaxCount:   nil,
				UnreadOnly: true,
			},
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name: "maxCount of 0 returns empty",
			threads: []types.NotificationThread{
				{ID: "A", Unread: true},
				{ID: "B", Unread: true},
			},
			syncCtx: SyncContext{
				MaxCount:   intPtr(0),
				UnreadOnly: false,
			},
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyInitialSyncLimitsFromContext(tt.threads, tt.syncCtx)

			require.Len(t, result, tt.expectedCount)

			actualIDs := make([]string, len(result))
			for i, thread := range result {
				actualIDs[i] = thread.ID
			}
			require.Equal(t, tt.expectedIDs, actualIDs)
		})
	}
}

// TestGetSyncContext tests the GetSyncContext method
func TestGetSyncContext(t *testing.T) {
	tests := []struct {
		name            string
		setupMock       func(sqlmock.Sqlmock)
		expectedContext SyncContext
		expectError     bool
	}{
		{
			name: "initial sync with days configured",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GetUser returns user with sync settings (30 days, 100 max count, unread only)
				syncSettingsJSON := `{"initialSyncDays": 30, "initialSyncMaxCount": 100, "initialSyncUnreadOnly": true, "setupCompleted": true}`
				userRows := sqlmock.NewRows([]string{
					"id", "github_user_id", "github_username", "github_token_encrypted", "created_at", "updated_at", "sync_settings", "retention_settings", "muted_until",
				}).AddRow(1, sql.NullString{}, sql.NullString{}, sql.NullString{}, time.Now(), time.Now(), []byte(syncSettingsJSON), sql.NullString{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM users`).WillReturnRows(userRows)

				// GetSyncState returns empty state (no initial sync completed)
				syncStateRows := sqlmock.NewRows([]string{
					"id", "user_id", "last_successful_poll", "last_notification_etag", "created_at", "updated_at",
					"latest_notification_at", "initial_sync_completed_at", "oldest_notification_synced_at",
				}).AddRow(1, "test-user-id", time.Now(), sql.NullString{}, time.Now(), time.Now(), sql.NullTime{}, sql.NullTime{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).WillReturnRows(syncStateRows)
			},
			expectedContext: SyncContext{
				IsSyncConfigured: true,
				IsInitialSync:    true,
				MaxCount:         intPtr(100),
				UnreadOnly:       true,
				// SinceTimestamp will be set but we can't predict exact value (uses time.Now())
			},
			expectError: false,
		},
		{
			name: "initial sync with all-time (nil days)",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GetUser returns user with sync settings (no days limit)
				syncSettingsJSON := `{"initialSyncMaxCount": 500, "setupCompleted": true}`
				userRows := sqlmock.NewRows([]string{
					"id", "github_user_id", "github_username", "github_token_encrypted", "created_at", "updated_at", "sync_settings", "retention_settings", "muted_until",
				}).AddRow(1, sql.NullString{}, sql.NullString{}, sql.NullString{}, time.Now(), time.Now(), []byte(syncSettingsJSON), sql.NullString{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM users`).WillReturnRows(userRows)

				// GetSyncState returns empty state (no initial sync completed)
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).WillReturnError(sql.ErrNoRows)
			},
			expectedContext: SyncContext{
				IsSyncConfigured: true,
				IsInitialSync:    true,
				SinceTimestamp:   nil, // All time
				MaxCount:         intPtr(500),
				UnreadOnly:       false,
			},
			expectError: false,
		},
		{
			name: "regular sync (initial sync already completed)",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GetUser returns user with sync settings
				syncSettingsJSON := `{"initialSyncDays": 30, "setupCompleted": true}`
				userRows := sqlmock.NewRows([]string{
					"id", "github_user_id", "github_username", "github_token_encrypted", "created_at", "updated_at", "sync_settings", "retention_settings", "muted_until",
				}).AddRow(1, sql.NullString{}, sql.NullString{}, sql.NullString{}, time.Now(), time.Now(), []byte(syncSettingsJSON), sql.NullString{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM users`).WillReturnRows(userRows)

				// GetSyncState returns state with initial sync completed
				completedAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
				latestNotification := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)
				syncStateRows := sqlmock.NewRows([]string{
					"id", "user_id", "last_successful_poll", "last_notification_etag", "created_at", "updated_at",
					"latest_notification_at", "initial_sync_completed_at", "oldest_notification_synced_at",
				}).AddRow(
					1, "test-user-id", time.Now(), sql.NullString{}, time.Now(), time.Now(),
					sql.NullTime{
						Valid: true,
						Time:  latestNotification,
					}, sql.NullTime{Valid: true, Time: completedAt}, sql.NullTime{},
				)
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).WillReturnRows(syncStateRows)
			},
			expectedContext: SyncContext{
				IsSyncConfigured: true,
				IsInitialSync:    false,
				// SinceTimestamp should be set to latestNotification
				UnreadOnly: false,
			},
			expectError: false,
		},
		{
			name: "sync not configured (no user settings)",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GetUser returns user with null sync settings
				userRows := sqlmock.NewRows([]string{
					"id", "github_user_id", "github_username", "github_token_encrypted", "created_at", "updated_at", "sync_settings", "retention_settings", "muted_until",
				}).AddRow(1, sql.NullString{}, sql.NullString{}, sql.NullString{}, time.Now(), time.Now(), nil, sql.NullString{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM users`).WillReturnRows(userRows)
			},
			expectedContext: SyncContext{
				IsSyncConfigured: false,
			},
			expectError: false,
		},
		{
			name: "sync not configured (setup not completed)",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GetUser returns user with sync settings where setup is not completed
				syncSettingsJSON := `{"initialSyncDays": 30, "setupCompleted": false}`
				userRows := sqlmock.NewRows([]string{
					"id", "github_user_id", "github_username", "github_token_encrypted", "created_at", "updated_at", "sync_settings", "retention_settings", "muted_until",
				}).AddRow(1, sql.NullString{}, sql.NullString{}, sql.NullString{}, time.Now(), time.Now(), []byte(syncSettingsJSON), sql.NullString{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM users`).WillReturnRows(userRows)
			},
			expectedContext: SyncContext{
				IsSyncConfigured: false,
			},
			expectError: false,
		},
		{
			name: "user retrieval error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM users`).
					WillReturnError(errors.New("database error"))
			},
			expectedContext: SyncContext{},
			expectError:     true,
		},
		{
			name: "sync state retrieval error",
			setupMock: func(mock sqlmock.Sqlmock) {
				// GetUser returns user with sync settings
				syncSettingsJSON := `{"initialSyncDays": 30, "setupCompleted": true}`
				userRows := sqlmock.NewRows([]string{
					"id", "github_user_id", "github_username", "github_token_encrypted", "created_at", "updated_at", "sync_settings", "retention_settings", "muted_until",
				}).AddRow(1, sql.NullString{}, sql.NullString{}, sql.NullString{}, time.Now(), time.Now(), []byte(syncSettingsJSON), sql.NullString{}, sql.NullTime{})
				mock.ExpectQuery(`SELECT (.+) FROM users`).WillReturnRows(userRows)

				// GetSyncState returns error
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
					WillReturnError(errors.New("database error"))
			},
			expectedContext: SyncContext{},
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			dbConn, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer dbConn.Close()

			tt.setupMock(mock)

			mockClient := githubmocks.NewMockClient(ctrl)
			service := setupSyncService(t, dbConn, mockClient)

			result, err := service.GetSyncContext(context.Background(), "test-user-id")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedContext.IsSyncConfigured, result.IsSyncConfigured)
				require.Equal(t, tt.expectedContext.IsInitialSync, result.IsInitialSync)
				require.Equal(t, tt.expectedContext.UnreadOnly, result.UnreadOnly)

				// Check MaxCount if expected
				if tt.expectedContext.MaxCount != nil {
					require.NotNil(t, result.MaxCount)
					require.Equal(t, *tt.expectedContext.MaxCount, *result.MaxCount)
				}

				// For initial sync with nil days, SinceTimestamp should be nil
				if tt.expectedContext.IsSyncConfigured && tt.expectedContext.IsInitialSync && tt.expectedContext.SinceTimestamp == nil && tt.name == "initial sync with all-time (nil days)" {
					require.Nil(t, result.SinceTimestamp)
				}
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestIsInitialSyncComplete tests the IsInitialSyncComplete method
func TestIsInitialSyncComplete(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(sqlmock.Sqlmock)
		expectedResult bool
		expectError    bool
	}{
		{
			name: "initial sync completed returns true",
			setupMock: func(mock sqlmock.Sqlmock) {
				completedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "last_successful_poll", "last_notification_etag", "created_at", "updated_at",
					"latest_notification_at", "initial_sync_completed_at", "oldest_notification_synced_at",
				}).AddRow(
					1, "test-user-id", time.Now(), sql.NullString{}, time.Now(), time.Now(),
					sql.NullTime{
						Valid: true,
						Time:  time.Now(),
					}, sql.NullTime{Valid: true, Time: completedAt}, sql.NullTime{},
				)
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).WillReturnRows(rows)
			},
			expectedResult: true,
			expectError:    false,
		},
		{
			name: "initial sync not completed returns false",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "user_id", "last_successful_poll", "last_notification_etag", "created_at", "updated_at",
					"latest_notification_at", "initial_sync_completed_at", "oldest_notification_synced_at",
				}).AddRow(
					1, "test-user-id", time.Now(), sql.NullString{}, time.Now(), time.Now(),
					sql.NullTime{
						Valid: true,
						Time:  time.Now(),
					}, sql.NullTime{Valid: false}, sql.NullTime{},
				)
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).WillReturnRows(rows)
			},
			expectedResult: false,
			expectError:    false,
		},
		{
			name: "database error returns error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
					WillReturnError(errors.New("database error"))
			},
			expectedResult: false,
			expectError:    true,
		},
		{
			name: "no sync state exists returns false (handled as empty state)",
			setupMock: func(mock sqlmock.Sqlmock) {
				// When no sync state exists, GetSyncState returns sql.ErrNoRows
				// which is handled by returning an empty SyncState (not an error)
				mock.ExpectQuery(`SELECT (.+) FROM sync_state`).
					WillReturnError(sql.ErrNoRows)
			},
			expectedResult: false,
			expectError:    false, // sql.ErrNoRows is handled gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			dbConn, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer dbConn.Close()

			tt.setupMock(mock)

			mockClient := githubmocks.NewMockClient(ctrl)
			service := setupSyncService(t, dbConn, mockClient)

			result, err := service.IsInitialSyncComplete(context.Background(), "test-user-id")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
