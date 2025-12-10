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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	notificationmocks "github.com/octobud-hq/octobud/backend/internal/core/notification/mocks"
	"github.com/octobud-hq/octobud/backend/internal/core/pullrequest"
	pullrequestmocks "github.com/octobud-hq/octobud/backend/internal/core/pullrequest/mocks"
	"github.com/octobud-hq/octobud/backend/internal/core/repository"
	repositorymocks "github.com/octobud-hq/octobud/backend/internal/core/repository/mocks"
	"github.com/octobud-hq/octobud/backend/internal/core/syncstate"
	syncstatemocks "github.com/octobud-hq/octobud/backend/internal/core/syncstate/mocks"
	"github.com/octobud-hq/octobud/backend/internal/db"
	dbmocks "github.com/octobud-hq/octobud/backend/internal/db/mocks"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// mockClock returns a fixed time for testing
func mockClock() time.Time {
	return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
}

// setupSyncService creates a Service with mocked dependencies for testing
func setupSyncService(
	_ *gomock.Controller,
	mockClient githubinterfaces.Client,
	mockSyncState syncstate.SyncStateService,
	mockRepository repository.RepositoryService,
	mockPullRequest pullrequest.PullRequestService,
	mockNotification notification.NotificationService,
	mockUserStore db.Store,
) *Service {
	return NewService(
		zap.NewNop(),
		mockClock,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)
}

// TestFetchNotificationsToSync_InitialSync tests fetching notifications when no sync state exists
func TestFetchNotificationsToSync_InitialSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)
	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    true,
		SinceTimestamp:   nil,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.NoError(t, err)
	require.Len(t, threads, 1)
	require.Equal(t, "notif-1", threads[0].ID)
}

// TestFetchNotificationsToSync_WithExistingState tests fetching with existing sync state
func TestFetchNotificationsToSync_WithExistingState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)
	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    false,
		SinceTimestamp:   &latestNotification,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.NoError(t, err)
	require.Len(t, threads, 1)
	require.Equal(t, "notif-2", threads[0].ID)
	require.NotNil(t, capturedSince)
	require.Equal(t, latestNotification, *capturedSince)
}

// TestFetchNotificationsToSync_EmptyResults tests when GitHub returns no new notifications
func TestFetchNotificationsToSync_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]types.NotificationThread{}, nil)

	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)
	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    false,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.NoError(t, err)
	require.Len(t, threads, 0)
}

// TestFetchNotificationsToSync_ClientError tests handling of GitHub client errors
func TestFetchNotificationsToSync_ClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockClient.EXPECT().
		FetchNotifications(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("API error"))

	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)
	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	syncCtx := SyncContext{
		IsSyncConfigured: true,
		IsInitialSync:    true,
	}

	threads, err := service.FetchNotificationsToSync(context.Background(), syncCtx)

	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch notifications")
	require.Nil(t, threads)
}

// TestUpdateSyncStateAfterProcessing_Success tests successful sync state update
func TestUpdateSyncStateAfterProcessing_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	latestUpdate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	mockSyncState.EXPECT().
		UpsertSyncStateWithInitialSync(gomock.Any(), "test-user-id", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(models.SyncState{}, nil)

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	err := service.UpdateSyncStateAfterProcessing(
		context.Background(),
		"test-user-id",
		latestUpdate,
	)

	require.NoError(t, err)
}

// TestUpdateSyncStateAfterProcessing_ZeroTime tests update with zero time
func TestUpdateSyncStateAfterProcessing_ZeroTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	mockSyncState.EXPECT().
		UpsertSyncStateWithInitialSync(gomock.Any(), "test-user-id", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(models.SyncState{}, nil)

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	err := service.UpdateSyncStateAfterProcessing(context.Background(), "test-user-id", time.Time{})

	require.NoError(t, err)
}

