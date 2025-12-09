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
	"time"

	"github.com/octobud-hq/octobud/backend/internal/models"
)

// NotificationResponse is the response type for a single notification
type NotificationResponse = models.Notification

// RepositoryResponse is the response type for a repository
type RepositoryResponse = models.Repository

// TagResponse is the response type for a tag
type TagResponse = models.Tag

// ActionHints is the response type for action hints
type ActionHints = models.ActionHints

// ThreadItem represents either a comment, review, or timeline event in the unified response.
type ThreadItem struct {
	Type        string       `json:"type"` // "comment", "review", "committed", "merged", etc.
	ID          interface{}  `json:"id"`
	Body        string       `json:"body"`
	Author      ThreadAuthor `json:"author"`
	CreatedAt   *string      `json:"createdAt,omitempty"`   // For comments
	UpdatedAt   *string      `json:"updatedAt,omitempty"`   // For comments
	SubmittedAt *string      `json:"submittedAt,omitempty"` // For reviews
	State       *string      `json:"state,omitempty"`       // For reviews
	Message     *string      `json:"message,omitempty"`     // For commits
	SHA         *string      `json:"sha,omitempty"`         // For commits
	HTMLURL     string       `json:"htmlURL"`
	Timestamp   time.Time    `json:"-"` // Internal field for sorting
}

// ThreadAuthor represents the author of a comment, review, or event.
type ThreadAuthor struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatarURL"`
}

// TimelineResponse is the response structure for the timeline endpoint.
type TimelineResponse struct {
	Items   []ThreadItem `json:"items"`
	Total   int          `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"perPage"`
	HasMore bool         `json:"hasMore"`
}

// listNotificationsResponse is the response type for a list of notifications
type listNotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int64                  `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"pageSize"`
}

type notificationDetailResponse struct {
	Notification NotificationResponse `json:"notification"`
}

type refreshNotificationSubjectResponse struct {
	Notification NotificationResponse `json:"notification"`
}

type notificationActionResponse struct {
	Notification NotificationResponse `json:"notification"`
}

type bulkNotificationsResponse struct {
	Count int `json:"count"`
}

// PollNotificationResponse contains only the minimal fields needed for polling
type PollNotificationResponse struct {
	ID                int64  `json:"id"`
	GithubID          string `json:"githubId"`
	EffectiveSortDate string `json:"effectiveSortDate"`
	Archived          bool   `json:"archived"`
	Muted             bool   `json:"muted"`
	// Fields needed for desktop notifications
	RepoFullName string  `json:"repoFullName,omitempty"`
	SubjectTitle string  `json:"subjectTitle,omitempty"`
	SubjectType  string  `json:"subjectType,omitempty"`
	Reason       *string `json:"reason,omitempty"`
}

// listPollNotificationsResponse is the response type for poll notification polling
type listPollNotificationsResponse struct {
	Notifications []PollNotificationResponse `json:"notifications"`
	Total         int64                      `json:"total"`
	Page          int                        `json:"page"`
	PageSize      int                        `json:"pageSize"`
}
