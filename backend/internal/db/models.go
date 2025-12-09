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

package db

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Notification represents a notification
type Notification struct {
	ID                      int64
	UserID                  string
	GithubID                string
	RepositoryID            int64
	PullRequestID           sql.NullInt64
	SubjectType             string
	SubjectTitle            string
	SubjectURL              sql.NullString
	SubjectLatestCommentURL sql.NullString
	Reason                  sql.NullString
	Archived                bool
	GithubUnread            sql.NullBool
	GithubUpdatedAt         sql.NullTime
	GithubLastReadAt        sql.NullTime
	GithubURL               sql.NullString
	GithubSubscriptionURL   sql.NullString
	ImportedAt              time.Time
	Payload                 NullRawMessage
	SubjectRaw              NullRawMessage
	SubjectFetchedAt        sql.NullTime
	AuthorLogin             sql.NullString
	AuthorID                sql.NullInt64
	IsRead                  bool
	Muted                   bool
	SnoozedUntil            sql.NullTime
	EffectiveSortDate       time.Time
	SnoozedAt               sql.NullTime
	Starred                 bool
	Filtered                bool
	TagIDs                  []string // Changed from []int64 to []string for UUID tags
	SubjectNumber           sql.NullInt32
	SubjectState            sql.NullString
	SubjectMerged           sql.NullBool
	SubjectStateReason      sql.NullString
}

// PullRequest represents a pull request
type PullRequest struct {
	ID           int64
	UserID       string
	RepositoryID int64
	GithubID     sql.NullInt64
	NodeID       sql.NullString
	Number       int32
	Title        sql.NullString
	State        sql.NullString
	Draft        sql.NullBool
	Merged       sql.NullBool
	AuthorLogin  sql.NullString
	AuthorID     sql.NullInt64
	CreatedAt    sql.NullTime
	UpdatedAt    sql.NullTime
	ClosedAt     sql.NullTime
	MergedAt     sql.NullTime
	Raw          NullRawMessage
}

// Repository represents a repository
type Repository struct {
	ID             int64
	UserID         string
	GithubID       sql.NullInt64
	NodeID         sql.NullString
	Name           string
	FullName       string
	OwnerLogin     sql.NullString
	OwnerID        sql.NullInt64
	Private        sql.NullBool
	Description    sql.NullString
	HTMLURL        sql.NullString
	Fork           sql.NullBool
	Visibility     sql.NullString
	DefaultBranch  sql.NullString
	Archived       sql.NullBool
	Disabled       sql.NullBool
	PushedAt       sql.NullTime
	CreatedAt      sql.NullTime
	UpdatedAt      sql.NullTime
	Raw            NullRawMessage
	OwnerAvatarURL sql.NullString
	OwnerHTMLURL   sql.NullString
}

// Rule represents a rule
type Rule struct {
	ID           string // UUID
	UserID       string
	Name         string
	Description  sql.NullString
	Query        sql.NullString
	Enabled      bool
	Actions      json.RawMessage
	DisplayOrder int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ViewID       sql.NullString // References views.id (UUID)
}

// SyncState represents a sync state
type SyncState struct {
	ID                         int64
	UserID                     string
	LastSuccessfulPoll         sql.NullTime
	LastNotificationEtag       sql.NullString
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	LatestNotificationAt       sql.NullTime
	InitialSyncCompletedAt     sql.NullTime
	OldestNotificationSyncedAt sql.NullTime
}

// Tag represents a tag
type Tag struct {
	ID           string // UUID
	UserID       string
	Name         string
	Color        sql.NullString
	Description  sql.NullString
	CreatedAt    time.Time
	DisplayOrder int32
	Slug         string
}

// TagAssignment represents a tag assignment
type TagAssignment struct {
	ID         string // UUID
	UserID     string
	TagID      string // References tags.id (UUID)
	EntityType string
	EntityID   int64
	CreatedAt  time.Time
}

// User represents a user
type User struct {
	ID                   int64
	GithubUserID         sql.NullString
	GithubUsername       sql.NullString
	GithubTokenEncrypted sql.NullString
	CreatedAt            time.Time
	UpdatedAt            time.Time
	SyncSettings         NullRawMessage
	RetentionSettings    NullRawMessage
	UpdateSettings       NullRawMessage
	MutedUntil           sql.NullTime
}

// View represents a view
type View struct {
	ID           string // UUID
	UserID       string
	Name         string
	Description  sql.NullString
	IsDefault    bool
	CreatedAt    time.Time
	Icon         sql.NullString
	Slug         string
	Query        sql.NullString
	DisplayOrder int32
}
