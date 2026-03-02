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
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	"github.com/octobud-hq/octobud/backend/internal/db"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestHandler_handleGetReviewComments(t *testing.T) {
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

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/review-comments", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetReviewComments(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp ReviewCommentsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Empty(t, resp.ReviewComments)
	})

	t.Run("PR groups comments by review ID", func(t *testing.T) {
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

		now := time.Now()
		pos := 10
		mockGithub.EXPECT().
			FetchPullRequestComments(gomock.Any(), "owner", "repo", 42, 100, 1).
			Return([]types.PullRequestComment{
				{
					ID:                  1,
					PullRequestReviewID: 100,
					Body:                "comment on review 100",
					Path:                "file.go",
					DiffHunk:            "@@ -1,3 +1,3 @@",
					Position:            &pos,
					User:                types.SimpleUser{Login: "alice"},
					CreatedAt:           &now,
				},
				{
					ID:                  2,
					PullRequestReviewID: 100,
					Body:                "another on review 100",
					Path:                "file2.go",
					Position:            &pos,
					User:                types.SimpleUser{Login: "alice"},
					CreatedAt:           &now,
				},
				{
					ID:                  3,
					PullRequestReviewID: 200,
					Body:                "comment on review 200",
					Path:                "main.go",
					Position:            nil, // outdated
					User:                types.SimpleUser{Login: "bob"},
					CreatedAt:           &now,
				},
			}, nil)

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/review-comments", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetReviewComments(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp ReviewCommentsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Len(t, resp.ReviewComments["100"], 2)
		require.Len(t, resp.ReviewComments["200"], 1)
		require.True(t, resp.ReviewComments["200"][0].Outdated)
	})

	t.Run("threads replies under parent comment across reviews", func(t *testing.T) {
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

		now := time.Now()
		pos := 10
		parentID := int64(1)
		mockGithub.EXPECT().
			FetchPullRequestComments(gomock.Any(), "owner", "repo", 42, 100, 1).
			Return([]types.PullRequestComment{
				{
					ID:                  1,
					PullRequestReviewID: 100,
					Body:                "parent comment",
					Path:                "file.go",
					DiffHunk:            "@@ -1,3 +1,3 @@",
					Position:            &pos,
					User:                types.SimpleUser{Login: "alice"},
					CreatedAt:           &now,
				},
				{
					ID:                  2,
					PullRequestReviewID: 100,
					InReplyToID:         &parentID,
					Body:                "same-review reply",
					Path:                "file.go",
					Position:            &pos,
					User:                types.SimpleUser{Login: "alice"},
					CreatedAt:           &now,
				},
				{
					ID:                  3,
					PullRequestReviewID: 200, // different review
					InReplyToID:         &parentID,
					Body:                "cross-review reply",
					Path:                "file.go",
					Position:            &pos,
					User:                types.SimpleUser{Login: "bob"},
					CreatedAt:           &now,
				},
			}, nil)

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/review-comments", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetReviewComments(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp ReviewCommentsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		// Parent's review group should have 1 top-level comment with 2 replies
		require.Len(t, resp.ReviewComments["100"], 1)
		require.Equal(t, "parent comment", resp.ReviewComments["100"][0].Body)
		require.Len(t, resp.ReviewComments["100"][0].Replies, 2)
		require.Equal(t, "same-review reply", resp.ReviewComments["100"][0].Replies[0].Body)
		require.Equal(t, "cross-review reply", resp.ReviewComments["100"][0].Replies[1].Body)

		// The cross-review reply's original review group should have no entries
		require.Empty(t, resp.ReviewComments["200"])
	})

	t.Run("trims reply threads to maxReplies", func(t *testing.T) {
		parentID := int64(1)
		now := time.Now()
		pos := 10

		// Build 1 parent + 8 replies (exceeds maxReplies=6)
		comments := []types.PullRequestComment{
			{
				ID:                  1,
				PullRequestReviewID: 100,
				Body:                "parent",
				Path:                "file.go",
				Position:            &pos,
				User:                types.SimpleUser{Login: "alice"},
				CreatedAt:           &now,
			},
		}
		for i := int64(2); i <= 9; i++ {
			comments = append(comments, types.PullRequestComment{
				ID:                  i,
				PullRequestReviewID: 100,
				InReplyToID:         &parentID,
				Body:                "reply " + strconv.FormatInt(i, 10),
				Path:                "file.go",
				Position:            &pos,
				User:                types.SimpleUser{Login: "bob"},
				CreatedAt:           &now,
			})
		}

		result := threadReviewComments(comments)

		require.Len(t, result["100"], 1)
		parent := result["100"][0]
		require.Equal(t, "parent", parent.Body)
		// 8 replies trimmed to the last 6
		require.Len(t, parent.Replies, 6)
		// Oldest kept reply should be reply 4 (IDs 4-9)
		require.Equal(t, "reply 4", parent.Replies[0].Body)
		require.Equal(t, "reply 9", parent.Replies[5].Body)
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
			FetchPullRequestComments(gomock.Any(), "owner", "repo", 42, 100, 1).
			Return(nil, errors.New("API rate limit exceeded"))

		req := createRequest(http.MethodGet, "/notifications/"+testGithubID+"/review-comments", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("githubID", testGithubID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(helpers.ContextWithUserID(req.Context(), testUserID))

		w := httptest.NewRecorder()
		handler.handleGetReviewComments(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
