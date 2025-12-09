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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestService_BulkUpdate_MarkRead(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success with deduplication and sorting",
			githubIDs: []string{"notif-2", "notif-1", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				// Verify deduplication and sorting happens - should receive sorted, deduplicated list
				m.EXPECT().
					BulkMarkNotificationsRead(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpMarkRead,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_MarkNotificationsUnread(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success marks multiple notifications as unread",
			githubIDs: []string{"notif-2", "notif-1", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkMarkNotificationsUnread(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpMarkUnread,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_ArchiveNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success archives multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkArchiveNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpArchive,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_SnoozeNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name         string
		githubIDs    []string
		snoozedUntil string
		setupMock    func(*mocks.MockStore, string, []string, string)
		expectErr    bool
		checkErr     func(*testing.T, error)
		checkResult  func(*testing.T, int64)
	}{
		{
			name:         "success snoozes multiple notifications",
			githubIDs:    []string{"notif-2", "notif-1"},
			snoozedUntil: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			setupMock: func(m *mocks.MockStore, userID string, _ []string, _ string) {
				m.EXPECT().
					BulkSnoozeNotifications(gomock.Any(), userID, gomock.Any()).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:         "empty IDs returns error before DB call",
			githubIDs:    []string{},
			snoozedUntil: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			setupMock: func(_ *mocks.MockStore, _ string, _ []string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
		{
			name:         "invalid time format returns error before DB call",
			githubIDs:    []string{"notif-1"},
			snoozedUntil: "invalid-time",
			setupMock: func(_ *mocks.MockStore, _ string, _ []string, _ string) {
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
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs, tt.snoozedUntil)
			service := NewService(mockQuerier)

			ctx := context.Background()

			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpSnooze,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{SnoozedUntil: tt.snoozedUntil},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_UnarchiveNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success unarchives multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkUnarchiveNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnarchive,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_UnsnoozeNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success unsnoozes multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkUnsnoozeNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnsnooze,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_MuteNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success mutes multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkMuteNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpMute,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_UnmuteNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success unmutes multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkUnmuteNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnmute,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_StarNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success stars multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkStarNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpStar,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_UnstarNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success unstars multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkUnstarNotifications(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnstar,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_UnfilterNotifications(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		githubIDs   []string
		setupMock   func(*mocks.MockStore, string, []string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:      "success unfilters multiple notifications",
			githubIDs: []string{"notif-2", "notif-1"},
			setupMock: func(m *mocks.MockStore, userID string, _ []string) {
				m.EXPECT().
					BulkMarkNotificationsUnfiltered(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:      "empty IDs returns error before DB call",
			githubIDs: []string{},
			setupMock: func(_ *mocks.MockStore, _ string, _ []string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrNoNotificationIDs))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.githubIDs)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnfilter,
				models.BulkOperationTarget{IDs: tt.githubIDs},
				models.BulkUpdateParams{},
			)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
				require.Equal(t, int64(0), result)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_BulkUpdate_MarkNotificationsReadByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success marks notifications as read by query",
			queryStr: "is:unread",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkMarkNotificationsReadByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpMarkRead,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_MarkNotificationsUnreadByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success marks notifications as unread by query",
			queryStr: "is:read",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkMarkNotificationsUnreadByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpMarkUnread,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_ArchiveNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success archives notifications by query",
			queryStr: "is:unread",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkArchiveNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpArchive,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_UnarchiveNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success unarchives notifications by query",
			queryStr: "is:archived",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkUnarchiveNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnarchive,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_SnoozeNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name         string
		queryStr     string
		snoozedUntil string
		setupMock    func(*mocks.MockStore, string, string, string)
		expectErr    bool
		checkErr     func(*testing.T, error)
		checkResult  func(*testing.T, int64)
	}{
		{
			name:         "success snoozes notifications by query",
			queryStr:     "is:unread",
			snoozedUntil: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			setupMock: func(m *mocks.MockStore, userID string, _ string, _ string) {
				m.EXPECT().
					BulkSnoozeNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:         "invalid time format returns error before DB call",
			queryStr:     "is:unread",
			snoozedUntil: "invalid-time",
			setupMock: func(_ *mocks.MockStore, _ string, _ string, _ string) {
				// No mock expectations - should fail before DB call
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrInvalidSnoozedUntilFormat))
			},
		},
		{
			name:         "error wrapping query build failure",
			queryStr:     "invalid:query:format",
			snoozedUntil: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
			setupMock: func(_ *mocks.MockStore, _ string, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr, tt.snoozedUntil)
			service := NewService(mockQuerier)

			ctx := context.Background()

			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpSnooze,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{SnoozedUntil: tt.snoozedUntil},
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

func TestService_BulkUpdate_UnsnoozeNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success unsnoozes notifications by query",
			queryStr: "is:snoozed",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkUnsnoozeNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnsnooze,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_MuteNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success mutes notifications by query",
			queryStr: "is:unread",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkMuteNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpMute,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_UnmuteNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success unmutes notifications by query",
			queryStr: "is:muted",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkUnmuteNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnmute,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_StarNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success stars notifications by query",
			queryStr: "is:unread",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkStarNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpStar,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_UnstarNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success unstars notifications by query",
			queryStr: "is:starred",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					BulkUnstarNotificationsByQuery(gomock.Any(), userID, gomock.Any()).
					Return(int64(1), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(1), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnstar,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestService_BulkUpdate_UnfilterNotificationsByQuery(t *testing.T) {
	const testUserID = "test-user-id"
	tests := []struct {
		name        string
		queryStr    string
		setupMock   func(*mocks.MockStore, string, string)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, int64)
	}{
		{
			name:     "success unfilters notifications by query",
			queryStr: "is:filtered",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				// First call to ListNotificationsFromQuery to get matching notifications
				now := time.Now().UTC()
				notifications := []db.Notification{
					{ID: 1, UserID: userID, GithubID: "notif-1", Filtered: true, ImportedAt: now},
					{ID: 2, UserID: userID, GithubID: "notif-2", Filtered: true, ImportedAt: now},
				}
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), userID, gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{
						Notifications: notifications,
						Total:         2,
					}, nil)
				// Then call to models.BulkUnfilterNotifications
				m.EXPECT().
					BulkMarkNotificationsUnfiltered(gomock.Any(), userID, []string{"notif-1", "notif-2"}).
					Return(int64(2), nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(2), count)
			},
		},
		{
			name:     "empty result returns empty slice",
			queryStr: "is:filtered",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), userID, gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{
						Notifications: []db.Notification{},
						Total:         0,
					}, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, count int64) {
				require.Equal(t, int64(0), count)
			},
		},
		{
			name:     "error wrapping query build failure",
			queryStr: "invalid:query:format",
			setupMock: func(_ *mocks.MockStore, _ string, _ string) {
				// No mock expectations - should fail at query build stage
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToBuildQuery))
			},
		},
		{
			name:     "error wrapping list notifications failure",
			queryStr: "is:filtered",
			setupMock: func(m *mocks.MockStore, userID string, _ string) {
				dbError := errors.New("database error")
				m.EXPECT().
					ListNotificationsFromQuery(gomock.Any(), userID, gomock.Any()).
					Return(db.ListNotificationsFromQueryResult{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToListNotifications))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, testUserID, tt.queryStr)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.BulkUpdate(
				ctx,
				testUserID,
				models.BulkOpUnfilter,
				models.BulkOperationTarget{Query: tt.queryStr},
				models.BulkUpdateParams{},
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

func TestDedupeAndSort_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "mixed duplicates and whitespace",
			input:    []string{" a ", "b", " a", "  ", "c", "b ", ""},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all empty",
			input:    []string{"", "  ", "   "},
			expected: []string{},
		},
		{
			name:     "case sensitive",
			input:    []string{"A", "a", "B", "b"},
			expected: []string{"A", "B", "a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupeAndSort(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
