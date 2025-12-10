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

package notifications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	authmocks "github.com/octobud-hq/octobud/backend/internal/core/auth/mocks"
	notificationmocks "github.com/octobud-hq/octobud/backend/internal/core/notification/mocks"
	repositorymocks "github.com/octobud-hq/octobud/backend/internal/core/repository/mocks"
	tagmocks "github.com/octobud-hq/octobud/backend/internal/core/tag/mocks"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	syncmocks "github.com/octobud-hq/octobud/backend/internal/sync/mocks"
)

func setupTestHandler(
	ctrl *gomock.Controller,
) (*Handler, *notificationmocks.MockNotificationService, *tagmocks.MockTagService, *authmocks.MockAuthService) {
	logger := zap.NewNop()
	mockNotificationSvc := notificationmocks.NewMockNotificationService(ctrl)
	mockRepositorySvc := repositorymocks.NewMockRepositoryService(ctrl)
	mockTagSvc := tagmocks.NewMockTagService(ctrl)
	mockAuthSvc := authmocks.NewMockAuthService(ctrl)
	mockSyncService := syncmocks.NewMockSyncOperations(ctrl)

	handler := &Handler{
		logger:        logger,
		store:         nil, // Not used in handler methods
		notifications: mockNotificationSvc,
		repositorySvc: mockRepositorySvc,
		tagSvc:        mockTagSvc,
		timelineSvc:   nil, // Set in individual tests if needed
		githubClient:  nil, // Set in individual tests if needed
		syncService:   mockSyncService,
		scheduler:     nil, // Set in individual tests if needed
		authSvc:       mockAuthSvc,
	}

	return handler, mockNotificationSvc, mockTagSvc, mockAuthSvc
}

func createRequest(method, requestURL string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, requestURL, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandler_handleListNotifications(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "success returns list of notifications",
			queryParams: map[string]string{"page": "1", "pageSize": "10"},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					ListNotifications(gomock.Any(), "test-user-id", gomock.Any()).
					Return(models.ListDetailsResult{
						Notifications: []models.Notification{
							{ID: 1, GithubID: "test-1", SubjectTitle: "Test Notification"},
						},
						Total:    1,
						Page:     1,
						PageSize: 10,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, int64(1), response.Total)
				require.Len(t, response.Notifications, 1)
			},
		},
		{
			name:        "invalid query parameters",
			queryParams: map[string]string{"page": "invalid"},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				// parseNotificationListOptions doesn't return errors for invalid params, so handler still calls service
				mockSvc.EXPECT().
					ListNotifications(gomock.Any(), "test-user-id", gomock.Any()).
					Return(models.ListDetailsResult{
						Notifications: []models.Notification{},
						Total:         0,
						Page:          0,
						PageSize:      0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
			},
		},
		{
			name:        "service error returns 400",
			queryParams: map[string]string{},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					ListNotifications(gomock.Any(), "test-user-id", gomock.Any()).
					Return(models.ListDetailsResult{}, errors.New("database error"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "database error", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockSvc, _, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			req := createRequest(http.MethodGet, "/notifications", nil)
			if len(tt.queryParams) > 0 {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}
			req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleListNotifications(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleGetNotification(t *testing.T) {
	tests := []struct {
		name           string
		githubID       string
		queryStr       string
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success returns notification",
			githubID: "test-id",
			queryStr: "",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{ID: 1, GithubID: "test-id", SubjectTitle: "Test"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response notificationDetailResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "test-id", response.Notification.GithubID)
			},
		},
		{
			name:           "missing githubID returns 400",
			githubID:       "",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid githubID encoding returns 400",
			githubID:       "%",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "not found returns 404",
			githubID: "not-found",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "not-found", "").
					Return(models.Notification{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "service error returns 500",
			githubID: "test-id",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockSvc, _, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			req := createRequest(http.MethodGet, "/notifications/"+url.PathEscape(tt.githubID), nil)
			if tt.queryStr != "" {
				req.URL.RawQuery = "query=" + url.QueryEscape(tt.queryStr)
			}

			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleGetNotification(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

// TestHandler_handleRefreshNotificationSubject_Basic tests basic endpoint behavior
// for the refresh-subject endpoint. Full testing including rule re-application
// logic is tested in the ApplyRulesToNotificationHandler tests and RefreshSubjectData tests.
func TestHandler_handleRefreshNotificationSubject_Basic(t *testing.T) {
	tests := []struct {
		name               string
		githubID           string
		setupMocks         func(*notificationmocks.MockNotificationService)
		excludeSyncService bool
		expectedStatus     int
	}{
		{
			name:               "sync service unavailable returns 503",
			githubID:           "notif-123",
			excludeSyncService: true,
			setupMocks: func(mockNotifSvc *notificationmocks.MockNotificationService) {
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:     "notification not found returns 404",
			githubID: "not-found",
			setupMocks: func(mockNotifSvc *notificationmocks.MockNotificationService) {
				// GetByGithubID is called first to verify notification exists (before refreshSubjectData)
				mockNotifSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "not-found").
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockNotifSvc, _, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			// syncService is set by default in setupTestHandler as a mock
			// Override to nil if excludeSyncService is true
			if tt.excludeSyncService {
				handler.syncService = nil
				handler.scheduler = nil
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockNotifSvc)
			}

			req := createRequest(
				http.MethodPost,
				"/notifications/"+url.PathEscape(tt.githubID)+"/refresh-subject",
				nil,
			)

			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleRefreshNotificationSubject(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
