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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/sync"
	syncmocks "github.com/octobud-hq/octobud/backend/internal/sync/mocks"
)

// mockEnqueuer implements NotificationEnqueuer for testing
type mockEnqueuer struct {
	enqueuedData   [][]byte
	enqueuedUserID string
	err            error
}

func (m *mockEnqueuer) EnqueueProcessNotification(
	_ context.Context,
	userID string,
	data []byte,
) error {
	if m.err != nil {
		return m.err
	}
	m.enqueuedUserID = userID
	m.enqueuedData = append(m.enqueuedData, data)
	return nil
}

func TestSyncNotificationsHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			ID: "notif-2",
			Repository: types.RepositorySnapshot{
				ID:       456,
				FullName: "test/repo2",
			},
			UpdatedAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	syncCtx := sync.SyncContext{IsSyncConfigured: true, IsInitialSync: false}
	mockSync.EXPECT().GetSyncContext(gomock.Any(), "test-user-id").Return(syncCtx, nil)
	mockSync.EXPECT().FetchNotificationsToSync(gomock.Any(), syncCtx).Return(notifications, nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncNotificationsHandler(mockSync, enqueuer, zap.NewNop())

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Threads, 2)
	require.Equal(t, time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC), result.LatestUpdate)
	require.Len(t, enqueuer.enqueuedData, 2)
}

func TestSyncNotificationsHandler_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	syncCtx := sync.SyncContext{IsSyncConfigured: true, IsInitialSync: false}
	mockSync.EXPECT().GetSyncContext(gomock.Any(), "test-user-id").Return(syncCtx, nil)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any(), syncCtx).
		Return([]types.NotificationThread{}, nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncNotificationsHandler(mockSync, enqueuer, zap.NewNop())

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.Nil(t, result) // No result for empty notifications
	require.Len(t, enqueuer.enqueuedData, 0)
}

func TestSyncNotificationsHandler_NotConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		GetSyncContext(gomock.Any(), "test-user-id").
		Return(sync.SyncContext{IsSyncConfigured: false}, nil)

		// No FetchNotificationsToSync call expected

	enqueuer := &mockEnqueuer{}
	handler := NewSyncNotificationsHandler(mockSync, enqueuer, zap.NewNop())

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, enqueuer.enqueuedData, 0)
}

func TestSyncNotificationsHandler_FetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	syncCtx := sync.SyncContext{IsSyncConfigured: true, IsInitialSync: false}
	mockSync.EXPECT().GetSyncContext(gomock.Any(), "test-user-id").Return(syncCtx, nil)
	mockSync.EXPECT().
		FetchNotificationsToSync(gomock.Any(), syncCtx).
		Return(nil, errors.New("API error"))

	enqueuer := &mockEnqueuer{}
	handler := NewSyncNotificationsHandler(mockSync, enqueuer, zap.NewNop())

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error")
	require.Nil(t, result)
}

func TestSyncNotificationsHandler_QueueingFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID: "notif-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	syncCtx := sync.SyncContext{IsSyncConfigured: true, IsInitialSync: false}
	mockSync.EXPECT().GetSyncContext(gomock.Any(), "test-user-id").Return(syncCtx, nil)
	mockSync.EXPECT().FetchNotificationsToSync(gomock.Any(), syncCtx).Return(notifications, nil)

	enqueuer := &mockEnqueuer{err: errors.New("queue full")}
	handler := NewSyncNotificationsHandler(mockSync, enqueuer, zap.NewNop())

	// Handler returns error if any enqueue fails - this prevents sync state update
	// so we'll re-fetch the same notifications on the next sync (safe because processing is idempotent)
	result, err := handler.Handle(context.Background(), "test-user-id")
	require.Error(t, err)
	require.Nil(t, result)
}

func TestSyncNotificationsHandler_InitialSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifications := []types.NotificationThread{
		{
			ID:        "notif-1",
			UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        "notif-2",
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC), // Older
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	syncCtx := sync.SyncContext{IsSyncConfigured: true, IsInitialSync: true}
	mockSync.EXPECT().GetSyncContext(gomock.Any(), "test-user-id").Return(syncCtx, nil)
	mockSync.EXPECT().FetchNotificationsToSync(gomock.Any(), syncCtx).Return(notifications, nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncNotificationsHandler(mockSync, enqueuer, zap.NewNop())

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsInitialSync)
	// Should track oldest notification
	require.Equal(t, time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC), result.OldestNotification)
}

func TestSyncNotificationsHandler_UpdateSyncState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	latestUpdate := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessing(gomock.Any(), "test-user-id", latestUpdate).
		Return(nil)

	handler := NewSyncNotificationsHandler(mockSync, nil, zap.NewNop())

	result := &SyncResult{
		UserID:        "test-user-id",
		LatestUpdate:  latestUpdate,
		IsInitialSync: false,
	}

	err := handler.UpdateSyncState(context.Background(), result)
	require.NoError(t, err)
}