// TestProcessNotification_RepositoryError tests repository upsert failure
func TestProcessNotification_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	mockRepository.EXPECT().
		UpsertRepository(gomock.Any(), "test-user-id", gomock.Any()).
		Return(db.Repository{}, errors.New("database error"))

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	err := service.ProcessNotification(context.Background(), "test-user-id", thread)

	require.Error(t, err)
	require.Contains(t, err.Error(), "upsert repository")
}

// TestRefreshSubjectData_NotificationNotFound tests error when notification doesn't exist
func TestRefreshSubjectData_NotificationNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	mockNotification.EXPECT().
		GetByGithubID(gomock.Any(), "test-user-id", "nonexistent").
		Return(db.Notification{}, sql.ErrNoRows)

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	_, err := service.RefreshSubjectData(context.Background(), "test-user-id", "nonexistent")

	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

// TestProcessNotification_RetriableSubjectFetchError tests that retriable errors cause ProcessNotification to fail
func TestProcessNotification_RetriableSubjectFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Test Issue",
			Type:  "Issue",
			URL:   "https://api.github.com/repos/owner/test-repo/issues/1",
		},
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	mockRepository.EXPECT().
		UpsertRepository(gomock.Any(), "test-user-id", gomock.Any()).
		Return(db.Repository{ID: 1}, nil)

	mockClient.EXPECT().
		FetchSubjectRaw(gomock.Any(), "https://api.github.com/repos/owner/test-repo/issues/1").
		Return(nil, errors.New("github: subject status 500: internal server error"))

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	err := service.ProcessNotification(context.Background(), "test-user-id", thread)

	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch subject")
}

// TestProcessNotification_NonRetriableSubjectFetchError tests that non-retriable errors allow processing to continue
func TestProcessNotification_NonRetriableSubjectFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Test Issue",
			Type:  "Issue",
			URL:   "https://api.github.com/repos/owner/test-repo/issues/1",
		},
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	mockRepository.EXPECT().
		UpsertRepository(gomock.Any(), "test-user-id", gomock.Any()).
		Return(db.Repository{ID: 1}, nil)

	mockClient.EXPECT().
		FetchSubjectRaw(gomock.Any(), "https://api.github.com/repos/owner/test-repo/issues/1").
		Return(nil, errors.New("github: subject status 403: forbidden"))

	mockNotification.EXPECT().
		UpsertNotification(gomock.Any(), "test-user-id", gomock.Any()).
		Return(db.Notification{ID: 1, GithubID: "notif-123"}, nil)

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	err := service.ProcessNotification(context.Background(), "test-user-id", thread)

	require.NoError(t, err)
}

// TestRefreshSubjectData_ExtractsAuthor tests that RefreshSubjectData extracts and saves author information
func TestRefreshSubjectData_ExtractsAuthor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := githubmocks.NewMockClient(ctrl)
	mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
	mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
	mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
	mockNotification := notificationmocks.NewMockNotificationService(ctrl)
	mockUserStore := dbmocks.NewMockStore(ctrl)

	notif := db.Notification{
		ID:           1,
		UserID:       "test-user-id",
		GithubID:     "notif-123",
		RepositoryID: 1,
		SubjectType:  "Issue", // Not a PullRequest, so GetRepositoryByID won't be called
		SubjectURL: sql.NullString{
			String: "https://api.github.com/repos/owner/test-repo/issues/1",
			Valid:  true,
		},
		SubjectFetchedAt: sql.NullTime{Valid: false},
		AuthorLogin:      sql.NullString{Valid: false},
	}
	mockNotification.EXPECT().
		GetByGithubID(gomock.Any(), "test-user-id", "notif-123").
		Return(notif, nil)

	subjectJSON := `{"id": 1, "number": 42, "title": "Test Issue", "user": {"login": "testuser", "id": 12345}}`
	mockClient.EXPECT().
		FetchSubjectRaw(gomock.Any(), "https://api.github.com/repos/owner/test-repo/issues/1").
		Return([]byte(subjectJSON), nil)

	mockNotification.EXPECT().
		UpdateNotificationSubject(gomock.Any(), "test-user-id", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params db.UpdateNotificationSubjectParams) error {
			require.True(t, params.AuthorLogin.Valid)
			require.Equal(t, "testuser", params.AuthorLogin.String)
			require.True(t, params.AuthorID.Valid)
			require.Equal(t, int64(12345), params.AuthorID.Int64)
			return nil
		})

	service := setupSyncService(
		ctrl,
		mockClient,
		mockSyncState,
		mockRepository,
		mockPullRequest,
		mockNotification,
		mockUserStore,
	)

	wasMissing, err := service.RefreshSubjectData(context.Background(), "test-user-id", "notif-123")

	require.NoError(t, err)
	require.True(t, wasMissing)
}

