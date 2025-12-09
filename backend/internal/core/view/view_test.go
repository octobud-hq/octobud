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

package view

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestService_GetView(t *testing.T) {
	tests := []struct {
		name        string
		viewID      string
		setupMock   func(*mocks.MockStore, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.View)
	}{
		{
			name:   "success returns view",
			viewID: "1",
			setupMock: func(m *mocks.MockStore, id string) {
				expectedView := db.View{
					ID:   id,
					Name: "My View",
					Slug: "my-view",
				}
				m.EXPECT().
					GetView(gomock.Any(), "test-user-id", id).
					Return(expectedView, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, view db.View) {
				require.Equal(t, "1", view.ID)
				require.Equal(t, "My View", view.Name)
			},
		},
		{
			name:   "error wrapping database failure",
			viewID: "999",
			setupMock: func(m *mocks.MockStore, id string) {
				dbError := errors.New("database error")
				m.EXPECT().
					GetView(gomock.Any(), "test-user-id", id).
					Return(db.View{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetView))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.viewID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetView(ctx, "test-user-id", tt.viewID)

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

func TestService_CreateView(t *testing.T) {
	tests := []struct {
		name        string
		viewName    string
		description *string
		icon        *string
		isDefault   *bool
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.View)
	}{
		{
			name:        "success creates view",
			viewName:    "My View",
			description: stringPtr("A test view"),
			icon:        stringPtr("star"),
			isDefault:   boolPtr(false),
			queryStr:    "is:unread",
			setupMock: func(m *mocks.MockStore, name string, query string) {
				expectedView := db.View{
					ID:    "1",
					Name:  name,
					Slug:  "my-view",
					Query: sql.NullString{String: query, Valid: true},
				}
				m.EXPECT().
					CreateView(gomock.Any(), "test-user-id", gomock.Any()).
					Return(expectedView, nil)
				// calculateViewUnreadCount may be called
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{Total: 5}, nil).
					AnyTimes()
			},
			expectErr: false,
			checkResult: func(t *testing.T, view models.View) {
				require.Equal(t, "My View", view.Name)
				require.Equal(t, "my-view", view.Slug)
			},
		},
		{
			name:        "empty name returns error before DB call",
			viewName:    "",
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    "is:unread",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "name is required")
			},
		},
		{
			name:        "name with no alphanumeric characters returns error",
			viewName:    "---",
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    "is:unread",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(
					t,
					err.Error(),
					"name must contain at least one alphanumeric character",
				)
			},
		},
		{
			name:        "reserved slug returns error before DB call",
			viewName:    "inbox",
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    "is:unread",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "reserved and cannot be used")
			},
		},
		{
			name:        "empty query returns error before DB call",
			viewName:    "My View",
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    "",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "query is required")
			},
		},
		{
			name:        "error wrapping database failure",
			viewName:    "My View",
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    "is:unread",
			setupMock: func(m *mocks.MockStore, _ string, _ string) {
				dbError := errors.New("unique constraint violation")
				m.EXPECT().
					CreateView(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.View{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(
					t,
					errors.Is(err, ErrViewNameAlreadyExists) ||
						errors.Is(err, ErrFailedToCreateView),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.viewName, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.CreateView(
				ctx,
				"test-user-id",
				tt.viewName,
				tt.description,
				tt.icon,
				tt.isDefault,
				tt.queryStr,
			)

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

func TestService_UpdateView(t *testing.T) {
	tests := []struct {
		name        string
		viewID      string
		namePtr     *string
		description *string
		icon        *string
		isDefault   *bool
		queryStr    *string
		setupMock   func(*mocks.MockStore, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.View)
	}{
		{
			name:        "success updates view",
			viewID:      "1",
			namePtr:     stringPtr("Updated View"),
			description: stringPtr("Updated description"),
			icon:        nil,
			isDefault:   boolPtr(true),
			queryStr:    stringPtr("is:read"),
			setupMock: func(m *mocks.MockStore, id string) {
				expectedView := db.View{
					ID:   id,
					Name: "Updated View",
					Slug: "updated-view",
				}
				m.EXPECT().
					UpdateView(gomock.Any(), "test-user-id", gomock.Any()).
					Return(expectedView, nil)
				// calculateViewUnreadCount may be called
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{Total: 0}, nil).
					AnyTimes()
			},
			expectErr: false,
			checkResult: func(t *testing.T, view models.View) {
				require.Equal(t, "Updated View", view.Name)
			},
		},
		{
			name:        "empty name returns error before DB call",
			viewID:      "1",
			namePtr:     stringPtr(""),
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    nil,
			setupMock: func(_ *mocks.MockStore, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "name cannot be empty")
			},
		},
		{
			name:        "empty query returns error before DB call",
			viewID:      "1",
			namePtr:     nil,
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    stringPtr(""),
			setupMock: func(_ *mocks.MockStore, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "query cannot be empty")
			},
		},
		{
			name:        "not found returns ErrViewNotFound",
			viewID:      "999",
			namePtr:     stringPtr("Updated View"),
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    nil,
			setupMock: func(m *mocks.MockStore, _ string) {
				m.EXPECT().
					UpdateView(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.View{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrViewNotFound))
			},
		},
		{
			name:        "error wrapping database failure",
			viewID:      "1",
			namePtr:     stringPtr("Updated View"),
			description: nil,
			icon:        nil,
			isDefault:   nil,
			queryStr:    nil,
			setupMock: func(m *mocks.MockStore, _ string) {
				dbError := errors.New("database error")
				m.EXPECT().
					UpdateView(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.View{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToUpdateView))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.viewID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UpdateView(
				ctx,
				"test-user-id",
				tt.viewID,
				tt.namePtr,
				tt.description,
				tt.icon,
				tt.isDefault,
				tt.queryStr,
			)

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

func TestService_DeleteView(t *testing.T) {
	tests := []struct {
		name                    string
		viewID                  string
		force                   bool
		setupMock               func(*mocks.MockStore, string)
		expectErr               bool
		expectedLinkedRuleCount int
		checkErr                func(*testing.T, error)
	}{
		{
			name:   "success deletes view with no linked rules",
			viewID: "1",
			force:  false,
			setupMock: func(m *mocks.MockStore, id string) {
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), "test-user-id", gomock.Any()).
					Return([]db.Rule{}, nil)
				m.EXPECT().
					DeleteView(gomock.Any(), "test-user-id", id).
					Return(int64(0), nil)
			},
			expectErr:               false,
			expectedLinkedRuleCount: 0,
		},
		{
			name:   "returns linked rule count without deleting",
			viewID: "1",
			force:  false,
			setupMock: func(m *mocks.MockStore, _ string) {
				linkedRules := []db.Rule{
					{ID: "1", Name: "Rule 1"},
					{ID: "2", Name: "Rule 2"},
				}
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), "test-user-id", gomock.Any()).
					Return(linkedRules, nil)
			},
			expectErr:               false,
			expectedLinkedRuleCount: 2,
		},
		{
			name:   "force delete with linked rules deletes view",
			viewID: "1",
			force:  true,
			setupMock: func(m *mocks.MockStore, id string) {
				linkedRules := []db.Rule{
					{ID: "1", Name: "Rule 1"},
					{ID: "2", Name: "Rule 2"},
				}
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), "test-user-id", gomock.Any()).
					Return(linkedRules, nil)
				m.EXPECT().
					DeleteView(gomock.Any(), "test-user-id", id).
					Return(int64(0), nil)
			},
			expectErr:               false,
			expectedLinkedRuleCount: 2,
		},
		{
			name:   "not found returns ErrViewNotFound",
			viewID: "999",
			force:  false,
			setupMock: func(m *mocks.MockStore, id string) {
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), "test-user-id", gomock.Any()).
					Return([]db.Rule{}, nil)
				m.EXPECT().
					DeleteView(gomock.Any(), "test-user-id", id).
					Return(int64(0), sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrViewNotFound))
			},
		},
		{
			name:   "error wrapping check linked rules failure",
			viewID: "1",
			force:  false,
			setupMock: func(m *mocks.MockStore, _ string) {
				dbError := errors.New("database error")
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), "test-user-id", gomock.Any()).
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToCheckLinkedRules))
			},
		},
		{
			name:   "error wrapping delete failure",
			viewID: "1",
			force:  false,
			setupMock: func(m *mocks.MockStore, id string) {
				m.EXPECT().
					GetRulesByViewID(gomock.Any(), "test-user-id", gomock.Any()).
					Return([]db.Rule{}, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					DeleteView(gomock.Any(), "test-user-id", id).
					Return(int64(0), dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToDeleteView))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.viewID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			linkedRuleCount, err := service.DeleteView(ctx, "test-user-id", tt.viewID, tt.force)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedLinkedRuleCount, linkedRuleCount)
			}
		})
	}
}

