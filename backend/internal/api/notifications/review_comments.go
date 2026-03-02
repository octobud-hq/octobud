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
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	"github.com/octobud-hq/octobud/backend/internal/github"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

// ReviewCommentsResponse is the response structure for the review comments endpoint.
type ReviewCommentsResponse struct {
	ReviewComments map[string][]ThreadReviewComment `json:"reviewComments"`
}

func (h *Handler) handleGetReviewComments(
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
			"failed to fetch notification for review comments",
			zap.String("github_id", githubID),
			zap.Error(errors.Join(ErrFailedToFetchNotification, err)),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to fetch notification")
		return
	}

	// Only PR types have review comments
	normalizedType := strings.ToLower(strings.ReplaceAll(notification.SubjectType, "_", ""))
	if normalizedType != "pullrequest" && normalizedType != "pr" {
		helpers.WriteJSON(w, http.StatusOK, ReviewCommentsResponse{
			ReviewComments: map[string][]ThreadReviewComment{},
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
			"failed to parse subject info for review comments",
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

	// Fetch inline review comments for this PR.
	// Limited to the first 100 comments (single page). PRs with more than 100
	// inline comments will be truncated; pagination can be added if needed.
	comments, err := h.githubClient.FetchPullRequestComments(
		ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number,
		100, 1,
	)
	if err != nil {
		h.logger.Error(
			"failed to fetch review comments",
			zap.String("github_id", githubID),
			zap.String("owner", subjectInfo.Owner),
			zap.String("repo", subjectInfo.Repo),
			zap.Int("number", subjectInfo.Number),
			zap.Error(err),
		)
		helpers.WriteError(w, http.StatusInternalServerError, "failed to fetch review comments")
		return
	}

	helpers.WriteJSON(w, http.StatusOK, ReviewCommentsResponse{
		ReviewComments: threadReviewComments(comments),
	})
}

// threadReviewComments groups PR review comments by review ID and threads
// replies under their parent comment (including cross-review replies).
// Reply threads are trimmed to the last 6 entries.
func threadReviewComments(comments []types.PullRequestComment) map[string][]ThreadReviewComment {
	// Convert all comments to response format, indexed by their own ID.
	allComments := make(map[string]*ThreadReviewComment, len(comments))
	commentReviewKey := make(map[string]string, len(comments)) // comment ID → review key

	for i := range comments {
		c := &comments[i]
		id := strconv.FormatInt(c.ID, 10)
		trc := &ThreadReviewComment{
			ID:       id,
			Body:     c.Body,
			Path:     c.Path,
			DiffHunk: c.DiffHunk,
			Outdated: c.Position == nil,
			Author: ThreadAuthor{
				Login:     c.User.Login,
				AvatarURL: c.User.AvatarURL,
			},
			HTMLURL: c.HTMLURL,
		}
		if c.CreatedAt != nil {
			ts := c.CreatedAt.Format(time.RFC3339)
			trc.CreatedAt = &ts
		}
		allComments[id] = trc
		commentReviewKey[id] = strconv.FormatInt(c.PullRequestReviewID, 10)
	}

	// Thread replies under their parent comment.
	// Replies from different reviews are moved to the parent's review group.
	topLevel := make(map[string][]string) // review key → ordered top-level comment IDs
	for i := range comments {
		c := &comments[i]
		id := strconv.FormatInt(c.ID, 10)

		if c.InReplyToID != nil {
			parentID := strconv.FormatInt(*c.InReplyToID, 10)
			if parent, ok := allComments[parentID]; ok {
				parent.Replies = append(parent.Replies, *allComments[id])
				delete(allComments, id) // consumed as a reply
				continue
			}
		}

		// Top-level comment: add to its review group
		reviewKey := commentReviewKey[id]
		topLevel[reviewKey] = append(topLevel[reviewKey], id)
	}

	// Trim reply threads to the last 6 entries
	const maxReplies = 6
	for _, trc := range allComments {
		if len(trc.Replies) > maxReplies {
			trc.Replies = trc.Replies[len(trc.Replies)-maxReplies:]
		}
	}

	// Build the result grouped by review key
	result := make(map[string][]ThreadReviewComment, len(topLevel))
	for reviewKey, ids := range topLevel {
		for _, id := range ids {
			if trc, ok := allComments[id]; ok {
				result[reviewKey] = append(result[reviewKey], *trc)
			}
		}
	}

	return result
}