// TestRefreshSubjectData_WasMissing tests the wasMissing return value
func TestRefreshSubjectData_WasMissing(t *testing.T) {
	subjectJSON := `{"id": 1, "number": 42, "title": "Test Issue", "user": {"login": "testuser", "id": 12345}}`

	tests := []struct {
		name            string
		subjectFetched  sql.NullTime
		authorLogin     sql.NullString
		expectedMissing bool
	}{
		{
			name:            "both missing returns true",
			subjectFetched:  sql.NullTime{Valid: false},
			authorLogin:     sql.NullString{Valid: false},
			expectedMissing: true,
		},
		{
			name:            "subjectFetched missing returns true",
			subjectFetched:  sql.NullTime{Valid: false},
			authorLogin:     sql.NullString{String: "testuser", Valid: true},
			expectedMissing: true,
		},
		{
			name:            "authorLogin missing returns true",
			subjectFetched:  sql.NullTime{Time: time.Now(), Valid: true},
			authorLogin:     sql.NullString{Valid: false},
			expectedMissing: true,
		},
		{
			name:            "both present returns false",
			subjectFetched:  sql.NullTime{Time: time.Now(), Valid: true},
			authorLogin:     sql.NullString{String: "testuser", Valid: true},
			expectedMissing: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := githubmocks.NewMockClient(ctrl)
			mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
			mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
			mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
			mockNotification := notificationmocks.NewMockNotificationService(ctrl)
			mockUserStore := dbmocks.NewMockStore(ctrl)

			notif := db.Notification{
				ID:           1,
				UserID:       "test-user-id",
				GithubID:     "notif-123",
				RepositoryID: 1,
				SubjectType:  "Issue", // Not a PullRequest, so GetRepositoryByID won't be called
				SubjectURL: sql.NullString{
					String: "https://api.github.com/repos/owner/test-repo/issues/1",
					Valid:  true,
				},
				SubjectFetchedAt: tt.subjectFetched,
				AuthorLogin:      tt.authorLogin,
			}
			mockNotification.EXPECT().
				GetByGithubID(gomock.Any(), "test-user-id", "notif-123").
				Return(notif, nil)

			mockClient.EXPECT().
				FetchSubjectRaw(gomock.Any(), "https://api.github.com/repos/owner/test-repo/issues/1").
				Return([]byte(subjectJSON), nil)

			// GetRepositoryByID not called for Issue type
			mockNotification.EXPECT().
				UpdateNotificationSubject(gomock.Any(), "test-user-id", gomock.Any()).
				Return(nil)

			service := setupSyncService(
				ctrl,
				mockClient,
				mockSyncState,
				mockRepository,
				mockPullRequest,
				mockNotification,
				mockUserStore,
			)

			wasMissing, err := service.RefreshSubjectData(
				context.Background(),
				"test-user-id",
				"notif-123",
			)

			require.NoError(t, err)
			require.Equal(t, tt.expectedMissing, wasMissing)
		})
	}
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

			diff := now.Sub(result)
			require.InDelta(t, tt.expectedDiff.Seconds(), diff.Seconds(), 1.0)
			require.Equal(t, time.UTC, result.Location())
		})
	}
}

