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

package tags

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	authmocks "github.com/octobud-hq/octobud/backend/internal/core/auth/mocks"
	tagmocks "github.com/octobud-hq/octobud/backend/internal/core/tag/mocks"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func setupTestHandler(
	ctrl *gomock.Controller,
) (*Handler, *tagmocks.MockTagService, *authmocks.MockAuthService) {
	logger := zap.NewNop()
	mockTagSvc := tagmocks.NewMockTagService(ctrl)
	mockAuthSvc := authmocks.NewMockAuthService(ctrl)
	return New(logger, mockTagSvc, mockAuthSvc), mockTagSvc, mockAuthSvc
}

func createRequest(method, url string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			// In test helper, return empty body on marshal error
			reqBody = nil
		}
	}
	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandler_handleListAllTags(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*tagmocks.MockTagService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns list of tags",
			setupMock: func(m *tagmocks.MockTagService) {
				tags := []models.Tag{
					{ID: "1", Name: "bug", Slug: "bug", UnreadCount: intPtr(5)},
					{ID: "2", Name: "feature", Slug: "feature", UnreadCount: intPtr(5)},
				}
				m.EXPECT().ListTagsWithUnreadCounts(gomock.Any(), "test-user-id").Return(tags, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listTagsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Tags, 2)
				require.Equal(t, "1", response.Tags[0].ID)
				require.Equal(t, "bug", response.Tags[0].Name)
			},
		},
		{
			name: "service error returns 500",
			setupMock: func(m *tagmocks.MockTagService) {
				m.EXPECT().
					ListTagsWithUnreadCounts(gomock.Any(), "test-user-id").
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to list tags", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockTagSvc, mockAuthSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockTagSvc)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodGet, "/tags", nil)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()

			handler.handleListAllTags(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleCreateTag(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*tagmocks.MockTagService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success creates tag",
			requestBody: createTagRequest{
				Name:        "Test Tag",
				Color:       stringPtr("#ff0000"),
				Description: stringPtr("Test description"),
			},
			setupMock: func(m *tagmocks.MockTagService) {
				m.EXPECT().
					CreateTag(gomock.Any(), "test-user-id", "Test Tag", stringPtr("#ff0000"), stringPtr("Test description")).
					Return(db.Tag{
						ID:           "1",
						Name:         "Test Tag",
						Slug:         "test-tag",
						Color:        sql.NullString{String: "#ff0000", Valid: true},
						Description:  sql.NullString{String: "Test description", Valid: true},
						DisplayOrder: 0,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Tag.ID)
				require.Equal(t, "Test Tag", response.Tag.Name)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid request body", response.Error)
			},
		},
		{
			name: "missing name returns 400",
			requestBody: createTagRequest{
				Name: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "name is required", response.Error)
			},
		},
		{
			name: "service error returns 500",
			requestBody: createTagRequest{
				Name: "Test Tag",
			},
			setupMock: func(m *tagmocks.MockTagService) {
				m.EXPECT().
					CreateTag(gomock.Any(), "test-user-id", "Test Tag", nil, nil).
					Return(db.Tag{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to create tag", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockTagSvc, mockAuthSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockTagSvc)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodPost, "/tags", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()

			handler.handleCreateTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleUpdateTag(t *testing.T) {
	tests := []struct {
		name           string
		tagID          string
		requestBody    interface{}
		setupMock      func(*tagmocks.MockTagService, string)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "success updates tag",
			tagID: "1",
			requestBody: updateTagRequest{
				Name:        "Updated Tag",
				Color:       stringPtr("#00ff00"),
				Description: stringPtr("Updated description"),
			},
			setupMock: func(m *tagmocks.MockTagService, tagID string) {
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", tagID, "Updated Tag", stringPtr("#00ff00"), stringPtr("Updated description")).
					Return(db.Tag{
						ID:           tagID,
						Name:         "Updated Tag",
						Slug:         "updated-tag",
						Color:        sql.NullString{String: "#00ff00", Valid: true},
						Description:  sql.NullString{String: "Updated description", Valid: true},
						DisplayOrder: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response tagEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "1", response.Tag.ID)
				require.Equal(t, "Updated Tag", response.Tag.Name)
			},
		},
		{
			name:           "missing tag ID returns 400",
			tagID:          "",
			requestBody:    updateTagRequest{Name: "Test"},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "tag ID is required", response.Error)
			},
		},
		{
			name:  "invalid tag ID returns 404",
			tagID: "invalid",
			requestBody: updateTagRequest{
				Name: "Test",
			},
			setupMock: func(m *tagmocks.MockTagService, tagID string) {
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", tagID, "Test", gomock.Any(), gomock.Any()).
					Return(db.Tag{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid body returns 400",
			tagID:          "1",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "invalid request body", response.Error)
			},
		},
		{
			name:  "missing name returns 400",
			tagID: "1",
			requestBody: updateTagRequest{
				Name: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "name is required", response.Error)
			},
		},
		{
			name:  "not found returns 404",
			tagID: "999",
			requestBody: updateTagRequest{
				Name: "Updated Tag",
			},
			setupMock: func(m *tagmocks.MockTagService, tagID string) {
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", tagID, "Updated Tag", gomock.Any(), gomock.Any()).
					Return(db.Tag{}, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "tag not found", response.Error)
			},
		},
		{
			name:  "service error returns 500",
			tagID: "1",
			requestBody: updateTagRequest{
				Name: "Updated Tag",
			},
			setupMock: func(m *tagmocks.MockTagService, tagID string) {
				m.EXPECT().
					UpdateTag(gomock.Any(), "test-user-id", tagID, "Updated Tag", gomock.Any(), gomock.Any()).
					Return(db.Tag{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, "failed to update tag", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockTagSvc, mockAuthSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockTagSvc, tt.tagID)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodPut, "/tags/"+tt.tagID, tt.requestBody)
			rctx := chi.NewRouteContext()
			if tt.tagID != "" {
				rctx.URLParams.Add("id", tt.tagID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()

			handler.handleUpdateTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

func TestHandler_handleDeleteTag(t *testing.T) {
	tests := []struct {
		name           string
		tagID          string
		setupMock      func(*tagmocks.MockTagService, string)
		expectedStatus int
	}{
		{
			name:  "success deletes tag",
			tagID: "1",
			setupMock: func(m *tagmocks.MockTagService, _ string) {
				m.EXPECT().DeleteTag(gomock.Any(), "test-user-id", "1").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing tag ID returns 400",
			tagID:          "",
			setupMock:      func(_ *tagmocks.MockTagService, _ string) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid tag ID returns 500",
			tagID: "invalid",
			setupMock: func(m *tagmocks.MockTagService, tagID string) {
				m.EXPECT().
					DeleteTag(gomock.Any(), "test-user-id", tagID).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:  "service error returns 500",
			tagID: "1",
			setupMock: func(m *tagmocks.MockTagService, _ string) {
				m.EXPECT().
					DeleteTag(gomock.Any(), "test-user-id", "1").
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockTagSvc, mockAuthSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockTagSvc, tt.tagID)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodDelete, "/tags/"+tt.tagID, nil)
			rctx := chi.NewRouteContext()
			if tt.tagID != "" {
				rctx.URLParams.Add("id", tt.tagID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))

			w := httptest.NewRecorder()

			handler.handleDeleteTag(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_handleReorderTags(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*tagmocks.MockTagService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success reorders tags",
			requestBody: reorderTagsRequest{
				TagIDs: []string{"1", "2", "3"},
			},
			setupMock: func(m *tagmocks.MockTagService) {
				m.EXPECT().
					ReorderTags(gomock.Any(), "test-user-id", []string{"1", "2", "3"}).
					Return([]db.Tag{
						{ID: "1", Name: "Tag 1", Slug: "tag-1", DisplayOrder: 0},
						{ID: "2", Name: "Tag 2", Slug: "tag-2", DisplayOrder: 1},
						{ID: "3", Name: "Tag 3", Slug: "tag-3", DisplayOrder: 2},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listTagsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Tags, 3)
			},
		},
		{
			name:           "invalid body returns 400",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty tagIDs returns 400",
			requestBody: reorderTagsRequest{
				TagIDs: []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response errorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response.Error, "tagIDs is required")
			},
		},
		{
			name: "invalid tag ID returns 500",
			requestBody: reorderTagsRequest{
				TagIDs: []string{"1", "invalid", "3"},
			},
			setupMock: func(m *tagmocks.MockTagService) {
				m.EXPECT().
					ReorderTags(gomock.Any(), "test-user-id", []string{"1", "invalid", "3"}).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "service error returns 500",
			requestBody: reorderTagsRequest{
				TagIDs: []string{"1", "2"},
			},
			setupMock: func(m *tagmocks.MockTagService) {
				m.EXPECT().
					ReorderTags(gomock.Any(), "test-user-id", []string{"1", "2"}).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			const testUserID = "test-user-id"
			handler, mockTagSvc, mockAuthSvc := setupTestHandler(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockTagSvc)
			}
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: testUserID}, nil).
				AnyTimes()

			req := createRequest(http.MethodPost, "/tags/reorder", tt.requestBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), testUserID))
			w := httptest.NewRecorder()

			handler.handleReorderTags(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}

// Helper functions
func intPtr(i int64) *int64 {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

type errorResponse struct {
	Error string `json:"error"`
}
