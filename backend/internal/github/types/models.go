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

// Package types provides the types for the GitHub API.
package types //nolint:revive // Sigh.

import (
	"encoding/json"
	"time"
)

// NotificationThread represents one GitHub notification thread response item.
type NotificationThread struct {
	ID              string              `json:"id"`
	Repository      RepositorySnapshot  `json:"repository"`
	Subject         NotificationSubject `json:"subject"`
	Reason          string              `json:"reason"`
	Unread          bool                `json:"unread"`
	UpdatedAt       time.Time           `json:"updated_at"`
	LastReadAt      *time.Time          `json:"last_read_at"`
	URL             string              `json:"url"`
	SubscriptionURL string              `json:"subscription_url"`
	Raw             json.RawMessage     `json:"-"`
}

// NotificationSubject provides the subject payload for a notification thread.
type NotificationSubject struct {
	Title            string `json:"title"`
	URL              string `json:"url"`
	LatestCommentURL string `json:"latest_comment_url"`
	Type             string `json:"type"`
}

// RepositorySnapshot models the subset of repository metadata we persist.
type RepositorySnapshot struct {
	ID            int64      `json:"id"`
	NodeID        string     `json:"node_id"`
	Name          string     `json:"name"`
	FullName      string     `json:"full_name"`
	Owner         SimpleUser `json:"owner"`
	Private       bool       `json:"private"`
	Description   *string    `json:"description"`
	HTMLURL       string     `json:"html_url"`
	Fork          bool       `json:"fork"`
	Visibility    *string    `json:"visibility"`
	DefaultBranch *string    `json:"default_branch"`
	Archived      bool       `json:"archived"`
	Disabled      bool       `json:"disabled"`
	PushedAt      *time.Time `json:"pushed_at"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// Raw returns the JSON-encoded representation of the repository snapshot.
func (r RepositorySnapshot) Raw() json.RawMessage {
	raw, err := json.Marshal(r)
	if err != nil {
		return nil
	}
	return raw
}

// SimpleUser captures the minimal set of user fields we care about for repository owners.
type SimpleUser struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// SubjectInfo contains extracted location information about a notification's subject.
// Used to make GitHub API calls for the subject (e.g., fetching timeline).
// Note: Subject type should come from the notification's SubjectType field, not from URL parsing.
type SubjectInfo struct {
	Owner  string
	Repo   string
	Number int
}

// IssueComment represents a comment on an issue or pull request.
type IssueComment struct {
	ID        int64      `json:"id"`
	Body      string     `json:"body"`
	User      SimpleUser `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	HTMLURL   string     `json:"html_url"`
}

// PullRequestReview represents a review on a pull request.
type PullRequestReview struct {
	ID          int64      `json:"id"`
	State       string     `json:"state"`
	Body        string     `json:"body"`
	User        SimpleUser `json:"user"`
	SubmittedAt time.Time  `json:"submitted_at"`
	HTMLURL     string     `json:"html_url"`
}

// TimelineEvent represents a single event in a PR/issue timeline.
type TimelineEvent struct {
	Event       string          `json:"event"`
	ID          json.RawMessage `json:"id,omitempty"` // Can be int64 or string
	CreatedAt   *time.Time      `json:"created_at,omitempty"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
	SubmittedAt *time.Time      `json:"submitted_at,omitempty"`
	Actor       *SimpleUser     `json:"actor,omitempty"`
	User        *SimpleUser     `json:"user,omitempty"`
	Body        string          `json:"body,omitempty"`
	State       string          `json:"state,omitempty"`   // For reviews
	Message     string          `json:"message,omitempty"` // For commits
	SHA         string          `json:"sha,omitempty"`     // For commits
	HTMLURL     string          `json:"html_url,omitempty"`
	CommitID    string          `json:"commit_id,omitempty"`  // For some events
	CommitURL   string          `json:"commit_url,omitempty"` // For some events
}