// TestApplyInitialSyncLimitsFromContext tests the applyInitialSyncLimitsFromContext helper
func TestApplyInitialSyncLimitsFromContext(t *testing.T) {
	createThreads := func(count int, allUnread bool) []types.NotificationThread {
		threads := make([]types.NotificationThread, count)
		for i := 0; i < count; i++ {
			threads[i] = types.NotificationThread{
				ID:     string(rune('A' + i)),
				Unread: allUnread || i%2 == 0,
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
		setupMocks      func(*gomock.Controller) (db.Store, syncstate.SyncStateService)
		expectedContext SyncContext
		expectError     bool
	}{
		{
			name: "initial sync with days configured",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				syncSettingsJSON := `{"initialSyncDays": 30, "initialSyncMaxCount": 100, "initialSyncUnreadOnly": true, "setupCompleted": true}`
				user := db.User{
					ID: 1,
					SyncSettings: db.NullRawMessage{
						RawMessage: json.RawMessage(syncSettingsJSON),
						Valid:      true,
					},
				}
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						InitialSyncCompletedAt: sql.NullTime{Valid: false},
					}, nil)

				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{
				UserID:           "test-user-id",
				IsSyncConfigured: true,
				IsInitialSync:    true,
				MaxCount:         intPtr(100),
				UnreadOnly:       true,
			},
			expectError: false,
		},
		{
			name: "initial sync with all-time (nil days)",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				syncSettingsJSON := `{"initialSyncMaxCount": 500, "setupCompleted": true}`
				user := db.User{
					ID: 1,
					SyncSettings: db.NullRawMessage{
						RawMessage: json.RawMessage(syncSettingsJSON),
						Valid:      true,
					},
				}
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				// GetSyncState returns empty state when no rows found (handled gracefully)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						InitialSyncCompletedAt: sql.NullTime{Valid: false},
					}, nil)

				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{
				UserID:           "test-user-id",
				IsSyncConfigured: true,
				IsInitialSync:    true,
				SinceTimestamp:   nil,
				MaxCount:         intPtr(500),
				UnreadOnly:       false,
			},
			expectError: false,
		},
		{
			name: "regular sync (initial sync already completed)",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				syncSettingsJSON := `{"initialSyncDays": 30, "setupCompleted": true}`
				user := db.User{
					ID: 1,
					SyncSettings: db.NullRawMessage{
						RawMessage: json.RawMessage(syncSettingsJSON),
						Valid:      true,
					},
				}
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				completedAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
				latestNotification := time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						LatestNotificationAt:   sql.NullTime{Valid: true, Time: latestNotification},
						InitialSyncCompletedAt: sql.NullTime{Valid: true, Time: completedAt},
					}, nil)

				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{
				UserID:           "test-user-id",
				IsSyncConfigured: true,
				IsInitialSync:    false,
				SinceTimestamp:   timePtr(time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)),
				UnreadOnly:       false,
			},
			expectError: false,
		},
		{
			name: "sync not configured (no user settings)",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				user := db.User{
					ID:           1,
					SyncSettings: db.NullRawMessage{Valid: false},
				}
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{
				UserID:           "test-user-id",
				IsSyncConfigured: false,
			},
			expectError: false,
		},
		{
			name: "sync not configured (setup not completed)",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				syncSettingsJSON := `{"initialSyncDays": 30, "setupCompleted": false}`
				user := db.User{
					ID: 1,
					SyncSettings: db.NullRawMessage{
						RawMessage: json.RawMessage(syncSettingsJSON),
						Valid:      true,
					},
				}
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{
				UserID:           "test-user-id",
				IsSyncConfigured: false,
			},
			expectError: false,
		},
		{
			name: "user retrieval error",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(db.User{}, errors.New("database error"))

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{},
			expectError:     true,
		},
		{
			name: "sync state retrieval error",
			setupMocks: func(ctrl *gomock.Controller) (db.Store, syncstate.SyncStateService) {
				mockUserStore := dbmocks.NewMockStore(ctrl)
				syncSettingsJSON := `{"initialSyncDays": 30, "setupCompleted": true}`
				user := db.User{
					ID: 1,
					SyncSettings: db.NullRawMessage{
						RawMessage: json.RawMessage(syncSettingsJSON),
						Valid:      true,
					},
				}
				mockUserStore.EXPECT().
					GetUser(gomock.Any()).
					Return(user, nil)

				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{}, errors.New("database error"))

				return mockUserStore, mockSyncState
			},
			expectedContext: SyncContext{},
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserStore, mockSyncState := tt.setupMocks(ctrl)
			mockClient := githubmocks.NewMockClient(ctrl)
			mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
			mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
			mockNotification := notificationmocks.NewMockNotificationService(ctrl)

			service := setupSyncService(
				ctrl,
				mockClient,
				mockSyncState,
				mockRepository,
				mockPullRequest,
				mockNotification,
				mockUserStore,
			)

			result, err := service.GetSyncContext(context.Background(), "test-user-id")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedContext.UserID, result.UserID)
				require.Equal(t, tt.expectedContext.IsSyncConfigured, result.IsSyncConfigured)
				require.Equal(t, tt.expectedContext.IsInitialSync, result.IsInitialSync)
				require.Equal(t, tt.expectedContext.UnreadOnly, result.UnreadOnly)

				if tt.expectedContext.MaxCount != nil {
					require.NotNil(t, result.MaxCount)
					require.Equal(t, *tt.expectedContext.MaxCount, *result.MaxCount)
				}

				if tt.expectedContext.SinceTimestamp != nil {
					require.NotNil(t, result.SinceTimestamp)
					require.Equal(t, tt.expectedContext.SinceTimestamp.Unix(), result.SinceTimestamp.Unix())
				} else if tt.name == "initial sync with all-time (nil days)" {
					require.Nil(t, result.SinceTimestamp)
				}
			}
		})
	}
}

