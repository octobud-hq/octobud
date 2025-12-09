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

package syncstate

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
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestService_GetSyncState(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, models.SyncState)
	}{
		{
			name: "success returns sync state",
			setupMock: func(m *mocks.MockStore) {
				now := time.Now().UTC()
				expectedState := db.GetSyncStateRow{
					ID:                   1,
					LastSuccessfulPoll:   sql.NullTime{Time: now, Valid: true},
					LatestNotificationAt: sql.NullTime{Time: now, Valid: true},
					UpdatedAt:            now,
				}
				m.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(expectedState, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.True(t, state.LastSuccessfulPoll.Valid)
				require.True(t, state.LatestNotificationAt.Valid)
			},
		},
		{
			name: "no rows returns empty state without error",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(db.GetSyncStateRow{}, sql.ErrNoRows)
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.False(t, state.LastSuccessfulPoll.Valid)
				require.False(t, state.LatestNotificationAt.Valid)
			},
		},
		{
			name: "error wrapping database failure",
			setupMock: func(m *mocks.MockStore) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(db.GetSyncStateRow{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToGetSyncState))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewSyncStateService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetSyncState(ctx, "test-user-id")

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

func TestService_UpsertSyncState(t *testing.T) {
	tests := []struct {
		name                 string
		lastSuccessfulPoll   *time.Time
		latestNotificationAt *time.Time
		setupMock            func(*mocks.MockStore)
		expectErr            bool
		checkResult          func(*testing.T, models.SyncState)
	}{
		{
			name:                 "success with both timestamps",
			lastSuccessfulPoll:   timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			latestNotificationAt: timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertSyncState(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, params db.UpsertSyncStateParams) (db.UpsertSyncStateRow, error) {
						require.True(t, params.LastSuccessfulPoll.Valid)
						require.True(t, params.LatestNotificationAt.Valid)
						return db.UpsertSyncStateRow{
							ID:                   1,
							LastSuccessfulPoll:   params.LastSuccessfulPoll,
							LatestNotificationAt: params.LatestNotificationAt,
							UpdatedAt:            time.Now(),
						}, nil
					})
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.True(t, state.LastSuccessfulPoll.Valid)
				require.True(t, state.LatestNotificationAt.Valid)
			},
		},
		{
			name:                 "success with nil latest notification",
			lastSuccessfulPoll:   timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			latestNotificationAt: nil,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertSyncState(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, params db.UpsertSyncStateParams) (db.UpsertSyncStateRow, error) {
						require.True(t, params.LastSuccessfulPoll.Valid)
						require.False(t, params.LatestNotificationAt.Valid)
						return db.UpsertSyncStateRow{
							ID:                   1,
							LastSuccessfulPoll:   params.LastSuccessfulPoll,
							LatestNotificationAt: params.LatestNotificationAt,
							UpdatedAt:            time.Now(),
						}, nil
					})
			},
			expectErr: false,
		},
		{
			name:                 "database error",
			lastSuccessfulPoll:   timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			latestNotificationAt: nil,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertSyncState(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.UpsertSyncStateRow{}, errors.New("database error"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewSyncStateService(mockQuerier)

			ctx := context.Background()
			result, err := service.UpsertSyncState(
				ctx,
				"test-user-id",
				tt.lastSuccessfulPoll,
				tt.latestNotificationAt,
			)

			if tt.expectErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrFailedToUpdateSyncState))
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_UpsertSyncStateWithInitialSync(t *testing.T) {
	tests := []struct {
		name                       string
		lastSuccessfulPoll         *time.Time
		latestNotificationAt       *time.Time
		initialSyncCompletedAt     *time.Time
		oldestNotificationSyncedAt *time.Time
		setupMock                  func(*mocks.MockStore)
		expectErr                  bool
		checkResult                func(*testing.T, models.SyncState)
	}{
		{
			name:                       "success with initial sync completed",
			lastSuccessfulPoll:         timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			latestNotificationAt:       timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
			initialSyncCompletedAt:     timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			oldestNotificationSyncedAt: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertSyncState(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, params db.UpsertSyncStateParams) (db.UpsertSyncStateRow, error) {
						require.True(t, params.LastSuccessfulPoll.Valid)
						require.True(t, params.LatestNotificationAt.Valid)
						require.True(t, params.InitialSyncCompletedAt.Valid)
						require.True(t, params.OldestNotificationSyncedAt.Valid)
						return db.UpsertSyncStateRow{
							ID:                         1,
							LastSuccessfulPoll:         params.LastSuccessfulPoll,
							LatestNotificationAt:       params.LatestNotificationAt,
							InitialSyncCompletedAt:     params.InitialSyncCompletedAt,
							OldestNotificationSyncedAt: params.OldestNotificationSyncedAt,
							UpdatedAt:                  time.Now(),
						}, nil
					})
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.True(t, state.LastSuccessfulPoll.Valid)
				require.True(t, state.LatestNotificationAt.Valid)
				require.True(t, state.InitialSyncCompletedAt.Valid)
				require.True(t, state.OldestNotificationSyncedAt.Valid)
			},
		},
		{
			name:                       "success with nil initial sync fields",
			lastSuccessfulPoll:         timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			latestNotificationAt:       timePtr(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)),
			initialSyncCompletedAt:     nil,
			oldestNotificationSyncedAt: nil,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertSyncState(gomock.Any(), "test-user-id", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, params db.UpsertSyncStateParams) (db.UpsertSyncStateRow, error) {
						require.True(t, params.LastSuccessfulPoll.Valid)
						require.True(t, params.LatestNotificationAt.Valid)
						require.False(t, params.InitialSyncCompletedAt.Valid)
						require.False(t, params.OldestNotificationSyncedAt.Valid)
						return db.UpsertSyncStateRow{
							ID:                   1,
							LastSuccessfulPoll:   params.LastSuccessfulPoll,
							LatestNotificationAt: params.LatestNotificationAt,
							UpdatedAt:            time.Now(),
						}, nil
					})
			},
			expectErr: false,
			checkResult: func(t *testing.T, state models.SyncState) {
				require.True(t, state.LastSuccessfulPoll.Valid)
				require.True(t, state.LatestNotificationAt.Valid)
				require.False(t, state.InitialSyncCompletedAt.Valid)
				require.False(t, state.OldestNotificationSyncedAt.Valid)
			},
		},
		{
			name:                       "database error",
			lastSuccessfulPoll:         timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			latestNotificationAt:       nil,
			initialSyncCompletedAt:     timePtr(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)),
			oldestNotificationSyncedAt: nil,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					UpsertSyncState(gomock.Any(), "test-user-id", gomock.Any()).
					Return(db.UpsertSyncStateRow{}, errors.New("database error"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewSyncStateService(mockQuerier)

			ctx := context.Background()
			result, err := service.UpsertSyncStateWithInitialSync(
				ctx,
				"test-user-id",
				tt.lastSuccessfulPoll,
				tt.latestNotificationAt,
				tt.initialSyncCompletedAt,
				tt.oldestNotificationSyncedAt,
			)

			if tt.expectErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrFailedToUpdateSyncState))
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
