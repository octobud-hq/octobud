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
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	notificationmocks "github.com/octobud-hq/octobud/backend/internal/core/notification/mocks"
	tagmocks "github.com/octobud-hq/octobud/backend/internal/core/tag/mocks"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestHandler_handleBulkOperation(t *testing.T) {
	tests := []struct {
		name           string
		operation      BulkOperation
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "success with githubIDs",
			operation: BulkOpMarkRead,
			requestBody: bulkMarkNotificationsRequest{
				GithubIDs: []string{"id1", "id2"},
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpMarkRead, gomock.Any(), gomock.Any()).
					Return(int64(2), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 2, response.Count)
			},
		},
		{
			name:      "success with query",
			operation: BulkOpMarkRead,
			requestBody: bulkMarkNotificationsRequest{
				Query: "is:unread",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpMarkRead, gomock.Any(), gomock.Any()).
					Return(int64(5), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 5, response.Count)
			},
		},
		{
			name:           "invalid body returns 400",
			operation:      BulkOpMarkRead,
			requestBody:    "invalid json",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "empty githubIDs and empty query uses query path",
			operation: BulkOpMarkRead,
			requestBody: bulkMarkNotificationsRequest{
				GithubIDs: []string{},
				Query:     "",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				// Empty query and empty IDs is treated as query operation with empty query
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpMarkRead, gomock.Any(), gomock.Any()).
					Return(int64(0), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 0, response.Count)
			},
		},
		{
			name:      "both githubIDs and query returns 400",
			operation: BulkOpMarkRead,
			requestBody: bulkMarkNotificationsRequest{
				GithubIDs: []string{"id1"},
				Query:     "is:unread",
			},
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "no notification ids error returns 400",
			operation: BulkOpMarkRead,
			requestBody: bulkMarkNotificationsRequest{
				GithubIDs: []string{"id1"},
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpMarkRead, gomock.Any(), gomock.Any()).
					Return(int64(0), notification.ErrNoNotificationIDs)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "service error returns 500",
			operation: BulkOpMarkRead,
			requestBody: bulkMarkNotificationsRequest{
				GithubIDs: []string{"id1"},
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpMarkRead, gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database error"))
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
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodPost, "/notifications/bulk/mark-read", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()
			handler.handleBulkOperation(w, req, tt.operation)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleBulkAssignTag(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService, *tagmocks.MockTagService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success with githubIDs",
			requestBody: bulkTagNotificationsRequest{
				GithubIDs: []string{"id1", "id2"},
				TagID:     "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService, mockTagSvc *tagmocks.MockTagService) {
				mockTagSvc.EXPECT().
					GetTag(gomock.Any(), "test-user-id", "1").
					Return(db.Tag{ID: "1", Name: "test"}, nil)
				mockSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "id1").
					Return(db.Notification{GithubID: "id1"}, nil)
				mockSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "id2").
					Return(db.Notification{GithubID: "id2"}, nil)
				mockSvc.EXPECT().
					BulkAssignTag(gomock.Any(), "test-user-id", gomock.Any(), "1").
					Return(2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 2, response.Count)
			},
		},
		{
			name: "success with query",
			requestBody: bulkTagNotificationsRequest{
				Query: "is:unread",
				TagID: "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService, mockTagSvc *tagmocks.MockTagService) {
				mockTagSvc.EXPECT().
					GetTag(gomock.Any(), "test-user-id", "1").
					Return(db.Tag{ID: "1", Name: "test"}, nil)
				mockSvc.EXPECT().
					ListNotificationsFromQueryString(gomock.Any(), "test-user-id", "is:unread", int32(999999)).
					Return([]db.Notification{
						{GithubID: "id1"},
						{GithubID: "id2"},
					}, nil)
				mockSvc.EXPECT().
					BulkAssignTag(gomock.Any(), "test-user-id", gomock.Any(), "1").
					Return(2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 2, response.Count)
			},
		},
		{
			name:           "missing tagId returns 400",
			requestBody:    bulkTagNotificationsRequest{TagID: ""},
			setupMock:      func(*notificationmocks.MockNotificationService, *tagmocks.MockTagService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "tag not found returns 404",
			requestBody: bulkTagNotificationsRequest{TagID: "999", GithubIDs: []string{"id1"}},
			setupMock: func(_ *notificationmocks.MockNotificationService, mockTagSvc *tagmocks.MockTagService) {
				mockTagSvc.EXPECT().
					GetTag(gomock.Any(), "test-user-id", "999").
					Return(db.Tag{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "no notifications found returns 0 count",
			requestBody: bulkTagNotificationsRequest{
				GithubIDs: []string{"not-found"},
				TagID:     "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService, mockTagSvc *tagmocks.MockTagService) {
				mockTagSvc.EXPECT().
					GetTag(gomock.Any(), "test-user-id", "1").
					Return(db.Tag{ID: "1", Name: "test"}, nil)
				mockSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "not-found").
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 0, response.Count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockSvc, mockTagSvc, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()
			if tt.setupMock != nil {
				tt.setupMock(mockSvc, mockTagSvc)
			}

			req := createRequest(http.MethodPost, "/notifications/bulk/assign-tag", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()
			handler.handleBulkAssignTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleBulkRemoveTag(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success with githubIDs",
			requestBody: bulkTagNotificationsRequest{
				GithubIDs: []string{"id1", "id2"},
				TagID:     "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "id1").
					Return(db.Notification{GithubID: "id1"}, nil)
				mockSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "id2").
					Return(db.Notification{GithubID: "id2"}, nil)
				mockSvc.EXPECT().
					BulkRemoveTag(gomock.Any(), "test-user-id", gomock.Any(), "1").
					Return(2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 2, response.Count)
			},
		},
		{
			name: "success with query",
			requestBody: bulkTagNotificationsRequest{
				Query: "is:unread",
				TagID: "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					ListNotificationsFromQueryString(gomock.Any(), "test-user-id", "is:unread", int32(999999)).
					Return([]db.Notification{
						{GithubID: "id1"},
						{GithubID: "id2"},
					}, nil)
				mockSvc.EXPECT().
					BulkRemoveTag(gomock.Any(), "test-user-id", gomock.Any(), "1").
					Return(2, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 2, response.Count)
			},
		},
		{
			name:           "missing tagId returns 400",
			requestBody:    bulkTagNotificationsRequest{TagID: ""},
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "no notifications found returns 0 count",
			requestBody: bulkTagNotificationsRequest{
				GithubIDs: []string{"not-found"},
				TagID:     "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					GetByGithubID(gomock.Any(), "test-user-id", "not-found").
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response bulkNotificationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, 0, response.Count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockSvc, _, mockAuthSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodPost, "/notifications/bulk/remove-tag", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()
			handler.handleBulkRemoveTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}