func TestService_ReorderViews(t *testing.T) {
	tests := []struct {
		name        string
		viewIDs     []string
		setupMock   func(*mocks.MockStore, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []models.View)
	}{
		{
			name:    "success reorders views",
			viewIDs: []string{"3", "1", "2"},
			setupMock: func(m *mocks.MockStore, ids []string) {
				// ReorderViews calls GetView for each ID to verify it exists
				for _, id := range ids {
					m.EXPECT().
						GetView(gomock.Any(), "test-user-id", id).
						Return(db.View{ID: id, Name: "view"}, nil)
				}
				// Then updates the order
				for range ids {
					m.EXPECT().
						UpdateViewOrder(gomock.Any(), "test-user-id", gomock.Any()).
						DoAndReturn(func(_ context.Context, _ string, arg db.UpdateViewOrderParams) error {
							require.Contains(t, ids, arg.ID)
							return nil
						})
				}
				expectedViews := []db.View{
					{ID: "1", Name: "view1", DisplayOrder: 100},
					{ID: "2", Name: "view2", DisplayOrder: 200},
					{ID: "3", Name: "view3", DisplayOrder: 0},
				}
				m.EXPECT().
					ListViews(gomock.Any(), "test-user-id").
					Return(expectedViews, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, views []models.View) {
				require.Len(t, views, 3)
			},
		},
		{
			name:    "empty IDs returns error before DB call",
			viewIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "viewIDs cannot be empty")
			},
		},
		{
			name:    "error wrapping database failure",
			viewIDs: []string{"1"},
			setupMock: func(m *mocks.MockStore, _ []string) {
				// ReorderViews calls GetView first
				m.EXPECT().
					GetView(gomock.Any(), "test-user-id", "1").
					Return(db.View{ID: "1", Name: "view"}, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					UpdateViewOrder(gomock.Any(), "test-user-id", gomock.Any()).
					Return(dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToReorderViews))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.viewIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ReorderViews(ctx, "test-user-id", tt.viewIDs)

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

func boolPtr(b bool) *bool {
	return &b
}
