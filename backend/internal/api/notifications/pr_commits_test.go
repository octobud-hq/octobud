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
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	"github.com/octobud-hq/octobud/backend/internal/db"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestHandler_handleGetPRCommits(t *testing.T) {
	const testUserID = "test-user-id"
	const testGithubID = "12345"

	prNotification := db.Notification{
		ID:          1,
		GithubID:    testGithubID,
		SubjectType: "PullRequest",
		SubjectURL: sql.NullString{
			String: "https://api.github.com/repos/owner/repo/pulls/42",
			Valid:  true,
		},
	}

	issueNotification := db.Notification{
		ID:          2,
		GithubID:    testGithubID,
		SubjectType: "Issue",
	}

	t.Run("non-PR returns empty map", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handler, mockNotifSvc, _, mockAuthSvc := setupTestHandler(ctrl)
		mockAuthSvc.EXPECT().GetUser(gomock.Any()).
			Return(&models.User{GithubUserID: testUserID}, nil).AnyTimes()
		mockNotifSvc.EXPECT().
			GetByGithubID(gomock.Any(), testUserID, testGithubID).
			Return(issueNotification, nil)

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/pr-commits", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetPRCommits(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp PRCommitsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.Commits)
	})

	t.Run("PR returns SHA-to-author map", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handler, mockNotifSvc, _, mockAuthSvc := setupTestHandler(ctrl)
		mockGithub := githubmocks.NewMockClient(ctrl)
		handler.githubClient = mockGithub

		mockAuthSvc.EXPECT().GetUser(gomock.Any()).
			Return(&models.User{GithubUserID: testUserID}, nil).AnyTimes()
		mockNotifSvc.EXPECT().
			GetByGithubID(gomock.Any(), testUserID, testGithubID).
			Return(prNotification, nil)

		mockGithub.EXPECT().
			FetchPRCommits(gomock.Any(), "owner", "repo", 42).
			Return([]types.PRCommit{
				{
					SHA: "abc1234",
					Author: &types.SimpleUser{
						Login:     "alice",
						AvatarURL: "https://avatars.githubusercontent.com/u/1",
					},
				},
				{
					SHA: "def5678",
					Author: &types.SimpleUser{
						Login:     "bob",
						AvatarURL: "https://avatars.githubusercontent.com/u/2",
					},
				},
			}, nil)

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/pr-commits", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetPRCommits(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp PRCommitsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Len(t, resp.Commits, 2)
		require.Equal(t, "alice", resp.Commits["abc1234"].Login)
		require.Equal(
			t,
			"https://avatars.githubusercontent.com/u/1",
			resp.Commits["abc1234"].AvatarURL,
		)
		require.Equal(t, "bob", resp.Commits["def5678"].Login)
	})

	t.Run("commit with nil author is excluded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handler, mockNotifSvc, _, mockAuthSvc := setupTestHandler(ctrl)
		mockGithub := githubmocks.NewMockClient(ctrl)
		handler.githubClient = mockGithub

		mockAuthSvc.EXPECT().GetUser(gomock.Any()).
			Return(&models.User{GithubUserID: testUserID}, nil).AnyTimes()
		mockNotifSvc.EXPECT().
			GetByGithubID(gomock.Any(), testUserID, testGithubID).
			Return(prNotification, nil)

		mockGithub.EXPECT().
			FetchPRCommits(gomock.Any(), "owner", "repo", 42).
			Return([]types.PRCommit{
				{
					SHA: "abc1234",
					Author: &types.SimpleUser{
						Login:     "alice",
						AvatarURL: "https://avatars.githubusercontent.com/u/1",
					},
				},
				{
					SHA:    "def5678",
					Author: nil, // nil author (e.g. bot or unlinked git identity)
				},
			}, nil)

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/pr-commits", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetPRCommits(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp PRCommitsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Len(t, resp.Commits, 1)
		require.Equal(t, "alice", resp.Commits["abc1234"].Login)
	})

	t.Run("notification not found returns 404", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handler, mockNotifSvc, _, mockAuthSvc := setupTestHandler(ctrl)
		mockAuthSvc.EXPECT().GetUser(gomock.Any()).
			Return(&models.User{GithubUserID: testUserID}, nil).AnyTimes()
		mockNotifSvc.EXPECT().
			GetByGithubID(gomock.Any(), testUserID, testGithubID).
			Return(db.Notification{}, sql.ErrNoRows)

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/pr-commits", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetPRCommits(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GitHub error returns 500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		handler, mockNotifSvc, _, mockAuthSvc := setupTestHandler(ctrl)
		mockGithub := githubmocks.NewMockClient(ctrl)
		handler.githubClient = mockGithub

		mockAuthSvc.EXPECT().GetUser(gomock.Any()).
			Return(&models.User{GithubUserID: testUserID}, nil).AnyTimes()
		mockNotifSvc.EXPECT().
			GetByGithubID(gomock.Any(), testUserID, testGithubID).
			Return(prNotification, nil)
		mockGithub.EXPECT().
			FetchPRCommits(gomock.Any(), "owner", "repo", 42).
			Return(nil, errors.New("API rate limit exceeded"))

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/pr-commits", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetPRCommits(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
