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

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	notificationcore "github.com/octobud-hq/octobud/backend/internal/core/notification"
	notificationmocks "github.com/octobud-hq/octobud/backend/internal/core/notification/mocks"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestHandler_handleNotificationAction(t *testing.T) {
	tests := []struct {
		name           string
		githubID       string
		action         NotificationAction
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success marks notification as read",
			githubID: "test-id",
			action:   ActionMarkRead,
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					MarkNotificationRead(gomock.Any(), "test-user-id", "test-id").
					Return(db.Notification{}, nil)
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{ID: 1, GithubID: "test-id"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response notificationActionResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "test-id", response.Notification.GithubID)
			},
		},
		{
			name:           "missing githubID returns 400",
			githubID:       "",
			action:         ActionMarkRead,
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "not found returns 404",
			githubID: "not-found",
			action:   ActionMarkRead,
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					MarkNotificationRead(gomock.Any(), "test-user-id", "not-found").
					Return(db.Notification{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "service error returns 500",
			githubID: "test-id",
			action:   ActionMarkRead,
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					MarkNotificationRead(gomock.Any(), "test-user-id", "test-id").
					Return(db.Notification{}, errors.New("database error"))
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
			// Set up auth service to return test user
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(
				http.MethodPost,
				"/notifications/"+url.PathEscape(tt.githubID),
				nil,
			)
			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleNotificationAction(w, req, tt.action)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleSnoozeNotification(t *testing.T) {
	tests := []struct {
		name           string
		githubID       string
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success snoozes notification",
			githubID: "test-id",
			requestBody: snoozeNotificationRequest{
				SnoozedUntil: "2024-12-31T23:59:59Z",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					SnoozeNotification(gomock.Any(), "test-user-id", "test-id", "2024-12-31T23:59:59Z").
					Return(db.Notification{}, nil)
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{ID: 1, GithubID: "test-id"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response notificationActionResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "test-id", response.Notification.GithubID)
			},
		},
		{
			name:           "missing snoozedUntil returns 400",
			githubID:       "test-id",
			requestBody:    snoozeNotificationRequest{SnoozedUntil: ""},
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body returns 400",
			githubID:       "test-id",
			requestBody:    "invalid json",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "not found returns 404",
			githubID: "not-found",
			requestBody: snoozeNotificationRequest{
				SnoozedUntil: "2024-12-31T23:59:59Z",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					SnoozeNotification(gomock.Any(), "test-user-id", "not-found", "2024-12-31T23:59:59Z").
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
			handler, mockSvc, _, mockAuthSvc := setupTestHandler(ctrl)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			req := createRequest(
				http.MethodPost,
				"/notifications/"+url.PathEscape(tt.githubID),
				tt.requestBody,
			)
			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleSnoozeNotification(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleAssignTagToNotification(t *testing.T) {
	tests := []struct {
		name           string
		githubID       string
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success assigns tag",
			githubID: "test-id",
			requestBody: assignTagRequest{
				TagID: "1",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTag(gomock.Any(), "test-user-id", "test-id", "1").
					Return(db.Notification{}, nil)
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{ID: 1, GithubID: "test-id"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagActionResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "test-id", response.Notification.GithubID)
			},
		},
		{
			name:           "missing tagId returns 400",
			githubID:       "test-id",
			requestBody:    assignTagRequest{TagID: ""},
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "tag not found returns 404",
			githubID:    "test-id",
			requestBody: assignTagRequest{TagID: "999"},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTag(gomock.Any(), "test-user-id", "test-id", "999").
					Return(db.Notification{}, notificationcore.ErrTagNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "notification not found returns 404",
			githubID:    "not-found",
			requestBody: assignTagRequest{TagID: "1"},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTag(gomock.Any(), "test-user-id", "not-found", "1").
					Return(db.Notification{}, notificationcore.ErrNotificationNotFound)
			},
			expectedStatus: http.StatusNotFound,
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

			req := createRequest(
				http.MethodPost,
				"/notifications/"+url.PathEscape(tt.githubID),
				tt.requestBody,
			)
			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleAssignTagToNotification(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleAssignTagByName(t *testing.T) {
	tests := []struct {
		name           string
		githubID       string
		requestBody    interface{}
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success assigns tag by name",
			githubID: "test-id",
			requestBody: assignTagByNameRequest{
				TagName: "test-tag",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTagByName(gomock.Any(), "test-user-id", "test-id", "test-tag").
					Return(db.Notification{}, nil)
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{ID: 1, GithubID: "test-id"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagActionResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "test-id", response.Notification.GithubID)
			},
		},
		{
			name:           "missing tagName returns 400",
			githubID:       "test-id",
			requestBody:    assignTagByNameRequest{TagName: ""},
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body returns 400",
			githubID:       "test-id",
			requestBody:    "invalid json",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "notification not found returns 404",
			githubID: "not-found",
			requestBody: assignTagByNameRequest{
				TagName: "test-tag",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTagByName(gomock.Any(), "test-user-id", "not-found", "test-tag").
					Return(db.Notification{}, notificationcore.ErrNotificationNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "invalid slug error returns 400",
			githubID: "test-id",
			requestBody: assignTagByNameRequest{
				TagName: "!!!",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTagByName(gomock.Any(), "test-user-id", "test-id", "!!!").
					Return(db.Notification{}, notificationcore.ErrInvalidTagName)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "service error returns 500",
			githubID: "test-id",
			requestBody: assignTagByNameRequest{
				TagName: "test-tag",
			},
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					AssignTagByName(gomock.Any(), "test-user-id", "test-id", "test-tag").
					Return(db.Notification{}, errors.New("database error"))
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

			req := createRequest(
				http.MethodPost,
				"/notifications/"+url.PathEscape(tt.githubID),
				tt.requestBody,
			)
			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleAssignTagByName(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleRemoveTagFromNotification(t *testing.T) {
	tests := []struct {
		name           string
		githubID       string
		tagID          string
		setupMock      func(*notificationmocks.MockNotificationService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "success removes tag",
			githubID: "test-id",
			tagID:    "1",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					RemoveTag(gomock.Any(), "test-user-id", "test-id", "1").
					Return(db.Notification{}, nil)
				mockSvc.EXPECT().
					GetNotificationWithDetails(gomock.Any(), "test-user-id", "test-id", "").
					Return(models.Notification{ID: 1, GithubID: "test-id"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagActionResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "test-id", response.Notification.GithubID)
			},
		},
		{
			name:           "missing githubID returns 400",
			githubID:       "",
			tagID:          "1",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing tagId returns 400",
			githubID:       "test-id",
			tagID:          "",
			setupMock:      func(*notificationmocks.MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid tagId returns 500",
			githubID: "test-id",
			tagID:    "invalid",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					RemoveTag(gomock.Any(), "test-user-id", "test-id", "invalid").
					Return(db.Notification{}, notificationcore.ErrTagNotFound)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:     "notification not found returns 404",
			githubID: "not-found",
			tagID:    "1",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					RemoveTag(gomock.Any(), "test-user-id", "not-found", "1").
					Return(db.Notification{}, notificationcore.ErrNotificationNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "service error returns 500",
			githubID: "test-id",
			tagID:    "1",
			setupMock: func(mockSvc *notificationmocks.MockNotificationService) {
				mockSvc.EXPECT().
					RemoveTag(gomock.Any(), "test-user-id", "test-id", "1").
					Return(db.Notification{}, errors.New("database error"))
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

			req := createRequest(
				http.MethodDelete,
				"/notifications/"+url.PathEscape(tt.githubID)+"/tags/"+tt.tagID,
				nil,
			)
			rctx := chi.NewRouteContext()
			if tt.githubID != "" {
				rctx.URLParams.Add("githubID", tt.githubID)
			}
			if tt.tagID != "" {
				rctx.URLParams.Add("tagId", tt.tagID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()
			handler.handleRemoveTagFromNotification(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}
