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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/github/types"
	syncmocks "github.com/octobud-hq/octobud/backend/internal/sync/mocks"
)

func TestProcessNotificationHandler_Success(t *testing.T) {
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
		Reason:    "mention",
		Unread:    true,
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	threadData, err := json.Marshal(thread)
	require.NoError(t, err)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), "test-user-id", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, arg types.NotificationThread) error {
			require.Equal(t, "notif-123", arg.ID)
			require.Equal(t, "owner/test-repo", arg.Repository.FullName)
			return nil
		})

	handler := NewProcessNotificationHandler(nil, mockSync, zap.NewNop())

	err = handler.Handle(context.Background(), "test-user-id", threadData)
	require.NoError(t, err)
}

func TestProcessNotificationHandler_UnmarshalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	// No ProcessNotification call expected since unmarshal fails first

	handler := NewProcessNotificationHandler(nil, mockSync, zap.NewNop())

	err := handler.Handle(context.Background(), "test-user-id", []byte(`invalid json`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid")
}

func TestProcessNotificationHandler_ProcessError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
		},
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	threadData, err := json.Marshal(thread)
	require.NoError(t, err)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), "test-user-id", gomock.Any()).
		Return(errors.New("database error"))

	handler := NewProcessNotificationHandler(nil, mockSync, zap.NewNop())

	err = handler.Handle(context.Background(), "test-user-id", threadData)
	require.Error(t, err)
	require.Contains(t, err.Error(), "database error")
}

func TestProcessNotificationHandler_EmptyData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), "test-user-id", gomock.Any()).
		Return(nil).
		AnyTimes()

	handler := NewProcessNotificationHandler(nil, mockSync, zap.NewNop())

	err := handler.Handle(context.Background(), "test-user-id", []byte(`{}`))
	// Empty object should unmarshal successfully
	require.NoError(t, err)
}

func TestProcessNotificationHandler_PullRequestNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	thread := types.NotificationThread{
		ID: "notif-pr-123",
		Repository: types.RepositorySnapshot{
			ID:       789,
			FullName: "owner/test-repo",
			Name:     "test-repo",
		},
		Subject: types.NotificationSubject{
			Title: "Fix bug",
			Type:  "PullRequest",
			URL:   "https://api.github.com/repos/owner/test-repo/pulls/42",
		},
		Reason:    "review_requested",
		Unread:    true,
		UpdatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	threadData, err := json.Marshal(thread)
	require.NoError(t, err)

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	mockSync.EXPECT().
		ProcessNotification(gomock.Any(), "test-user-id", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, arg types.NotificationThread) error {
			require.Equal(t, "notif-pr-123", arg.ID)
			require.Equal(t, "PullRequest", arg.Subject.Type)
			return nil
		})

	handler := NewProcessNotificationHandler(nil, mockSync, zap.NewNop())

	err = handler.Handle(context.Background(), "test-user-id", threadData)
	require.NoError(t, err)
}

func TestProcessNotificationHandler_NilData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSync := syncmocks.NewMockSyncOperations(ctrl)
	// No ProcessNotification call expected since unmarshal fails first

	handler := NewProcessNotificationHandler(nil, mockSync, zap.NewNop())

	err := handler.Handle(context.Background(), "test-user-id", nil)
	// Nil data should result in unmarshal error
	require.Error(t, err)
}
