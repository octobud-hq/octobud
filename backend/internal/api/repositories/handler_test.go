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

package repositories

import (
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
	"github.com/octobud-hq/octobud/backend/internal/core/repository/mocks"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func setupTestHandler(
	ctrl *gomock.Controller,
) (*Handler, *mocks.MockRepositoryService, *authmocks.MockAuthService) {
	logger := zap.NewNop()
	mockSvc := mocks.NewMockRepositoryService(ctrl)
	mockAuthSvc := authmocks.NewMockAuthService(ctrl)
	return New(logger, mockSvc, mockAuthSvc), mockSvc, mockAuthSvc
}

func TestHandler_handleListRepositories(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockRepositoryService)
		expectedStatus int
		expectedBody   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success returns list of repositories",
			setupMock: func(m *mocks.MockRepositoryService) {
				repos := []models.Repository{
					{
						ID:       1,
						Name:     "test-repo",
						FullName: "owner/test-repo",
					},
					{
						ID:       2,
						Name:     "another-repo",
						FullName: "owner/another-repo",
					},
				}
				m.EXPECT().ListRepositories(gomock.Any(), "test-user-id").Return(repos, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listRepositoriesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Repositories, 2)
				require.Equal(t, "test-repo", response.Repositories[0].Name)
				require.Equal(t, "another-repo", response.Repositories[1].Name)
			},
		},
		{
			name: "success returns empty list when no repositories",
			setupMock: func(m *mocks.MockRepositoryService) {
				m.EXPECT().
					ListRepositories(gomock.Any(), "test-user-id").
					Return([]models.Repository{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response listRepositoriesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Len(t, response.Repositories, 0)
			},
		},
		{
			name: "error returns 500 when service fails",
			setupMock: func(m *mocks.MockRepositoryService) {
				m.EXPECT().
					ListRepositories(gomock.Any(), "test-user-id").
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Contains(t, response["error"], "failed to load repositories")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			handler, mockSvc, mockAuthSvc := setupTestHandler(ctrl)
			tt.setupMock(mockSvc)
			mockAuthSvc.EXPECT().
				GetUser(gomock.Any()).
				Return(&models.User{GithubUserID: "test-user-id"}, nil).
				AnyTimes()

			req := httptest.NewRequest(http.MethodGet, "/repositories", http.NoBody)
			req = req.WithContext(shared.ContextWithUserID(req.Context(), "test-user-id"))
			w := httptest.NewRecorder()

			router := chi.NewRouter()
			handler.Register(router)
			router.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				tt.expectedBody(t, w)
			}
		})
	}
}
