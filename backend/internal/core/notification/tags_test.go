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

package notification

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
)

func TestService_AssignTag(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		tagID       string
		setupMock   func(*mocks.MockStore, string, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success assigns tag to notification",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagID string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: tagID, Name: "bug"}
				updatedNotification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{tagID},
				}

				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTag(gomock.Any(), userID, tagID).
					Return(tag, nil)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(updatedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "notif-1", notification.GithubID)
				require.Contains(t, notification.TagIDs, "10")
			},
		},
		{
			name:     "notification not found returns error",
			githubID: "not-found",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID string, _ string) {
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "tag not found returns error",
			githubID: "notif-1",
			tagID:    "999",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagID string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTag(gomock.Any(), userID, tagID).
					Return(db.Tag{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping assign tag failure",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagID string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: tagID, Name: "bug"}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTag(gomock.Any(), userID, tagID).
					Return(tag, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping assign tag failure",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagID string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: tagID, Name: "bug"}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTag(gomock.Any(), userID, tagID).
					Return(tag, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping get updated notification failure",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagID string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: tagID, Name: "bug"}
				// First call to get notification
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTag(gomock.Any(), userID, tagID).
					Return(tag, nil)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
				// Second call to get updated notification fails
				dbError := errors.New("database error")
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(db.Notification{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubID, tt.tagID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.AssignTag(ctx, testUserID, tt.githubID, tt.tagID)

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

func TestService_AssignTagByName(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		tagName     string
		setupMock   func(*mocks.MockStore, string, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success assigns existing tag by name",
			githubID: "notif-1",
			tagName:  "bug",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: "10", Name: tagName, Slug: "bug"}
				updatedNotification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"10"},
				}

				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(tag, nil)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(updatedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "notif-1", notification.GithubID)
				require.Contains(t, notification.TagIDs, "10")
			},
		},
		{
			name:     "success creates and assigns new tag by name",
			githubID: "notif-1",
			tagName:  "new-tag",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				newTag := db.Tag{ID: "20", Name: tagName, Slug: "new-tag"}
				updatedNotification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"20"},
				}

				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(db.Tag{}, sql.ErrNoRows)
				m.EXPECT().
					UpsertTag(gomock.Any(), userID, gomock.Any()).
					Return(newTag, nil)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(updatedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "notif-1", notification.GithubID)
				require.Contains(t, notification.TagIDs, "20")
			},
		},
		{
			name:     "invalid tag name returns error",
			githubID: "notif-1",
			tagName:  "---",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(db.Tag{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping get tag by name failure",
			githubID: "notif-1",
			tagName:  "bug",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping create tag failure",
			githubID: "notif-1",
			tagName:  "new-tag",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(db.Tag{}, sql.ErrNoRows)
				dbError := errors.New("database error")
				m.EXPECT().
					UpsertTag(gomock.Any(), userID, gomock.Any()).
					Return(db.Tag{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping assign tag failure in AssignTagByName",
			githubID: "notif-1",
			tagName:  "bug",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: "10", Name: tagName, Slug: "bug"}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(tag, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping update tag IDs failure in AssignTagByName",
			githubID: "notif-1",
			tagName:  "bug",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				updatedNotification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"10"},
				}
				tag := db.Tag{ID: "10", Name: tagName, Slug: "bug"}
				// GetNotificationByGithubID is called twice: once at start, once at end
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil).
					Times(1)
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(updatedNotification, nil).
					Times(1)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(tag, nil)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "notif-1", notification.GithubID)
			},
		},
		{
			name:     "error wrapping get updated notification failure in AssignTagByName",
			githubID: "notif-1",
			tagName:  "bug",
			setupMock: func(m *mocks.MockStore, userID, githubID, tagName string) {
				notification := db.Notification{ID: 1, UserID: userID, GithubID: githubID}
				tag := db.Tag{ID: "10", Name: tagName, Slug: "bug"}
				// First call to get notification
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					GetTagByName(gomock.Any(), userID, tagName).
					Return(tag, nil)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
				// Second call to get updated notification fails
				dbError := errors.New("database error")
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(db.Notification{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubID, tt.tagName)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.AssignTagByName(ctx, testUserID, tt.githubID, tt.tagName)

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

func TestService_RemoveTag(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		tagID       string
		setupMock   func(*mocks.MockStore, string, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success removes tag from notification",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID, _ string) {
				notification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"10", "20"},
				}
				updatedNotification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"20"},
				}

				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), userID, gomock.Any()).
					Return(nil)
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(updatedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "notif-1", notification.GithubID)
				require.NotContains(t, notification.TagIDs, "10")
			},
		},
		{
			name:     "notification not found returns error",
			githubID: "not-found",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID string, _ string) {
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping remove tag assignment failure",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID string, _ string) {
				notification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"10", "20"},
				}
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				dbError := errors.New("database error")
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), userID, gomock.Any()).
					Return(dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error wrapping get updated notification failure in RemoveTag",
			githubID: "notif-1",
			tagID:    "10",
			setupMock: func(m *mocks.MockStore, userID, githubID string, _ string) {
				notification := db.Notification{
					ID:       1,
					UserID:   userID,
					GithubID: githubID,
					TagIDs:   []string{"10", "20"},
				}
				// First call to get notification
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(notification, nil)
				m.EXPECT().
					RemoveTagAssignment(gomock.Any(), userID, gomock.Any()).
					Return(nil)
				// Second call to get updated notification fails
				dbError := errors.New("database error")
				m.EXPECT().
					GetNotificationByGithubID(gomock.Any(), userID, githubID).
					Return(db.Notification{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubID, tt.tagID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.RemoveTag(ctx, testUserID, tt.githubID, tt.tagID)

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

func TestService_BulkAssignTag(t *testing.T) {
	tests := []struct {
		name          string
		notifications []db.Notification
		tagID         string
		setupMock     func(*mocks.MockStore, string, []db.Notification, string)
		expectErr     bool
		expectedCount int
		checkErr      func(*testing.T, error)
	}{
		{
			name: "success assigns tag to notifications",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, userID string, _ []db.Notification, _ string) {
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, gomock.Any()).
					Return(db.TagAssignment{}, nil)
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "duplicate key error still counts as success",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, _ string, _ []db.Notification, _ string) {
				// First call fails with duplicate key error
				duplicateError := errors.New("UNIQUE constraint failed: tag_assignments")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(db.TagAssignment{}, duplicateError)
			},
			expectErr:     false,
			expectedCount: 1, // Should count as success since tag was already assigned
		},
		{
			name: "non-duplicate error skips notification",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, _ string, _ []db.Notification, _ string) {
				// Non-duplicate error should skip
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(db.TagAssignment{}, errors.New("other database error"))
			},
			expectErr:     false,
			expectedCount: 0,
		},
		{
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, _ string, _ []db.Notification, _ string) {
				// First call fails with duplicate key error
				duplicateError := errors.New("UNIQUE constraint failed: tag_assignments")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(db.TagAssignment{}, duplicateError)
			},
			expectErr:     false,
			expectedCount: 1, // Duplicate key is treated as success
		},
		{
			name: "partial success with multiple notifications",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
				{ID: 2, GithubID: "notif-2"},
				{ID: 3, GithubID: "notif-3"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, userID string, _ []db.Notification, tagID string) {
				// First notification: success
				gomock.InOrder(
					m.EXPECT().
						AssignTagToEntity(gomock.Any(), userID, db.AssignTagToEntityParams{
							TagID:      tagID,
							EntityType: "notification",
							EntityID:   int64(1),
						}).
						Return(db.TagAssignment{}, nil),
				)
				duplicateError := errors.New("UNIQUE constraint failed: tag_assignments")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, db.AssignTagToEntityParams{
						TagID:      tagID,
						EntityType: "notification",
						EntityID:   int64(2),
					}).
					Return(db.TagAssignment{}, duplicateError)
				// Third notification: non-duplicate error, should skip
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, db.AssignTagToEntityParams{
						TagID:      tagID,
						EntityType: "notification",
						EntityID:   int64(3),
					}).
					Return(db.TagAssignment{}, errors.New("other error"))
			},
			expectErr:     false,
			expectedCount: 2, // First two succeed, third is skipped
		},
		{
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
				{ID: 2, GithubID: "notif-2"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, userID string, _ []db.Notification, tagID string) {
				// First notification: success
				gomock.InOrder(
					m.EXPECT().
						AssignTagToEntity(gomock.Any(), userID, db.AssignTagToEntityParams{
							TagID:      tagID,
							EntityType: "notification",
							EntityID:   int64(1),
						}).
						Return(db.TagAssignment{}, nil),
				)
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), userID, db.AssignTagToEntityParams{
						TagID:      tagID,
						EntityType: "notification",
						EntityID:   int64(2),
					}).
					Return(db.TagAssignment{}, nil)
			},
			expectErr:     false,
			expectedCount: 2, // Both succeed
		},
		{
			name: "UNIQUE constraint error variant",
			notifications: []db.Notification{
				{ID: 1, GithubID: "notif-1"},
			},
			tagID: "10",
			setupMock: func(m *mocks.MockStore, _ string, _ []db.Notification, _ string) {
				// Test UNIQUE constraint error (different from duplicate key)
				uniqueError := errors.New("UNIQUE constraint failed: tag_assignments")
				m.EXPECT().
					AssignTagToEntity(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(db.TagAssignment{}, uniqueError)
			},
			expectErr:     false,
			expectedCount: 1, // Should count as success
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			const testUserID = "test-user-id"
			tt.setupMock(mockQuerier, testUserID, tt.notifications, tt.tagID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			count, err := service.BulkAssignTag(ctx, testUserID, tt.notifications, tt.tagID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCount, count)
			}
		})
	}
}
