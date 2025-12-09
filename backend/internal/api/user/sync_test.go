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

package user

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	authmocks "github.com/octobud-hq/octobud/backend/internal/core/auth/mocks"
	syncstatemocks "github.com/octobud-hq/octobud/backend/internal/core/syncstate/mocks"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// mockScheduler is a simple mock for jobs.Scheduler
type mockScheduler struct {
	enqueueSyncOlderFn func(ctx context.Context, args jobs.SyncOlderNotificationsArgs) error
}

func (m *mockScheduler) Start(_ context.Context) error { return nil }
func (m *mockScheduler) Stop(_ context.Context) error  { return nil }
func (m *mockScheduler) EnqueueSyncNotifications(_ context.Context, _ string) error {
	return nil
}
func (m *mockScheduler) EnqueueProcessNotification(_ context.Context, _ string, _ []byte) error {
	return nil
}
func (m *mockScheduler) EnqueueApplyRule(_ context.Context, _, _ string) error { return nil }

func (m *mockScheduler) EnqueueSyncOlder(
	ctx context.Context,
	args jobs.SyncOlderNotificationsArgs,
) error {
	if m.enqueueSyncOlderFn != nil {
		return m.enqueueSyncOlderFn(ctx, args)
	}
	return nil
}

func setupTestHandler(ctrl *gomock.Controller) (*Handler, *authmocks.MockAuthService) {
	logger := zap.NewNop()
	mockService := authmocks.NewMockAuthService(ctrl)
	return New(logger, mockService), mockService
}

func createRequest(method, url string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			reqBody = nil
		}
	}
	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

