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

// Package db provides the database queries for the application.
package db

import (
	"database/sql"
)

// NotificationQuery represents a complete notification query ready for SQL execution
type NotificationQuery struct {
	Joins          []string      // SQL JOIN clauses needed
	Where          []string      // SQL WHERE conditions
	Args           []interface{} // Query parameters for prepared statements
	Limit          int32
	Offset         int32
	IncludeSubject bool // Whether to include subject_raw in SELECT (default: true for backward compatibility)
}

// ListNotificationsFromQueryResult contains the notifications and total count
type ListNotificationsFromQueryResult struct {
	Notifications []Notification
	Total         int64
}

// BulkSnoozeNotificationsByQueryParams contains the parameters for snoozing by query
type BulkSnoozeNotificationsByQueryParams struct {
	Query        NotificationQuery
	SnoozedUntil sql.NullTime
}

// BulkSnoozeNotificationsParams contains the parameters for bulk snooze
type BulkSnoozeNotificationsParams struct {
	GithubIDs    []string
	SnoozedUntil sql.NullTime
}

// SnoozeNotificationParams contains the parameters for snoozing a notification
type SnoozeNotificationParams struct {
	GithubID     string
	SnoozedUntil sql.NullTime
}

// UpsertTagParams contains the parameters for upserting a tag
type UpsertTagParams struct {
	Name        string
	Slug        string
	Color       sql.NullString
	Description sql.NullString
}

// UpdateTagParams contains the parameters for updating a tag
type UpdateTagParams struct {
	ID          string // UUID
	Name        string
	Slug        string
	Color       sql.NullString
	Description sql.NullString
}

// UpdateTagDisplayOrderParams contains the parameters for updating tag display order
type UpdateTagDisplayOrderParams struct {
	ID           string // UUID
	DisplayOrder int32
}

// ListTagsForEntityParams contains the parameters for listing tags for an entity
type ListTagsForEntityParams struct {
	EntityType string
	EntityID   int64
}

// AssignTagToEntityParams contains the parameters for assigning a tag to an entity
type AssignTagToEntityParams struct {
	TagID      string // UUID
	EntityType string
	EntityID   int64
}

// RemoveTagAssignmentParams contains the parameters for removing a tag assignment
type RemoveTagAssignmentParams struct {
	TagID      string // UUID
	EntityType string
	EntityID   int64
}

// CreateViewParams contains the parameters for creating a view
type CreateViewParams struct {
	Name        string
	Slug        string
	Description sql.NullString
	IsDefault   interface{}
	Icon        sql.NullString
	Query       sql.NullString
}

// UpdateViewParams contains the parameters for updating a view
type UpdateViewParams struct {
	ID          string // UUID
	Name        sql.NullString
	Slug        sql.NullString
	Description sql.NullString
	Icon        sql.NullString
	Query       sql.NullString
	IsDefault   sql.NullBool
}

// UpdateViewOrderParams contains the parameters for updating view display order
type UpdateViewOrderParams struct {
	ID           string // UUID
	DisplayOrder int32
}

// CreateRuleParams contains the parameters for creating a rule
type CreateRuleParams struct {
	Name         string
	Description  sql.NullString
	Query        sql.NullString
	ViewID       sql.NullString // UUID
	Enabled      bool
	Actions      []byte
	DisplayOrder int32
}

// UpdateRuleParams contains the parameters for updating a rule
type UpdateRuleParams struct {
	ID          string // UUID
	Name        sql.NullString
	Description sql.NullString
	Query       sql.NullString
	ClearQuery  sql.NullBool
	ViewID      sql.NullString // UUID
	ClearViewID sql.NullBool
	Enabled     sql.NullBool
	Actions     NullRawMessage
}

// UpdateRuleOrderParams contains the parameters for updating rule display order
type UpdateRuleOrderParams struct {
	ID           string // UUID
	DisplayOrder int32
}

// UpsertRepositoryParams contains the parameters for upserting a repository
type UpsertRepositoryParams struct {
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
	Archived       interface{}
	Disabled       sql.NullBool
	PushedAt       sql.NullTime
	CreatedAt      sql.NullTime
	UpdatedAt      sql.NullTime
	Raw            NullRawMessage
	OwnerAvatarURL sql.NullString
	OwnerHTMLURL   sql.NullString
}

// UpsertPullRequestParams contains the parameters for upserting a pull request
type UpsertPullRequestParams struct {
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

// GetSyncStateRow contains the result of getting sync state
type GetSyncStateRow struct {
	ID                         int64
	UserID                     string
	LastSuccessfulPoll         sql.NullTime
	LastNotificationEtag       sql.NullString
	CreatedAt                  interface{}
	UpdatedAt                  interface{}
	LatestNotificationAt       sql.NullTime
	InitialSyncCompletedAt     sql.NullTime
	OldestNotificationSyncedAt sql.NullTime
}

// UpsertSyncStateParams contains the parameters for upserting sync state
type UpsertSyncStateParams struct {
	LastSuccessfulPoll         sql.NullTime
	LastNotificationEtag       sql.NullString
	LatestNotificationAt       sql.NullTime
	InitialSyncCompletedAt     sql.NullTime
	OldestNotificationSyncedAt sql.NullTime
}

// UpsertSyncStateRow contains the result of upserting sync state
type UpsertSyncStateRow struct {
	ID                         int64
	UserID                     string
	LastSuccessfulPoll         sql.NullTime
	LastNotificationEtag       sql.NullString
	CreatedAt                  interface{}
	UpdatedAt                  interface{}
	LatestNotificationAt       sql.NullTime
	InitialSyncCompletedAt     sql.NullTime
	OldestNotificationSyncedAt sql.NullTime
}

// UpsertNotificationParams contains the parameters for upserting a notification
type UpsertNotificationParams struct {
	GithubID                string
	RepositoryID            int64
	PullRequestID           sql.NullInt64
	SubjectType             string
	SubjectTitle            string
	SubjectURL              sql.NullString
	SubjectLatestCommentURL sql.NullString
	Reason                  sql.NullString
	GithubUnread            sql.NullBool
	GithubUpdatedAt         sql.NullTime
	GithubLastReadAt        sql.NullTime
	GithubURL               sql.NullString
	GithubSubscriptionURL   sql.NullString
	Payload                 NullRawMessage
	SubjectRaw              NullRawMessage
	SubjectFetchedAt        sql.NullTime
	AuthorLogin             sql.NullString
	AuthorID                sql.NullInt64
	SubjectNumber           sql.NullInt32
	SubjectState            sql.NullString
	SubjectMerged           sql.NullBool
	SubjectStateReason      sql.NullString
}

// UpdateNotificationSubjectParams contains the parameters for updating notification subject
type UpdateNotificationSubjectParams struct {
	GithubID           string
	SubjectRaw         NullRawMessage
	SubjectFetchedAt   sql.NullTime
	PullRequestID      sql.NullInt64
	SubjectNumber      sql.NullInt32
	SubjectState       sql.NullString
	SubjectMerged      sql.NullBool
	SubjectStateReason sql.NullString
}

// UpdateUserGitHubIdentityParams contains the parameters for updating user's GitHub identity
type UpdateUserGitHubIdentityParams struct {
	GithubUserID   sql.NullString
	GithubUsername sql.NullString
}
