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
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestHandler_handleBulkSnoozeNotifications(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success with githubIDs",
			requestBody: bulkSnoozeNotificationsRequest{
				GithubIDs:    []string{"id1", "id2"},
				SnoozedUntil: "2024-12-31T23:59:59Z",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpSnooze, gomock.Any(), gomock.Any()).
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
			name: "success with query",
			requestBody: bulkSnoozeNotificationsRequest{
				Query:        "is:unread",
				SnoozedUntil: "2024-12-31T23:59:59Z",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpSnooze, gomock.Any(), gomock.Any()).
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
			name:           "missing snoozedUntil returns 400",
			requestBody:    bulkSnoozeNotificationsRequest{SnoozedUntil: ""},
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "no notification ids error returns 400",
			requestBody: bulkSnoozeNotificationsRequest{
				GithubIDs:    []string{"id1"},
				SnoozedUntil: "2024-12-31T23:59:59Z",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpSnooze, gomock.Any(), gomock.Any()).
					Return(int64(0), notification.ErrNoNotificationIDs)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error returns 500",
			requestBody: bulkSnoozeNotificationsRequest{
				GithubIDs:    []string{"id1"},
				SnoozedUntil: "2024-12-31T23:59:59Z",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					BulkUpdate(gomock.Any(), "test-user-id", models.BulkOpSnooze, gomock.Any(), gomock.Any()).
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

			req := createRequest(http.MethodPost, "/notifications/bulk/snooze", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()
			handler.handleBulkSnoozeNotifications(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}