type errorResponse struct {
	Error string `json:"error"`
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func TestHandler_HandleGetSyncSettings(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*authmocks.MockAuthService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns sync settings",
			setupMock: func(m *authmocks.MockAuthService) {
				days := 30
				maxCount := 1000
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(&models.SyncSettings{
					InitialSyncDays:       &days,
					InitialSyncMaxCount:   &maxCount,
					InitialSyncUnreadOnly: true,
					SetupCompleted:        true,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncSettingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotNil(t, response.InitialSyncDays)
				require.Equal(t, 30, *response.InitialSyncDays)
				require.NotNil(t, response.InitialSyncMaxCount)
				require.Equal(t, 1000, *response.InitialSyncMaxCount)
				require.True(t, response.InitialSyncUnreadOnly)
				require.True(t, response.SetupCompleted)
			},
		},
		{
			name: "returns empty response when no settings exist",
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncSettingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Nil(t, response.InitialSyncDays)
				require.Nil(t, response.InitialSyncMaxCount)
				require.False(t, response.InitialSyncUnreadOnly)
				require.False(t, response.SetupCompleted)
			},
		},
		{
			name: "service error returns 500",
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().
					GetUserSyncSettings(gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockService := setupTestHandler(ctrl)
			mockService.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: "test-user-id"}, nil).
				AnyTimes()
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			req := createRequest(http.MethodGet, "/api/user/sync-settings", nil)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), "test-user-id"))
			w := httptest.NewRecorder()

			handler.HandleGetSyncSettings(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleUpdateSyncSettings(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*authmocks.MockAuthService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success updates all settings",
			requestBody: SyncSettingsRequest{
				InitialSyncDays:       intPtr(30),
				InitialSyncMaxCount:   intPtr(1000),
				InitialSyncUnreadOnly: true,
				SetupCompleted:        false,
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
				m.EXPECT().UpdateUserSyncSettings(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, settings *models.SyncSettings) error {
						require.NotNil(t, settings.InitialSyncDays)
						require.Equal(t, 30, *settings.InitialSyncDays)
						require.NotNil(t, settings.InitialSyncMaxCount)
						require.Equal(t, 1000, *settings.InitialSyncMaxCount)
						require.True(t, settings.InitialSyncUnreadOnly)
						require.False(t, settings.SetupCompleted)
						return nil
					})
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncSettingsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotNil(t, response.InitialSyncDays)
				require.Equal(t, 30, *response.InitialSyncDays)
				require.NotNil(t, response.InitialSyncMaxCount)
				require.Equal(t, 1000, *response.InitialSyncMaxCount)
				require.True(t, response.InitialSyncUnreadOnly)
			},
		},
		{
			name: "success with nil days (all-time sync)",
			requestBody: SyncSettingsRequest{
				InitialSyncDays:       nil,
				InitialSyncMaxCount:   intPtr(500),
				InitialSyncUnreadOnly: false,
				SetupCompleted:        true,
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
				m.EXPECT().UpdateUserSyncSettings(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, settings *models.SyncSettings) error {
						require.Nil(t, settings.InitialSyncDays)
						return nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "initialSyncDays less than 1 returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(0),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncDays must be at least 1")
			},
		},
		{
			name: "initialSyncDays exceeds max returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(3651),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncDays cannot exceed 3650")
			},
		},
		{
			name: "initialSyncMaxCount less than 1 returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncMaxCount: intPtr(0),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncMaxCount must be at least 1")
			},
		},
		{
			name: "initialSyncMaxCount exceeds max returns 400",
			requestBody: SyncSettingsRequest{
				InitialSyncMaxCount: intPtr(100001),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "initialSyncMaxCount cannot exceed 100000")
			},
		},
		{
			name: "service error returns 500",
			requestBody: SyncSettingsRequest{
				InitialSyncDays: intPtr(30),
			},
			setupMock: func(m *authmocks.MockAuthService) {
				m.EXPECT().GetUserSyncSettings(gomock.Any()).Return(nil, nil)
				m.EXPECT().
					UpdateUserSyncSettings(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockService := setupTestHandler(ctrl)
			mockService.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: "test-user-id"}, nil).
				AnyTimes()
			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			req := createRequest(http.MethodPut, "/api/user/sync-settings", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), "test-user-id"))
			w := httptest.NewRecorder()

			handler.HandleUpdateSyncSettings(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleGetSyncState(t *testing.T) {
	tests := []struct {
		name           string
		setupHandler   func(*Handler, *gomock.Controller)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns sync state with timestamps",
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				completedTime := time.Date(2024, 1, 20, 12, 0, 0, 0, time.UTC)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
						InitialSyncCompletedAt:     sql.NullTime{Time: completedTime, Valid: true},
					}, nil)
				h.syncStateSvc = mockSyncState
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncStateResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.NotNil(t, response.OldestNotificationSyncedAt)
				require.NotNil(t, response.InitialSyncCompletedAt)
				require.Contains(t, *response.OldestNotificationSyncedAt, "2024-01-15")
				require.Contains(t, *response.InitialSyncCompletedAt, "2024-01-20")
			},
		},
		{
			name: "success returns empty response when no timestamps",
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{}, nil)
				h.syncStateSvc = mockSyncState
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response SyncStateResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Nil(t, response.OldestNotificationSyncedAt)
				require.Nil(t, response.InitialSyncCompletedAt)
			},
		},
		{
			name: "sync state service unavailable returns 503",
			setupHandler: func(h *Handler, _ *gomock.Controller) {
				h.syncStateSvc = nil
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "service error returns 500",
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{}, errors.New("database error"))
				h.syncStateSvc = mockSyncState
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: "test-user-id"}, nil).
				AnyTimes()
			if tt.setupHandler != nil {
				tt.setupHandler(handler, ctrl)
			}

			req := createRequest(http.MethodGet, "/api/user/sync-state", nil)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), "test-user-id"))
			w := httptest.NewRecorder()

			handler.HandleGetSyncState(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_HandleSyncOlder(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupHandler   func(*Handler, *gomock.Controller)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success queues sync older job",
			requestBody: SyncOlderRequest{
				Days:       30,
				MaxCount:   intPtr(100),
				UnreadOnly: false,
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
					}, nil)
				h.syncStateSvc = mockSyncState

				h.scheduler = &mockScheduler{
					enqueueSyncOlderFn: func(_ context.Context, args jobs.SyncOlderNotificationsArgs) error {
						require.Equal(t, 30, args.Days)
						require.Equal(t, oldestTime, args.UntilTime)
						require.NotNil(t, args.MaxCount)
						require.Equal(t, 100, *args.MaxCount)
						require.False(t, args.UnreadOnly)
						return nil
					},
				}
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "success with beforeDate override",
			requestBody: SyncOlderRequest{
				Days:       30,
				BeforeDate: stringPtr("2024-02-01T12:00:00Z"),
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				// No sync state query needed when using beforeDate override
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				h.syncStateSvc = mockSyncState

				expectedUntilTime := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
				h.scheduler = &mockScheduler{
					enqueueSyncOlderFn: func(_ context.Context, args jobs.SyncOlderNotificationsArgs) error {
						require.Equal(t, 30, args.Days)
						require.Equal(t, expectedUntilTime, args.UntilTime)
						return nil
					},
				}
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "invalid beforeDate format returns 400",
			requestBody: SyncOlderRequest{
				Days:       30,
				BeforeDate: stringPtr("invalid-date"),
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				h.syncStateSvc = mockSyncState
				h.scheduler = &mockScheduler{}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response["error"], "RFC3339")
			},
		},
		{
			name: "success with minimal request",
			requestBody: SyncOlderRequest{
				Days: 60,
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
					}, nil)
				h.syncStateSvc = mockSyncState
				h.scheduler = &mockScheduler{}
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "invalid request body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "days less than 1 returns 400",
			requestBody: SyncOlderRequest{
				Days: 0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "days must be at least 1")
			},
		},
		{
			name: "days exceeds max returns 400",
			requestBody: SyncOlderRequest{
				Days: 3651,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "days cannot exceed 3650")
			},
		},
		{
			name: "maxCount less than 1 returns 400",
			requestBody: SyncOlderRequest{
				Days:     30,
				MaxCount: intPtr(0),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "maxCount must be at least 1")
			},
		},
		{
			name: "maxCount exceeds max returns 400",
			requestBody: SyncOlderRequest{
				Days:     30,
				MaxCount: intPtr(100001),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "maxCount cannot exceed 100000")
			},
		},
		{
			name: "sync state service unavailable returns 503",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupHandler: func(h *Handler, _ *gomock.Controller) {
				h.syncStateSvc = nil
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "scheduler unavailable returns 503",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				h.syncStateSvc = mockSyncState
				h.scheduler = nil
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "no oldest notification returns 400",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						OldestNotificationSyncedAt: sql.NullTime{Valid: false},
					}, nil)
				h.syncStateSvc = mockSyncState
				h.scheduler = &mockScheduler{}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "No notifications have been synced yet")
			},
		},
		{
			name: "sync state error returns 500",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{}, errors.New("database error"))
				h.syncStateSvc = mockSyncState
				h.scheduler = &mockScheduler{}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "job queue error returns 500",
			requestBody: SyncOlderRequest{
				Days: 30,
			},
			setupHandler: func(h *Handler, ctrl *gomock.Controller) {
				oldestTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				mockSyncState := syncstatemocks.NewMockSyncStateService(ctrl)
				mockSyncState.EXPECT().
					GetSyncState(gomock.Any(), "test-user-id").
					Return(models.SyncState{
						OldestNotificationSyncedAt: sql.NullTime{Time: oldestTime, Valid: true},
					}, nil)
				h.syncStateSvc = mockSyncState
				h.scheduler = &mockScheduler{
					enqueueSyncOlderFn: func(_ context.Context, _ jobs.SyncOlderNotificationsArgs) error {
						return errors.New("queue error")
					},
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: "test-user-id"}, nil).
				AnyTimes()
			if tt.setupHandler != nil {
				tt.setupHandler(handler, ctrl)
			}

			req := createRequest(http.MethodPost, "/api/user/sync-older", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), "test-user-id"))
			w := httptest.NewRecorder()

			handler.HandleSyncOlder(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}
