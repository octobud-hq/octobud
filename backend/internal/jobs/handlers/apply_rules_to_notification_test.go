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
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	dbmocks "github.com/octobud-hq/octobud/backend/internal/db/mocks"
)

func TestApplyRulesToNotificationHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRulesToNotificationHandler(mockStore, zap.NewNop())

	payload := ApplyRulesToNotificationJobPayload{
		UserID:   "test-user-id",
		GithubID: "notif-123",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Mock notification exists (by githubID)
	mockStore.EXPECT().
		GetNotificationByGithubID(gomock.Any(), "test-user-id", "notif-123").
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	// MatchAndApplyRules calls GetNotificationByID and ListEnabledRulesOrdered
	mockStore.EXPECT().
		GetNotificationByID(gomock.Any(), "test-user-id", int64(1)).
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	// Mock rules list (empty - no rules to apply)
	mockStore.EXPECT().
		ListEnabledRulesOrdered(gomock.Any(), "test-user-id").
		Return([]db.Rule{}, nil)

	err = handler.Handle(context.Background(), "test-user-id", payloadBytes)
	require.NoError(t, err)
}

func TestApplyRulesToNotificationHandler_NotificationNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRulesToNotificationHandler(mockStore, zap.NewNop())

	payload := ApplyRulesToNotificationJobPayload{
		UserID:   "test-user-id",
		GithubID: "notif-123",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Mock notification not found
	mockStore.EXPECT().
		GetNotificationByGithubID(gomock.Any(), "test-user-id", "notif-123").
		Return(db.Notification{}, sql.ErrNoRows)

	err = handler.Handle(context.Background(), "test-user-id", payloadBytes)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestApplyRulesToNotificationHandler_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRulesToNotificationHandler(mockStore, zap.NewNop())

	err := handler.Handle(context.Background(), "test-user-id", []byte(`invalid json`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid character")
}

func TestApplyRulesToNotificationHandler_RuleApplicationFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRulesToNotificationHandler(mockStore, zap.NewNop())

	payload := ApplyRulesToNotificationJobPayload{
		UserID:   "test-user-id",
		GithubID: "notif-123",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Mock notification exists (by githubID)
	mockStore.EXPECT().
		GetNotificationByGithubID(gomock.Any(), "test-user-id", "notif-123").
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	// MatchAndApplyRules calls GetNotificationByID and ListEnabledRulesOrdered
	mockStore.EXPECT().
		GetNotificationByID(gomock.Any(), "test-user-id", int64(1)).
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	// Mock rules list fails
	mockStore.EXPECT().
		ListEnabledRulesOrdered(gomock.Any(), "test-user-id").
		Return(nil, errors.New("database error"))

	// Handler logs the error but returns nil (best-effort, doesn't fail the job)
	err = handler.Handle(context.Background(), "test-user-id", payloadBytes)
	require.NoError(t, err)
}

func TestApplyRulesToNotificationHandler_UsesPayloadUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRulesToNotificationHandler(mockStore, zap.NewNop())

	payload := ApplyRulesToNotificationJobPayload{
		UserID:   "payload-user-id", // Different from context userID
		GithubID: "notif-123",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Should use payload userID, not context userID
	mockStore.EXPECT().
		GetNotificationByGithubID(gomock.Any(), "payload-user-id", "notif-123").
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	// MatchAndApplyRules calls GetNotificationByID and ListEnabledRulesOrdered
	mockStore.EXPECT().
		GetNotificationByID(gomock.Any(), "payload-user-id", int64(1)).
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	mockStore.EXPECT().
		ListEnabledRulesOrdered(gomock.Any(), "payload-user-id").
		Return([]db.Rule{}, nil)

	err = handler.Handle(context.Background(), "context-user-id", payloadBytes)
	require.NoError(t, err)
}

func TestApplyRulesToNotificationHandler_FallsBackToContextUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRulesToNotificationHandler(mockStore, zap.NewNop())

	payload := ApplyRulesToNotificationJobPayload{
		UserID:   "", // Empty - should use context userID
		GithubID: "notif-123",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// Should use context userID when payload userID is empty
	mockStore.EXPECT().
		GetNotificationByGithubID(gomock.Any(), "context-user-id", "notif-123").
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	// MatchAndApplyRules calls GetNotificationByID and ListEnabledRulesOrdered
	mockStore.EXPECT().
		GetNotificationByID(gomock.Any(), "context-user-id", int64(1)).
		Return(db.Notification{
			ID:       1,
			GithubID: "notif-123",
		}, nil)

	mockStore.EXPECT().
		ListEnabledRulesOrdered(gomock.Any(), "context-user-id").
		Return([]db.Rule{}, nil)

	err = handler.Handle(context.Background(), "context-user-id", payloadBytes)
	require.NoError(t, err)
}
