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

func TestApplyRuleHandler_SuccessWithRuleQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullString{Valid: false},
		Actions: json.RawMessage(`{"markRead": true}`),
	}

	notifications := []db.Notification{
		{ID: 1, GithubID: "notif-1"},
		{ID: 2, GithubID: "notif-2"},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         2,
	}

	// Expect GetRule call
	mockStore.EXPECT().
		GetRule(gomock.Any(), "test-user-id", ruleID).
		Return(rule, nil)

	// Expect ListNotificationsFromQuery call
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
		Return(queryResult, nil)

	// Expect actions to be applied for each notification
	mockStore.EXPECT().
		MarkNotificationRead(gomock.Any(), "test-user-id", "notif-1").
		Return(db.Notification{}, nil)
	mockStore.EXPECT().
		MarkNotificationRead(gomock.Any(), "test-user-id", "notif-2").
		Return(db.Notification{}, nil)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.NoError(t, err)
}

func TestApplyRuleHandler_SuccessWithViewQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	viewID := "10"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false},
		ViewID:  sql.NullString{String: viewID, Valid: true},
		Actions: json.RawMessage(`{"archive": true}`),
	}

	view := db.View{
		ID:    viewID,
		Name:  "Test View",
		Query: sql.NullString{String: "is:unread", Valid: true},
	}

	notifications := []db.Notification{
		{ID: 1, GithubID: "notif-1"},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: notifications,
		Total:         1,
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)
	mockStore.EXPECT().GetView(gomock.Any(), "test-user-id", viewID).Return(view, nil)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
		Return(queryResult, nil)
	mockStore.EXPECT().
		ArchiveNotification(gomock.Any(), "test-user-id", "notif-1").
		Return(db.Notification{}, nil)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.NoError(t, err)
}

func TestApplyRuleHandler_DisabledRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: false, // Disabled
		Query:   sql.NullString{String: "is:unread", Valid: true},
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)
	// No further calls expected for disabled rule

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.NoError(t, err)
}

func TestApplyRuleHandler_RuleNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "999"

	mockStore.EXPECT().
		GetRule(gomock.Any(), "test-user-id", ruleID).
		Return(db.Rule{}, sql.ErrNoRows)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get rule")
}

func TestApplyRuleHandler_ViewNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	viewID := "999"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false},
		ViewID:  sql.NullString{String: viewID, Valid: true},
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)
	mockStore.EXPECT().
		GetView(gomock.Any(), "test-user-id", viewID).
		Return(db.View{}, sql.ErrNoRows)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get view")
}

func TestApplyRuleHandler_ViewWithoutQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	viewID := "10"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false},
		ViewID:  sql.NullString{String: viewID, Valid: true},
	}

	view := db.View{
		ID:    viewID,
		Name:  "Test View",
		Query: sql.NullString{Valid: false}, // No query
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)
	mockStore.EXPECT().GetView(gomock.Any(), "test-user-id", viewID).Return(view, nil)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no query defined")
}

func TestApplyRuleHandler_RuleWithoutQueryOrView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{Valid: false}, // No query
		ViewID:  sql.NullString{Valid: false}, // No view
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has neither query nor viewId set")
}

func TestApplyRuleHandler_QueryBuildError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "badfield:value", Valid: true}, // Unknown field
		ViewID:  sql.NullString{Valid: false},
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to build query")
}

func TestApplyRuleHandler_ListNotificationsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullString{Valid: false},
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)
	mockStore.EXPECT().ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
		Return(db.ListNotificationsFromQueryResult{}, errors.New("database error"))

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to execute query")
}

func TestApplyRuleHandler_EmptyNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStore(ctrl)
	handler := NewApplyRuleHandler(mockStore, zap.NewNop())

	ruleID := "1"
	rule := db.Rule{
		ID:      ruleID,
		Name:    "Test Rule",
		Enabled: true,
		Query:   sql.NullString{String: "is:unread", Valid: true},
		ViewID:  sql.NullString{Valid: false},
	}

	queryResult := db.ListNotificationsFromQueryResult{
		Notifications: []db.Notification{}, // Empty
		Total:         0,
	}

	mockStore.EXPECT().GetRule(gomock.Any(), "test-user-id", ruleID).Return(rule, nil)
	mockStore.EXPECT().
		ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
		Return(queryResult, nil)

	err := handler.Handle(context.Background(), "test-user-id", ruleID)
	require.NoError(t, err)
}
