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
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
)

func TestService_MarkNotificationRead(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success marks notification as read",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					SubjectType:  "PullRequest",
					SubjectTitle: "Update README",
					IsRead:       true,
					ImportedAt:   now,
				}
				m.EXPECT().
					MarkNotificationRead(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.True(t, notification.IsRead)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					MarkNotificationRead(gomock.Any(), userID, id).
					Return(db.Notification{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				// Verify error is propagated (service just passes through for simple operations)
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.MarkNotificationRead(ctx, testUserID, tt.githubID)

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

func TestService_MarkNotificationUnread(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success marks notification as unread",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					IsRead:       false,
					ImportedAt:   now,
				}
				m.EXPECT().
					MarkNotificationUnread(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.False(t, notification.IsRead)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					MarkNotificationUnread(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.MarkNotificationUnread(ctx, testUserID, tt.githubID)

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

func TestService_ArchiveNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success archives notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Archived:     true,
					ImportedAt:   now,
				}
				m.EXPECT().
					ArchiveNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.True(t, notification.Archived)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					ArchiveNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ArchiveNotification(ctx, testUserID, tt.githubID)

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

func TestService_UnarchiveNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success unarchives notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Archived:     false,
					ImportedAt:   now,
				}
				m.EXPECT().
					UnarchiveNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.False(t, notification.Archived)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					UnarchiveNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UnarchiveNotification(ctx, testUserID, tt.githubID)

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

func TestService_MuteNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success mutes notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Muted:        true,
					ImportedAt:   now,
				}
				m.EXPECT().
					MuteNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.True(t, notification.Muted)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					MuteNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.MuteNotification(ctx, testUserID, tt.githubID)

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

func TestService_UnmuteNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success unmutes notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Muted:        false,
					ImportedAt:   now,
				}
				m.EXPECT().
					UnmuteNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.False(t, notification.Muted)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					UnmuteNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UnmuteNotification(ctx, testUserID, tt.githubID)

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

func TestService_UnsnoozeNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success unsnoozes notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					SnoozedUntil: sql.NullTime{Valid: false},
					ImportedAt:   now,
				}
				m.EXPECT().
					UnsnoozeNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.False(t, notification.SnoozedUntil.Valid)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					UnsnoozeNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UnsnoozeNotification(ctx, testUserID, tt.githubID)

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

func TestService_StarNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success stars notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Starred:      true,
					ImportedAt:   now,
				}
				m.EXPECT().
					StarNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.True(t, notification.Starred)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					StarNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.StarNotification(ctx, testUserID, tt.githubID)

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

func TestService_UnstarNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success unstars notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Starred:      false,
					ImportedAt:   now,
				}
				m.EXPECT().
					UnstarNotification(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.False(t, notification.Starred)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					UnstarNotification(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UnstarNotification(ctx, testUserID, tt.githubID)

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

func TestService_UnfilterNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubID    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Notification)
	}{
		{
			name:     "success unfilters notification",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				now := time.Now().UTC()
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					RepositoryID: 10,
					Filtered:     false,
					ImportedAt:   now,
				}
				m.EXPECT().
					MarkNotificationUnfiltered(gomock.Any(), userID, id).
					Return(expectedNotification, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "abc", notification.GithubID)
				require.False(t, notification.Filtered)
			},
		},
		{
			name:     "error wrapping database failure",
			githubID: "abc",
			setupMock: func(m *mocks.MockStore, userID, id string) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					MarkNotificationUnfiltered(gomock.Any(), userID, id).
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
			tt.setupMock(mockQuerier, testUserID, tt.githubID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.UnfilterNotification(ctx, testUserID, tt.githubID)

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

func TestService_SnoozeNotification(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name         string
		githubID     string
		snoozedUntil string
		setupMock    func(*mocks.MockStore, string, string, string)
		expectErr    bool
		checkErr     func(*testing.T, error)
		checkResult  func(*testing.T, db.Notification)
	}{
		{
			name:         "success snoozes notification",
			githubID:     "notif-1",
			snoozedUntil: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			setupMock: func(m *mocks.MockStore, userID, id string, untilStr string) {
				now := time.Now().UTC()

				snoozeUntil, _ := time.Parse(time.RFC3339, untilStr)
				expectedNotification := db.Notification{
					ID:           1,
					UserID:       userID,
					GithubID:     id,
					SnoozedUntil: sql.NullTime{Time: snoozeUntil, Valid: true},
					ImportedAt:   now,
				}
				m.EXPECT().
					SnoozeNotification(gomock.Any(), userID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, arg db.SnoozeNotificationParams) (db.Notification, error) {
						require.Equal(t, id, arg.GithubID)
						require.True(t, arg.SnoozedUntil.Valid)
						return expectedNotification, nil
					})
			},
			expectErr: false,
			checkResult: func(t *testing.T, notification db.Notification) {
				require.Equal(t, "notif-1", notification.GithubID)
				require.True(t, notification.SnoozedUntil.Valid)
			},
		},
		{
			name:         "invalid time format returns error before DB call",
			githubID:     "notif-1",
			snoozedUntil: "invalid-time",
			setupMock: func(_ *mocks.MockStore, _ string, _ string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrInvalidSnoozedUntilFormat))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubID, tt.snoozedUntil)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.SnoozeNotification(ctx, testUserID, tt.githubID, tt.snoozedUntil)

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
