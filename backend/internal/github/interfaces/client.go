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

// Package githubinterfaces defines interfaces for GitHub operations.
// This package is separate to avoid import cycles when generating mocks.
package githubinterfaces

import (
	"context"
	"encoding/json"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

// TokenObserverFunc is invoked on each authenticated GitHub API response so the
// caller can observe token state (expiration countdown, invalidation on 401).
// expiresAt is nil when the response did not include an expiration header
// (e.g. classic PATs or PATs configured without expiry).
type TokenObserverFunc func(statusCode int, expiresAt *time.Time)

// Client defines the interface for GitHub API operations.
type Client interface {
	SetToken(ctx context.Context, token string) error
	// SetTokenObserver registers a callback invoked once per authenticated
	// response. Passing nil clears the observer.
	SetTokenObserver(observer TokenObserverFunc)
	// FetchNotifications retrieves notification threads from GitHub.
	// - since: only fetch notifications updated after this time (nil = use GitHub default window)
	// - before: only fetch notifications updated before this time (nil = no upper bound)
	// - unreadOnly: when true, only fetch unread notifications (all=false in GitHub API)
	//   The safe default is false, which fetches all notifications (all=true in GitHub API)
	FetchNotifications(
		ctx context.Context,
		since *time.Time,
		before *time.Time,
		unreadOnly bool,
	) ([]types.NotificationThread, error)
	FetchSubjectRaw(ctx context.Context, subjectURL string) (json.RawMessage, error)
	// FetchTimeline retrieves timeline events for an issue or pull request using REST API.
	// Returns the events on the requested page, plus the last page number parsed from
	// the Link response header (1 if no Link header / single page).
	FetchTimeline(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.TimelineEvent, int, error)
	FetchIssueComments(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.IssueComment, error)
	FetchPullRequestReviews(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.PullRequestReview, error)
	// FetchPullRequestComments retrieves inline review comments for a pull request.
	// These are the line-level comments attached to PR reviews.
	FetchPullRequestComments(
		ctx context.Context,
		owner, repo string,
		number, perPage, page int,
	) ([]types.PullRequestComment, error)
	// FetchPRCommits retrieves commits for a pull request.
	// Used to enrich "committed" timeline events with GitHub user info (login + avatar).
	FetchPRCommits(
		ctx context.Context,
		owner, repo string,
		number int,
	) ([]types.PRCommit, error)
	// FetchDiscussionComments retrieves comments for a discussion using GraphQL API.
	// Returns comments as TimelineEvents, hasNextPage, endCursor, and error.
	// This method converts discussion comments to TimelineEvent format for consistency.
	FetchDiscussionComments(
		ctx context.Context,
		owner, repo string,
		number, first int,
		after string,
	) ([]types.TimelineEvent, bool, string, error)
	// FetchRateLimit retrieves the current GitHub API rate-limit status. This
	// endpoint does not itself count against the rate limit, so it's safe to
	// call on demand (e.g. from a diagnostics page).
	FetchRateLimit(ctx context.Context) (*types.RateLimit, error)
}
