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
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	"github.com/octobud-hq/octobud/backend/internal/github"
)

// PRCommitAuthor represents a commit author for the pr-commits response.
type PRCommitAuthor struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatarUrl"`
}

// PRCommitsResponse is the response structure for the pr-commits endpoint.
type PRCommitsResponse struct {
	// Commits maps SHA to the GitHub user who authored the commit.
	Commits map[string]PRCommitAuthor `json:"commits"`
}

func (h *Handler) handleGetPRCommits(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()
	userID, ok := helpers.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}
	githubID := chi.URLParam(r, "githubID")

	// Fetch notification from database
	notification, err := h.notifications.GetByGithubID(ctx, userID, githubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			helpers.WriteError(w, http.StatusNotFound, "notification not found")
			return
		}
		h.logger.Error(
			"failed to fetch notification for pr commits",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToFetchNotification, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to fetch notification")
		return
	}

	// Only PR types have commits
	normalizedType := strings.ToLower(strings.ReplaceAll(notification.SubjectType, "_", ""))
	if normalizedType != "pullrequest" && normalizedType != "pr" {
		helpers.WriteJSON(w, http.StatusOK, PRCommitsResponse{
			Commits: map[string]PRCommitAuthor{},
		})
		return
	}

	// Extract subject info
	subjectURL := ""
	if notification.SubjectURL.Valid {
		subjectURL = notification.SubjectURL.String
	}

	var subjectRaw json.RawMessage
	if notification.SubjectRaw.Valid {
		subjectRaw = notification.SubjectRaw.RawMessage
	}

	subjectInfo, err := github.ExtractSubjectInfo(subjectURL, subjectRaw)
	if err != nil {
		h.logger.Error(
			"failed to parse subject info for pr commits",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToParseSubjectInfo, err)),
		)
		helpers.WriteError(w, http.StatusBadRequest, "invalid notification subject")
		return
	}

	if h.githubClient == nil {
		h.logger.Error(
			"GitHub client not configured",
			zap.String("github_id", githubID),
			zap.Error(ErrGitHubClientNotConfigured),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "GitHub client not configured")
		return
	}

	commits, err := h.githubClient.FetchPRCommits(
		ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number,
	)
	if err != nil {
		h.logger.Error(
			"failed to fetch PR commits",
			zap.String("github_id", githubID),
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Error(err),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to fetch PR commits")
		return
	}

	// Build SHA -> author map
	result := make(map[string]PRCommitAuthor, len(commits))
	for _, c := range commits {
		if c.Author != nil && c.SHA != "" {
			result[c.SHA] = PRCommitAuthor{
				Login:     c.Author.Login,
				AvatarURL: c.Author.AvatarURL,
			}
		}
	}

	helpers.WriteJSON(w, http.StatusOK, PRCommitsResponse{
		Commits: result,
	})
}
