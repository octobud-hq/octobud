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

package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestService_GetRule(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		ruleID      string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.Rule)
	}{
		{
			name:   "success returns rule",
			ruleID: "rule-1",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				actionsJSON, err := json.Marshal(models.RuleActions{SkipInbox: true})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				expectedRule := db.Rule{
					ID:           id,
					UserID:       userID,
					Name:         "My Rule",
					Query:        sql.NullString{String: "is:unread", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				m.EXPECT().
					GetRule(gomock.Any(), userID, id).
					Return(expectedRule, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rule models.Rule) {
				require.Equal(t, "rule-1", rule.ID)
				require.Equal(t, "My Rule", rule.Name)
				require.True(t, rule.Actions.SkipInbox)
			},
		},
		{
			name:   "not found returns ErrRuleNotFound",
			ruleID: "rule-999",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				m.EXPECT().
					GetRule(gomock.Any(), userID, id).
					Return(db.Rule{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrRuleNotFound))
			},
		},
		{
			name:   "error wrapping database failure",
			ruleID: "rule-1",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database error")
				m.EXPECT().
					GetRule(gomock.Any(), userID, id).
					Return(db.Rule{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetRule))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.ruleID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetRule(ctx, testUserID, tt.ruleID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_ListRules(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []models.Rule)
	}{
		{
			name: "success returns rules with data transformation",
			setupMock: func(m *mocks.MockStore, userID string) {
				actionsJSON, err := json.Marshal(models.RuleActions{SkipInbox: true})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				dbRules := []db.Rule{
					{
						ID:           "rule-1",
						UserID:       userID,
						Name:         "Rule 1",
						Query:        sql.NullString{String: "is:unread", Valid: true},
						Actions:      actionsJSON,
						Enabled:      true,
						DisplayOrder: 100,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "rule-2",
						UserID:       userID,
						Name:         "Rule 2",
						Query:        sql.NullString{String: "is:read", Valid: true},
						Actions:      actionsJSON,
						Enabled:      false,
						DisplayOrder: 200,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				m.EXPECT().
					ListRules(gomock.Any(), userID).
					Return(dbRules, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rules []models.Rule) {
				require.Len(t, rules, 2)
				require.Equal(t, "rule-1", rules[0].ID)
				require.Equal(t, "Rule 1", rules[0].Name)
				require.True(t, rules[0].Enabled)
				require.Equal(t, "rule-2", rules[1].ID)
				require.False(t, rules[1].Enabled)
			},
		},
		{
			name: "error wrapping database failure",
			setupMock: func(m *mocks.MockStore, userID string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					ListRules(gomock.Any(), userID).
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToLoadRules))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ListRules(ctx, testUserID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_GetRulesByViewID(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		viewID      string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []db.Rule)
	}{
		{
			name:   "success returns rules for view",
			viewID: "view-1",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				expectedRules := []db.Rule{
					{
						ID:     "rule-1",
						UserID: userID,
						Name:   "Rule 1",
						ViewID: sql.NullString{String: id, Valid: true},
					},
					{
						ID:     "rule-2",
						UserID: userID,
						Name:   "Rule 2",
						ViewID: sql.NullString{String: id, Valid: true},
					},
				}
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), userID, sql.NullString{String: id, Valid: true}).
					Return(expectedRules, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rules []db.Rule) {
				require.Len(t, rules, 2)
				require.Equal(t, "Rule 1", rules[0].Name)
			},
		},
		{
			name:   "error wrapping database failure",
			viewID: "view-1",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database error")
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), userID, sql.NullString{String: id, Valid: true}).
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetRulesByViewID))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.viewID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetRulesByViewID(ctx, testUserID, tt.viewID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_CreateRule(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		params      models.CreateRuleParams
		setupMock   func(*mocks.MockStore, string, models.CreateRuleParams)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.Rule)
	}{
		{
			name: "success creates rule with query",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				Query:   stringPtr("is:unread"),
				Actions: models.RuleActions{SkipInbox: true},
			},
			setupMock: func(m *mocks.MockStore, userID string, params models.CreateRuleParams) {
				m.EXPECT().
					ListRules(gomock.Any(), userID).
					Return([]db.Rule{}, nil)
				actionsJSON, err := json.Marshal(params.Actions)
				if err != nil {
					panic("failed to marshal params.Actions in test setup: " + err.Error())
				}
				expectedRule := db.Rule{
					ID:           "rule-1",
					UserID:       userID,
					Name:         params.Name,
					Query:        sql.NullString{String: *params.Query, Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				m.EXPECT().
					CreateRule(gomock.Any(), userID, gomock.Any()).
					Return(expectedRule, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rule models.Rule) {
				require.Equal(t, "My Rule", rule.Name)
				require.True(t, rule.Actions.SkipInbox)
			},
		},
		{
			name: "success creates rule with viewID",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				ViewID:  stringPtr("view-1"),
				Actions: models.RuleActions{SkipInbox: true},
			},
			setupMock: func(m *mocks.MockStore, userID string, params models.CreateRuleParams) {
				m.EXPECT().
					GetView(gomock.Any(), userID, "view-1").
					Return(db.View{ID: "view-1", UserID: userID, Name: "My View"}, nil)
				m.EXPECT().
					ListRules(gomock.Any(), userID).
					Return([]db.Rule{}, nil)
				actionsJSON, err := json.Marshal(params.Actions)
				if err != nil {
					panic("failed to marshal params.Actions in test setup: " + err.Error())
				}
				expectedRule := db.Rule{
					ID:           "rule-1",
					UserID:       userID,
					Name:         params.Name,
					ViewID:       sql.NullString{String: "view-1", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				m.EXPECT().
					CreateRule(gomock.Any(), userID, gomock.Any()).
					Return(expectedRule, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rule models.Rule) {
				require.Equal(t, "My Rule", rule.Name)
				require.NotNil(t, rule.ViewID)
				require.Equal(t, "view-1", *rule.ViewID)
			},
		},
		{
			name: "empty name returns error before DB call",
			params: models.CreateRuleParams{
				Name:    "",
				Query:   stringPtr("is:unread"),
				Actions: models.RuleActions{},
			},
			setupMock: func(_ *mocks.MockStore, _ string, _ models.CreateRuleParams) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "name is required")
			},
		},
		{
			name: "neither query nor viewID returns error",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				Query:   nil,
				ViewID:  nil,
				Actions: models.RuleActions{},
			},
			setupMock: func(_ *mocks.MockStore, _ string, _ models.CreateRuleParams) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "either query or viewId is required")
			},
		},
		{
			name: "both query and viewID returns error",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				Query:   stringPtr("is:unread"),
				ViewID:  stringPtr("1"),
				Actions: models.RuleActions{},
			},
			setupMock: func(_ *mocks.MockStore, _ string, _ models.CreateRuleParams) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "only one of query or viewId can be provided")
			},
		},
		{
			name: "invalid viewID returns error",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				ViewID:  stringPtr("invalid"),
				Actions: models.RuleActions{},
			},
			setupMock: func(m *mocks.MockStore, userID string, _ models.CreateRuleParams) {
				m.EXPECT().
					GetView(gomock.Any(), userID, "invalid").
					Return(db.View{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrViewNotFound))
			},
		},
		{
			name: "view not found returns ErrViewNotFound",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				ViewID:  stringPtr("999"),
				Actions: models.RuleActions{},
			},
			setupMock: func(m *mocks.MockStore, userID string, _ models.CreateRuleParams) {
				m.EXPECT().
					GetView(gomock.Any(), userID, "999").
					Return(db.View{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrViewNotFound))
			},
		},
		{
			name: "error wrapping database failure",
			params: models.CreateRuleParams{
				Name:    "My Rule",
				Query:   stringPtr("is:unread"),
				Actions: models.RuleActions{},
			},
			setupMock: func(m *mocks.MockStore, userID string, _ models.CreateRuleParams) {
				m.EXPECT().
					ListRules(gomock.Any(), userID).
					Return([]db.Rule{}, nil)
				dbError := errors.New("unique constraint violation")
				m.EXPECT().
					CreateRule(gomock.Any(), userID, gomock.Any()).
					Return(db.Rule{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(
					t,
					errors.Is(err, ErrRuleNameAlreadyExists) ||
						errors.Is(err, ErrFailedToCreateRule),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.params)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.CreateRule(ctx, testUserID, tt.params)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_UpdateRule(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		ruleID      string
		params      models.UpdateRuleParams
		setupMock   func(*mocks.MockStore, string, string, models.UpdateRuleParams)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.Rule)
	}{
		{
			name:   "success updates rule",
			ruleID: "rule-1",
			params: models.UpdateRuleParams{
				Name: stringPtr("Updated Rule"),
			},
			setupMock: func(m *mocks.MockStore, userID, id string, _ models.UpdateRuleParams) {
				actionsJSON, err := json.Marshal(models.RuleActions{SkipInbox: true})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				expectedRule := db.Rule{
					ID:           id,
					UserID:       userID,
					Name:         "Updated Rule",
					Query:        sql.NullString{String: "is:unread", Valid: true},
					Actions:      actionsJSON,
					Enabled:      true,
					DisplayOrder: 100,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				m.EXPECT().
					UpdateRule(gomock.Any(), userID, gomock.Any()).
					Return(expectedRule, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rule models.Rule) {
				require.Equal(t, "Updated Rule", rule.Name)
			},
		},
		{
			name:   "empty name returns error before DB call",
			ruleID: "rule-1",
			params: models.UpdateRuleParams{
				Name: stringPtr(""),
			},
			setupMock: func(_ *mocks.MockStore, _ string, _ string, _ models.UpdateRuleParams) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "name cannot be empty")
			},
		},
		{
			name:   "empty query returns error before DB call",
			ruleID: "rule-1",
			params: models.UpdateRuleParams{
				Query: stringPtr(""),
			},
			setupMock: func(_ *mocks.MockStore, _ string, _ string, _ models.UpdateRuleParams) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "query cannot be empty")
			},
		},
		{
			name:   "not found returns ErrRuleNotFound",
			ruleID: "rule-999",
			params: models.UpdateRuleParams{
				Name: stringPtr("Updated Rule"),
			},
			setupMock: func(m *mocks.MockStore, userID, _ string, _ models.UpdateRuleParams) {
				m.EXPECT().
					UpdateRule(gomock.Any(), userID, gomock.Any()).
					Return(db.Rule{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrRuleNotFound))
			},
		},
		{
			name:   "error wrapping database failure",
			ruleID: "rule-1",
			params: models.UpdateRuleParams{
				Name: stringPtr("Updated Rule"),
			},
			setupMock: func(m *mocks.MockStore, userID, _ string, _ models.UpdateRuleParams) {
				dbError := errors.New("database error")
				m.EXPECT().
					UpdateRule(gomock.Any(), userID, gomock.Any()).
					Return(db.Rule{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToUpdateRule))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.ruleID, tt.params)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UpdateRule(ctx, testUserID, tt.ruleID, tt.params)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_DeleteRule(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name      string
		ruleID    string
		setupMock func(*mocks.MockStore, string, string)
		expectErr bool
		checkErr  func(*testing.T, error)
	}{
		{
			name:   "success deletes rule",
			ruleID: "rule-1",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				m.EXPECT().
					DeleteRule(gomock.Any(), userID, id).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name:   "not found returns ErrRuleNotFound",
			ruleID: "rule-999",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				m.EXPECT().
					DeleteRule(gomock.Any(), userID, id).
					Return(sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrRuleNotFound))
			},
		},
		{
			name:   "error wrapping database failure",
			ruleID: "rule-1",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database error")
				m.EXPECT().
					DeleteRule(gomock.Any(), userID, id).
					Return(dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToDeleteRule))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.ruleID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			err := service.DeleteRule(ctx, testUserID, tt.ruleID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_ReorderRules(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		ruleIDs     []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []models.Rule)
	}{
		{
			name:    "success reorders rules",
			ruleIDs: []string{"rule-3", "rule-1", "rule-2"},
			setupMock: func(m *mocks.MockStore, userID string, ids []string) {
				for range ids {
					m.EXPECT().
						UpdateRuleOrder(gomock.Any(), userID, gomock.Any()).
						DoAndReturn(func(_ context.Context, _ string, arg db.UpdateRuleOrderParams) error {
							require.Contains(t, ids, arg.ID)
							return nil
						})
				}
				actionsJSON, err := json.Marshal(models.RuleActions{})
				if err != nil {
					panic("failed to marshal RuleActions in test setup: " + err.Error())
				}
				expectedRules := []db.Rule{
					{
						ID:           "rule-1",
						UserID:       userID,
						Name:         "rule1",
						DisplayOrder: 100,
						Actions:      actionsJSON,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "rule-2",
						UserID:       userID,
						Name:         "rule2",
						DisplayOrder: 200,
						Actions:      actionsJSON,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
					{
						ID:           "rule-3",
						UserID:       userID,
						Name:         "rule3",
						DisplayOrder: 0,
						Actions:      actionsJSON,
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					},
				}
				m.EXPECT().
					ListRules(gomock.Any(), userID).
					Return(expectedRules, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, rules []models.Rule) {
				require.Len(t, rules, 3)
			},
		},
		{
			name:    "empty IDs returns error before DB call",
			ruleIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "ruleIDs cannot be empty")
			},
		},
		{
			name:    "error wrapping database failure",
			ruleIDs: []string{"rule-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				dbError := errors.New("database error")
				m.EXPECT().
					UpdateRuleOrder(gomock.Any(), userID, gomock.Any()).
					Return(dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToReorderRules))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.ruleIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ReorderRules(ctx, testUserID, tt.ruleIDs)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}