// TestIsInitialSyncComplete tests the IsInitialSyncComplete method
func TestIsInitialSyncComplete(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*gomock.Controller) syncstate.SyncStateService
		expectedResult bool
		expectError    bool
	}{
		{
			name: "initial sync completed returns true",
			setupMocks: func(ctrl *gomock.Controller) syncstate.SyncStateService {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				completedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						LatestNotificationAt:   sql.NullTime{Valid: true, Time: time.Now()},
						InitialSyncCompletedAt: sql.NullTime{Valid: true, Time: completedAt},
					}, nil)
				return mockSyncState
			},
			expectedResult: true,
			expectError:    false,
		},
		{
			name: "initial sync not completed returns false",
			setupMocks: func(ctrl *gomock.Controller) syncstate.SyncStateService {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						LatestNotificationAt:   sql.NullTime{Valid: true, Time: time.Now()},
						InitialSyncCompletedAt: sql.NullTime{Valid: false},
					}, nil)
				return mockSyncState
			},
			expectedResult: false,
			expectError:    false,
		},
		{
			name: "database error returns error",
			setupMocks: func(ctrl *gomock.Controller) syncstate.SyncStateService {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{}, errors.New("database error"))
				return mockSyncState
			},
			expectedResult: false,
			expectError:    true,
		},
		{
			name: "no sync state exists returns false",
			setupMocks: func(ctrl *gomock.Controller) syncstate.SyncStateService {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{}, sql.ErrNoRows)
				return mockSyncState
			},
			expectedResult: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSyncState := tt.setupMocks(ctrl)
			mockClient := githubmocks.NewMockClient(ctrl)
			mockRepository := repositorymocks.NewMockRepositoryService(ctrl)
			mockPullRequest := pullrequestmocks.NewMockPullRequestService(ctrl)
			mockNotification := notificationmocks.NewMockNotificationService(ctrl)
			mockUserStore := dbmocks.NewMockStore(ctrl)

			service := setupSyncService(
				ctrl,
				mockClient,
				mockSyncState,
				mockRepository,
				mockPullRequest,
				mockNotification,
				mockUserStore,
			)

			result, err := service.IsInitialSyncComplete(context.Background(), "test-user-id")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// timePtr is a helper function to create a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
