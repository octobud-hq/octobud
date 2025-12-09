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

package tag

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

func TestService_ListTagsWithUnreadCounts(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []models.Tag)
	}{
		{
			name: "success with unread counts",
			setupMock: func(m *mocks.MockStore) {
				tags := []db.Tag{
					{ID: "1", Name: "bug", Slug: "bug", DisplayOrder: 0},
					{ID: "2", Name: "feature", Slug: "feature", DisplayOrder: 1},
				}
				m.EXPECT().
					ListAllTags(gomock.Any(), "test-user-id").
					Return(tags, nil)
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{Total: 5}, nil).
					Times(2)
			},
			expectErr: false,
			checkResult: func(t *testing.T, result []models.Tag) {
				require.Len(t, result, 2)
				require.Equal(t, "bug", result[0].Name)
				require.NotNil(t, result[0].UnreadCount)
				require.Equal(t, int64(5), *result[0].UnreadCount)
			},
		},
		{
			name: "error wrapping database failure",
			setupMock: func(m *mocks.MockStore) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					ListAllTags(gomock.Any(), "test-user-id").
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToListTags))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ListTagsWithUnreadCounts(ctx, "test-user-id")

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

func TestService_CreateTag(t *testing.T) {
	tests := []struct {
		name        string
		tagName     string
		color       *string
		description *string
		setupMock   func(*mocks.MockStore, string, *string, *string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Tag)
	}{
		{
			name:        "success with all fields",
			tagName:     "bug",
			color:       stringPtr("#ff0000"),
			description: stringPtr("Bug reports"),
			setupMock: func(m *mocks.MockStore, name string, color *string, desc *string) {
				expectedTag := db.Tag{
					ID:          "1",
					Name:        name,
					Slug:        "bug",
					Color:       sql.NullString{String: *color, Valid: true},
					Description: sql.NullString{String: *desc, Valid: true},
				}
				m.EXPECT().
					UpsertTag(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, arg db.UpsertTagParams) (db.Tag, error) {
						require.Equal(t, name, arg.Name)
						require.Equal(t, "bug", arg.Slug)
						require.True(t, arg.Color.Valid)
						require.Equal(t, *color, arg.Color.String)
						return expectedTag, nil
					})
			},
			expectErr: false,
			checkResult: func(t *testing.T, tag db.Tag) {
				require.Equal(t, "bug", tag.Name)
				require.Equal(t, "bug", tag.Slug)
			},
		},
		{
			name:        "success with minimal fields",
			tagName:     "feature",
			color:       nil,
			description: nil,
			setupMock: func(m *mocks.MockStore, name string, _ *string, _ *string) {
				expectedTag := db.Tag{
					ID:   "2",
					Name: name,
					Slug: "feature",
				}
				m.EXPECT().
					UpsertTag(gomock.Any(), "test-user-id", gomock.Any()).
					Return(expectedTag, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, tag db.Tag) {
				require.Equal(t, "feature", tag.Name)
			},
		},
		{
			name:        "invalid name cannot generate slug",
			tagName:     "---",
			color:       nil,
			description: nil,
			setupMock: func(_ *mocks.MockStore, _ string, _ *string, _ *string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Equal(t, ErrInvalidTagName, err)
			},
		},
		{
			name:        "error wrapping database failure",
			tagName:     "bug",
			color:       nil,
			description: nil,
			setupMock: func(m *mocks.MockStore, _ string, _ *string, _ *string) {
				dbError := errors.New("unique constraint violation")
				m.EXPECT().
					UpsertTag(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToCreateTag))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagName, tt.color, tt.description)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.CreateTag(
				ctx,
				"test-user-id",
				tt.tagName,
				tt.color,
				tt.description,
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

func TestService_UpdateTag(t *testing.T) {
	tests := []struct {
		name        string
		tagID       string
		tagName     string
		color       *string
		description *string
		setupMock   func(*mocks.MockStore, string, string, *string, *string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Tag)
	}{
		{
			name:        "success with all fields",
			tagID:       "1",
			tagName:     "bug",
			color:       stringPtr("#00ff00"),
			description: nil,
			setupMock: func(m *mocks.MockStore, id string, name string, color *string, _ *string) {
				expectedTag := db.Tag{
					ID:    id,
					Name:  name,
					Slug:  "bug",
					Color: sql.NullString{String: *color, Valid: true},
				}
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, arg db.UpdateTagParams) (db.Tag, error) {
						require.Equal(t, id, arg.ID)
						require.Equal(t, name, arg.Name)
						return expectedTag, nil
					})
			},
			expectErr: false,
			checkResult: func(t *testing.T, tag db.Tag) {
				require.Equal(t, "bug", tag.Name)
			},
		},
		{
			name:        "not found returns ErrTagNotFound",
			tagID:       "999",
			tagName:     "nonexistent",
			color:       nil,
			description: nil,
			setupMock: func(m *mocks.MockStore, _ string, _ string, _ *string, _ *string) {
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.Tag{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrTagNotFound))
				require.ErrorIs(t, err, sql.ErrNoRows)
			},
		},
		{
			name:        "error wrapping database failure",
			tagID:       "1",
			tagName:     "bug",
			color:       nil,
			description: nil,
			setupMock: func(m *mocks.MockStore, _ string, _ string, _ *string, _ *string) {
				dbError := errors.New("database error")
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToUpdateTag))
			},
		},
		{
			name:        "invalid name cannot generate slug",
			tagID:       "1",
			tagName:     "---",
			color:       nil,
			description: nil,
			setupMock: func(_ *mocks.MockStore, _ string, _ string, _ *string, _ *string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Equal(t, ErrInvalidTagName, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagID, tt.tagName, tt.color, tt.description)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UpdateTag(
				ctx,
				"test-user-id",
				tt.tagID,
				tt.tagName,
				tt.color,
				tt.description,
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

func TestService_DeleteTag(t *testing.T) {
	tests := []struct {
		name      string
		tagID     string
		setupMock func(*mocks.MockStore, string)
		expectErr bool
		checkErr  func(*testing.T, error)
	}{
		{
			name:  "success",
			tagID: "1",
			setupMock: func(m *mocks.MockStore, id string) {
				m.EXPECT().
					DeleteTag(gomock.Any(), "test-user-id", id).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name:  "error wrapping database failure",
			tagID: "1",
			setupMock: func(m *mocks.MockStore, id string) {
				dbError := errors.New("foreign key constraint violation")
				m.EXPECT().
					DeleteTag(gomock.Any(), "test-user-id", id).
					Return(dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToDeleteTag))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			err := service.DeleteTag(ctx, "test-user-id", tt.tagID)

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

func TestService_ReorderTags(t *testing.T) {
	tests := []struct {
		name        string
		tagIDs      []string
		setupMock   func(*mocks.MockStore, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []db.Tag)
	}{
		{
			name:   "success reorders tags",
			tagIDs: []string{"3", "1", "2"},
			setupMock: func(m *mocks.MockStore, ids []string) {
				m.EXPECT().
					UpdateTagDisplayOrder(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, arg db.UpdateTagDisplayOrderParams) error {
						require.Contains(t, ids, arg.ID)
						return nil
					}).
					Times(len(ids))
				expectedTags := []db.Tag{
					{ID: "1", Name: "tag1", DisplayOrder: 100},
					{ID: "2", Name: "tag2", DisplayOrder: 200},
					{ID: "3", Name: "tag3", DisplayOrder: 0},
				}
				m.EXPECT().
					ListAllTags(gomock.Any(), "test-user-id").
					Return(expectedTags, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, tags []db.Tag) {
				require.Len(t, tags, 3)
			},
		},
		{
			name:   "empty IDs returns error",
			tagIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Equal(t, ErrTagIDsRequired, err)
			},
		},
		{
			name:   "error wrapping database failure",
			tagIDs: []string{"1"},
			setupMock: func(m *mocks.MockStore, _ []string) {
				dbError := errors.New("database error")
				m.EXPECT().
					UpdateTagDisplayOrder(gomock.Any(), "test-user-id", gomock.Any()).
					Return(dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToUpdateTagDisplayOrder))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ReorderTags(ctx, "test-user-id", tt.tagIDs)

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

func TestService_GetTag(t *testing.T) {
	tests := []struct {
		name        string
		tagID       string
		setupMock   func(*mocks.MockStore, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Tag)
	}{
		{
			name:  "success",
			tagID: "1",
			setupMock: func(m *mocks.MockStore, id string) {
				expectedTag := db.Tag{
					ID:   id,
					Name: "bug",
					Slug: "bug",
				}
				m.EXPECT().
					GetTag(gomock.Any(), "test-user-id", id).
					Return(expectedTag, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, tag db.Tag) {
				require.Equal(t, "1", tag.ID)
				require.Equal(t, "bug", tag.Name)
			},
		},
		{
			name:  "error wrapping database failure",
			tagID: "1",
			setupMock: func(m *mocks.MockStore, id string) {
				dbError := errors.New("database error")
				m.EXPECT().
					GetTag(gomock.Any(), "test-user-id", id).
					Return(db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetTag))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetTag(ctx, "test-user-id", tt.tagID)

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

func TestService_GetTagByName(t *testing.T) {
	tests := []struct {
		name        string
		tagName     string
		setupMock   func(*mocks.MockStore, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Tag)
	}{
		{
			name:    "success",
			tagName: "bug",
			setupMock: func(m *mocks.MockStore, name string) {
				expectedTag := db.Tag{
					ID:   "1",
					Name: name,
					Slug: "bug",
				}
				m.EXPECT().
					GetTagByName(gomock.Any(), "test-user-id", name).
					Return(expectedTag, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, tag db.Tag) {
				require.Equal(t, "bug", tag.Name)
			},
		},
		{
			name:    "error wrapping database failure",
			tagName: "bug",
			setupMock: func(m *mocks.MockStore, name string) {
				dbError := errors.New("database error")
				m.EXPECT().
					GetTagByName(gomock.Any(), "test-user-id", name).
					Return(db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetTagByName))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagName)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetTagByName(ctx, "test-user-id", tt.tagName)

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

func TestService_CalculateTagUnreadCount(t *testing.T) {
	tests := []struct {
		name          string
		tagSlug       string
		setupMock     func(*mocks.MockStore, string)
		expectErr     bool
		checkErr      func(*testing.T, error)
		expectedCount int64
	}{
		{
			name:    "success returns count",
			tagSlug: "bug",
			setupMock: func(m *mocks.MockStore, _ string) {
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{Total: 5}, nil)
			},
			expectErr:     false,
			expectedCount: 5,
		},
		{
			name:    "error wrapping query build failure",
			tagSlug: "bug",
			setupMock: func(m *mocks.MockStore, _ string) {
				dbError := errors.New("query build failed")
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(
					t,
					errors.Is(err, ErrFailedToBuildQuery) ||
						errors.Is(err, ErrFailedToListNotifications),
				)
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.tagSlug)
			service := NewService(mockQuerier)

			ctx := context.Background()
			count, err := service.calculateTagUnreadCount(ctx, "test-user-id", tt.tagSlug)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, tt.expectedCount, count)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCount, count)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
