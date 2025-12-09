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
	syncmocks "github.com/octobud-hq/octobud/backend/internal/sync/mocks"
)

func TestSyncOlderHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30) // 30 days back

	notifications := []types.NotificationThread{
		{
			ID: "notif-old-1",
			Repository: types.RepositorySnapshot{
				ID:       123,
				FullName: "test/repo1",
			},
			UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		},
		{
			ID: "notif-old-2",
			Repository: types.RepositorySnapshot{
				ID:       456,
				FullName: "test/repo2",
			},
			UpdatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
		},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(notifications, nil)
	// The handler will pass:
	// - completedAt: nil (4th param)
	// - oldestNotification: pointer to Jan 5, 2024 (the oldest notification)
	oldestExpectedTime := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)

	// Use Do to capture arguments without gomock trying to format them
	// This avoids the nil pointer panic when gomock tries to call .String() on nil *time.Time
	var capturedCompletedAt *time.Time
	var capturedOldest *time.Time
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(
			gomock.Any(),
			"test-user-id",
			time.Time{},
			gomock.Any(), // completedAt - captured via Do
			gomock.Any(), // oldestNotification - captured via Do
		).
		Do(func(_ context.Context, _ string, _ time.Time, completedAt *time.Time, oldest *time.Time) {
			capturedCompletedAt = completedAt
			capturedOldest = oldest
		}).
		Return(nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncOlderHandler(mockSync, enqueuer, zap.NewNop())

	args := SyncOlderArgs{
		UserID:     "test-user-id",
		Days:       30,
		UntilTime:  untilTime,
		MaxCount:   nil,
		UnreadOnly: false,
	}

	err := handler.Handle(context.Background(), args)
	require.NoError(t, err)
	require.Len(t, enqueuer.enqueuedData, 2)

	// Verify the captured arguments
	require.Nil(t, capturedCompletedAt, "completedAt should be nil")
	require.NotNil(t, capturedOldest, "oldestNotification should not be nil")
	require.Equal(
		t,
		oldestExpectedTime,
		*capturedOldest,
		"oldestNotification should point to Jan 5, 2024",
	)
}

func TestSyncOlderHandler_EmptyResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return([]types.NotificationThread{}, nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncOlderHandler(mockSync, enqueuer, zap.NewNop())

	args := SyncOlderArgs{
		Days:       30,
		UntilTime:  untilTime,
		MaxCount:   nil,
		UnreadOnly: false,
	}

	err := handler.Handle(context.Background(), args)
	require.NoError(t, err)
	require.Len(t, enqueuer.enqueuedData, 0)
}

func TestSyncOlderHandler_WithMaxCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)
	maxCount := 100

	notifications := []types.NotificationThread{
		{ID: "notif-old-1", UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, &maxCount, false).
		Return(notifications, nil)

	// Use Do to capture arguments to avoid nil pointer panic
	// We don't need to verify the values in this test, just avoid the panic
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(gomock.Any(), "test-user-id", gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, _ string, _ time.Time, _ *time.Time, _ *time.Time) {
			// Capture but don't verify - just avoiding gomock's nil pointer formatting
		}).
		Return(nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncOlderHandler(mockSync, enqueuer, zap.NewNop())

	args := SyncOlderArgs{
		UserID:     "test-user-id",
		Days:       30,
		UntilTime:  untilTime,
		MaxCount:   &maxCount,
		UnreadOnly: false,
	}

	err := handler.Handle(context.Background(), args)
	require.NoError(t, err)
}

func TestSyncOlderHandler_WithUnreadOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	notifications := []types.NotificationThread{
		{ID: "notif-old-1", UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC), Unread: true},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), true).
		Return(notifications, nil)

	// Use Do to capture arguments to avoid nil pointer panic
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(gomock.Any(), "test-user-id", gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, _ string, _ time.Time, _ *time.Time, _ *time.Time) {
			// Capture but don't verify - just avoiding gomock's nil pointer formatting
		}).
		Return(nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncOlderHandler(mockSync, enqueuer, zap.NewNop())

	args := SyncOlderArgs{
		UserID:     "test-user-id",
		Days:       30,
		UntilTime:  untilTime,
		MaxCount:   nil,
		UnreadOnly: true,
	}

	err := handler.Handle(context.Background(), args)
	require.NoError(t, err)
}

func TestSyncOlderHandler_FetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(nil, errors.New("API error"))

	enqueuer := &mockEnqueuer{}
	handler := NewSyncOlderHandler(mockSync, enqueuer, zap.NewNop())

	args := SyncOlderArgs{
		Days:       30,
		UntilTime:  untilTime,
		MaxCount:   nil,
		UnreadOnly: false,
	}

	err := handler.Handle(context.Background(), args)
	require.Error(t, err)
	require.Contains(t, err.Error(), "API error")
}

func TestSyncOlderHandler_TracksOldestNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	untilTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	sinceTime := untilTime.AddDate(0, 0, -30)

	// Oldest notification should be Jan 5
	oldestTime := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
	notifications := []types.NotificationThread{
		{ID: "notif-old-1", UpdatedAt: time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)},
		{ID: "notif-old-2", UpdatedAt: oldestTime}, // Oldest
		{ID: "notif-old-3", UpdatedAt: time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC)},
	}

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		FetchOlderNotificationsToSync(gomock.Any(), sinceTime, untilTime, (*int)(nil), false).
		Return(notifications, nil)

	// Use Do to capture arguments to avoid nil pointer panic
	var capturedCompletedAt *time.Time
	var capturedOldest *time.Time
	mockSync.EXPECT().
		UpdateSyncStateAfterProcessingWithInitialSync(
			gomock.Any(),
			"test-user-id",
			time.Time{},
			gomock.Any(), // completedAt - captured via Do
			gomock.Any(), // oldestNotification - captured via Do
		).
		Do(func(_ context.Context, _ string, _ time.Time, completedAt *time.Time, oldest *time.Time) {
			capturedCompletedAt = completedAt
			capturedOldest = oldest
		}).
		Return(nil)

	enqueuer := &mockEnqueuer{}
	handler := NewSyncOlderHandler(mockSync, enqueuer, zap.NewNop())

	args := SyncOlderArgs{
		UserID:     "test-user-id",
		Days:       30,
		UntilTime:  untilTime,
		MaxCount:   nil,
		UnreadOnly: false,
	}

	err := handler.Handle(context.Background(), args)
	require.NoError(t, err)

	// Verify the captured arguments
	require.Nil(t, capturedCompletedAt, "completedAt should be nil")
	require.NotNil(t, capturedOldest, "oldestNotification should not be nil")
	require.Equal(t, oldestTime, *capturedOldest, "oldestNotification should point to Jan 5, 2024")
}
