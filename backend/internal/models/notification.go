// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"encoding/json"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Notification represents a notification with all enriched data (repository, tags, action hints)
type Notification struct {
	ID                      int64           `json:"id"`
	GithubID                string          `json:"githubId"`
	RepositoryID            int64           `json:"repositoryId"`
	PullRequestID           *int64          `json:"pullRequestId,omitempty"`
	SubjectType             string          `json:"subjectType"`
	SubjectTitle            string          `json:"subjectTitle"`
	SubjectURL              *string         `json:"subjectURL,omitempty"`
	SubjectLatestCommentURL *string         `json:"subjectLatestCommentURL,omitempty"`
	Reason                  *string         `json:"reason,omitempty"`
	Archived                bool            `json:"archived"`
	IsRead                  bool            `json:"isRead"`
	Muted                   bool            `json:"muted"`
	SnoozedUntil            *time.Time      `json:"snoozedUntil,omitempty"`
	SnoozedAt               *time.Time      `json:"snoozedAt,omitempty"`
	EffectiveSortDate       time.Time       `json:"effectiveSortDate"`
	Starred                 bool            `json:"starred"`
	Filtered                bool            `json:"filtered"`
	GithubUnread            *bool           `json:"githubUnread,omitempty"`
	GithubUpdatedAt         *time.Time      `json:"githubUpdatedAt,omitempty"`
	GithubLastReadAt        *time.Time      `json:"githubLastReadAt,omitempty"`
	GithubURL               *string         `json:"githubURL,omitempty"`
	GithubSubscriptionURL   *string         `json:"githubSubscriptionURL,omitempty"`
	ImportedAt              time.Time       `json:"importedAt"`
	Payload                 json.RawMessage `json:"payload,omitempty"`
	SubjectRaw              json.RawMessage `json:"subjectRaw,omitempty"`
	SubjectFetchedAt        *time.Time      `json:"subjectFetchedAt,omitempty"`
	SubjectNumber           *int64          `json:"subjectNumber,omitempty"`
	SubjectState            *string         `json:"subjectState,omitempty"`
	SubjectMerged           *bool           `json:"subjectMerged,omitempty"`
	SubjectStateReason      *string         `json:"subjectStateReason,omitempty"`
	AuthorLogin             *string         `json:"authorLogin,omitempty"`
	Repository              *Repository     `json:"repository,omitempty"`
	ActionHints             *ActionHints    `json:"actionHints,omitempty"`
	Tags                    []Tag           `json:"tags,omitempty"`
}

// NotificationFromDB converts a db.Notification to a models.Notification (without enrichment)
func NotificationFromDB(notification db.Notification) Notification {
	var payload json.RawMessage
	if notification.Payload.Valid {
		payload = notification.Payload.RawMessage
	}

	var subjectRaw json.RawMessage
	if notification.SubjectRaw.Valid {
		subjectRaw = notification.SubjectRaw.RawMessage
	}

	return Notification{
		ID:                      notification.ID,
		GithubID:                notification.GithubID,
		RepositoryID:            notification.RepositoryID,
		PullRequestID:           NullInt64Ptr(notification.PullRequestID),
		SubjectType:             notification.SubjectType,
		SubjectTitle:            notification.SubjectTitle,
		SubjectURL:              NullStringPtr(notification.SubjectURL),
		SubjectLatestCommentURL: NullStringPtr(notification.SubjectLatestCommentURL),
		Reason:                  NullStringPtr(notification.Reason),
		Archived:                notification.Archived,
		IsRead:                  notification.IsRead,
		Muted:                   notification.Muted,
		SnoozedUntil:            NullTimePtr(notification.SnoozedUntil),
		SnoozedAt:               NullTimePtr(notification.SnoozedAt),
		EffectiveSortDate:       notification.EffectiveSortDate,
		Starred:                 notification.Starred,
		Filtered:                notification.Filtered,
		GithubUnread:            NullBoolPtr(notification.GithubUnread),
		GithubUpdatedAt:         NullTimePtr(notification.GithubUpdatedAt),
		GithubLastReadAt:        NullTimePtr(notification.GithubLastReadAt),
		GithubURL:               NullStringPtr(notification.GithubURL),
		GithubSubscriptionURL:   NullStringPtr(notification.GithubSubscriptionURL),
		ImportedAt:              notification.ImportedAt,
		Payload:                 payload,
		AuthorLogin:             NullStringPtr(notification.AuthorLogin),
		SubjectRaw:              subjectRaw,
		SubjectFetchedAt:        NullTimePtr(notification.SubjectFetchedAt),
		SubjectNumber:           NullInt32ToInt64Ptr(notification.SubjectNumber),
		SubjectState:            NullStringPtr(notification.SubjectState),
		SubjectMerged:           NullBoolPtr(notification.SubjectMerged),
		SubjectStateReason:      NullStringPtr(notification.SubjectStateReason),
	}
}

// PollNotification contains only essential fields for polling. Any
// info that is needed for showing desktop notifications should be included here.
type PollNotification struct {
	ID                int64   `json:"id"`
	GithubID          string  `json:"githubId"`
	EffectiveSortDate string  `json:"effectiveSortDate"` // RFC3339Nano format
	Archived          bool    `json:"archived"`
	Muted             bool    `json:"muted"`
	RepoFullName      string  `json:"repoFullName,omitempty"`
	SubjectTitle      string  `json:"subjectTitle,omitempty"`
	SubjectType       string  `json:"subjectType,omitempty"`
	Reason            *string `json:"reason,omitempty"`
}
