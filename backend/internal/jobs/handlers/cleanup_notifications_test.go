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

package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
	"github.com/octobud-hq/octobud/backend/internal/jobs/handlers"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestCleanupNotificationsHandler_Handle_DisabledByDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	// User with no retention settings (default disabled)
	mockStore.EXPECT().GetUser(gomock.Any()).Return(db.User{
		ID:                1,
		RetentionSettings: db.NullRawMessage{Valid: false},
	}, nil)

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Skipped)
	require.Equal(t, "cleanup disabled", result.SkipReason)
}

func TestCleanupNotificationsHandler_Handle_EnabledWithValidSettings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	// User with retention settings enabled
	settings := &models.RetentionSettings{
		Enabled:        true,
		RetentionDays:  90,
		ProtectStarred: true,
		ProtectTagged:  true,
	}
	settingsJSON, _ := json.Marshal(settings)

	mockStore.EXPECT().GetUser(gomock.Any()).Return(db.User{
		ID: 1,
		RetentionSettings: db.NullRawMessage{
			RawMessage: settingsJSON,
			Valid:      true,
		},
	}, nil)

	// Expect deletion to be called (with any params that match the criteria)
	mockStore.EXPECT().
		DeleteOldArchivedNotifications(gomock.Any(), "test-user-id", gomock.Any()).
		Return(int64(50), nil)

	// Expect orphaned PRs cleanup
	mockStore.EXPECT().
		DeleteOrphanedPullRequests(gomock.Any(), "test-user-id").
		Return(int64(5), nil)

	// Expect settings update for lastCleanupAt
	mockStore.EXPECT().
		UpdateUserRetentionSettings(gomock.Any(), gomock.Any()).
		Return(db.User{}, nil)

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Skipped)
	require.Equal(t, int64(50), result.NotificationsDeleted)
	require.Equal(t, int64(5), result.PullRequestsDeleted)
}

func TestCleanupNotificationsHandler_Handle_NoUserConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	// No user in DB
	mockStore.EXPECT().GetUser(gomock.Any()).Return(db.User{}, sql.ErrNoRows)

	result, err := handler.Handle(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Skipped)
	require.Equal(t, "no user configured", result.SkipReason)
}

func TestCleanupNotificationsHandler_RunManualCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	// Expect deletion to be called - return 50 (less than batch size of 100) to indicate done
	mockStore.EXPECT().
		DeleteOldArchivedNotifications(gomock.Any(), "test-user-id", gomock.Any()).
		Return(int64(50), nil)

	// Expect orphaned PRs cleanup
	mockStore.EXPECT().
		DeleteOrphanedPullRequests(gomock.Any(), "test-user-id").
		Return(int64(10), nil)

	result, err := handler.RunManualCleanup(context.Background(), "test-user-id", 30, true, true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Skipped)
	require.Equal(t, int64(50), result.NotificationsDeleted)
	require.Equal(t, int64(10), result.PullRequestsDeleted)
}

func TestCleanupNotificationsHandler_RunManualCleanup_InvalidDays(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	result, err := handler.RunManualCleanup(context.Background(), "test-user-id", 0, true, true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Skipped)
	require.Equal(t, "invalid retention days", result.SkipReason)
}

func TestCleanupNotificationsHandler_GetStorageStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	expectedStats := db.StorageStats{
		TotalCount:    1000,
		ArchivedCount: 500,
		StarredCount:  50,
		SnoozedCount:  25,
		UnreadCount:   100,
		TaggedCount:   75,
	}

	mockStore.EXPECT().GetStorageStats(gomock.Any(), "test-user-id").Return(expectedStats, nil)

	stats, err := handler.GetStorageStats(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.Equal(t, expectedStats, stats)
}

func TestCleanupNotificationsHandler_CountEligibleForCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	mockStore.EXPECT().
		CountEligibleForCleanup(gomock.Any(), "test-user-id", gomock.Any()).
		Return(int64(250), nil)

	count, err := handler.CountEligibleForCleanup(
		context.Background(),
		"test-user-id",
		90,
		true,
		true,
	)
	require.NoError(t, err)
	require.Equal(t, int64(250), count)
}

func TestCleanupNotificationsHandler_GetRetentionSettings_Default(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	// User with no retention settings
	mockStore.EXPECT().GetUser(gomock.Any()).Return(db.User{
		ID:                1,
		RetentionSettings: db.NullRawMessage{Valid: false},
	}, nil)

	settings, err := handler.GetRetentionSettings(context.Background(), "test-user-id")
	require.NoError(t, err)
	require.NotNil(t, settings)
	// Should return defaults
	require.False(t, settings.Enabled)
	require.Equal(t, 90, settings.RetentionDays)
	require.True(t, settings.ProtectStarred)
	require.True(t, settings.ProtectTagged)
}

func TestCleanupNotificationsHandler_UpdateRetentionSettings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStore(ctrl)
	logger := zap.NewNop()

	handler := handlers.NewCleanupNotificationsHandler(mockStore, logger)

	settings := &models.RetentionSettings{
		Enabled:        true,
		RetentionDays:  60,
		ProtectStarred: true,
		ProtectTagged:  false,
	}

	mockStore.EXPECT().
		UpdateUserRetentionSettings(gomock.Any(), gomock.Any()).
		Return(db.User{}, nil)

	err := handler.UpdateRetentionSettings(context.Background(), "test-user-id", settings)
	require.NoError(t, err)
}
