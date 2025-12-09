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

// Package sqlite provides a SQLite implementation of the db.Store interface.
// It wraps the SQLC-generated Queries and converts SQLite types to db.* types.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Store implements db.Store for SQLite by wrapping sqlite.Queries
// and converting SQLite types to db.* types.
type Store struct {
	q      *Queries
	dbConn *sql.DB
}

// NewStore creates a new SQLite store.
func NewStore(database *sql.DB) *Store {
	return &Store{
		q:      New(database),
		dbConn: database,
	}
}

func init() {
	// Register the SQLite store factory with the db package
	db.RegisterSQLiteStore(func(database *sql.DB) db.Store {
		return NewStore(database)
	})
}

// Ensure Store implements db.Store at compile time
var _ db.Store = (*Store)(nil)

// --- Type conversion helpers ---

// parseTime converts a SQLite TEXT timestamp to time.Time
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try RFC3339 first (ISO 8601 with timezone)
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t
	}
	// Try SQLite datetime format
	t, err = time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		return t
	}
	return time.Time{}
}

// parseNullTime converts a nullable SQLite TEXT timestamp to sql.NullTime
func parseNullTime(ns sql.NullString) sql.NullTime {
	if !ns.Valid || ns.String == "" {
		return sql.NullTime{}
	}
	t := parseTime(ns.String)
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// formatTime formats a time.Time for SQLite storage
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// formatNullTime formats a sql.NullTime for SQLite storage
func formatNullTime(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(nt.Time), Valid: true}
}

// toBool converts SQLite int64 (0/1) to bool
func toBool(i int64) bool {
	return i != 0
}

// toNullBool converts SQLite nullable int64 to sql.NullBool
func toNullBool(ni sql.NullInt64) sql.NullBool {
	if !ni.Valid {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: ni.Int64 != 0, Valid: true}
}

// fromNullBool converts sql.NullBool to SQLite nullable int64
func fromNullBool(nb sql.NullBool) sql.NullInt64 {
	if !nb.Valid {
		return sql.NullInt64{}
	}
	val := int64(0)
	if nb.Bool {
		val = 1
	}
	return sql.NullInt64{Int64: val, Valid: true}
}

// toNullRawMessage converts SQLite nullable TEXT to db.NullRawMessage
func toNullRawMessage(ns sql.NullString) db.NullRawMessage {
	if !ns.Valid || ns.String == "" {
		return db.NullRawMessage{}
	}
	return db.NullRawMessage{
		RawMessage: json.RawMessage(ns.String),
		Valid:      true,
	}
}

// fromNullRawMessage converts db.NullRawMessage to SQLite nullable TEXT
func fromNullRawMessage(nrm db.NullRawMessage) sql.NullString {
	if !nrm.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: string(nrm.RawMessage), Valid: true}
}

// toRawMessage converts SQLite TEXT to json.RawMessage
func toRawMessage(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}

// toNullInt32 converts sql.NullInt64 to sql.NullInt32
func toNullInt32(ni sql.NullInt64) sql.NullInt32 {
	if !ni.Valid {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(ni.Int64), Valid: true}
}

// fromNullInt32 converts sql.NullInt32 to sql.NullInt64
func fromNullInt32(ni sql.NullInt32) sql.NullInt64 {
	if !ni.Valid {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(ni.Int32), Valid: true}
}

// --- Notification type conversion ---

func (s *Store) toDBNotification(
	ctx context.Context,
	userID string,
	n Notification,
) db.Notification {
	// Get tag IDs from junction table
	tagIDs, _ := s.getNotificationTagIDs(ctx, userID, n.ID)

	return db.Notification{
		ID:                      n.ID,
		UserID:                  n.UserID,
		GithubID:                n.GithubID,
		RepositoryID:            n.RepositoryID,
		PullRequestID:           n.PullRequestID,
		SubjectType:             n.SubjectType,
		SubjectTitle:            n.SubjectTitle,
		SubjectURL:              n.SubjectUrl,
		SubjectLatestCommentURL: n.SubjectLatestCommentUrl,
		Reason:                  n.Reason,
		Archived:                toBool(n.Archived),
		GithubUnread:            toNullBool(n.GithubUnread),
		GithubUpdatedAt:         parseNullTime(n.GithubUpdatedAt),
		GithubLastReadAt:        parseNullTime(n.GithubLastReadAt),
		GithubURL:               n.GithubUrl,
		GithubSubscriptionURL:   n.GithubSubscriptionUrl,
		ImportedAt:              parseTime(n.ImportedAt),
		Payload:                 toNullRawMessage(n.Payload),
		SubjectRaw:              toNullRawMessage(n.SubjectRaw),
		SubjectFetchedAt:        parseNullTime(n.SubjectFetchedAt),
		AuthorLogin:             n.AuthorLogin,
		AuthorID:                n.AuthorID,
		IsRead:                  toBool(n.IsRead),
		Muted:                   toBool(n.Muted),
		SnoozedUntil:            parseNullTime(n.SnoozedUntil),
		EffectiveSortDate:       parseTime(n.EffectiveSortDate),
		SnoozedAt:               parseNullTime(n.SnoozedAt),
		Starred:                 toBool(n.Starred),
		Filtered:                toBool(n.Filtered),
		TagIDs:                  tagIDs,
		SubjectNumber:           toNullInt32(n.SubjectNumber),
		SubjectState:            n.SubjectState,
		SubjectMerged:           toNullBool(n.SubjectMerged),
		SubjectStateReason:      n.SubjectStateReason,
	}
}

func (s *Store) getNotificationTagIDs(
	ctx context.Context,
	userID string,
	notificationID int64,
) ([]string, error) {
	tagIDs, err := db.RetryOnBusy(ctx, func() ([]string, error) {
		return s.q.GetNotificationTagIDs(ctx, GetNotificationTagIDsParams{
			UserID:   userID,
			EntityID: notificationID,
		})
	})
	if err != nil {
		return nil, err
	}
	return tagIDs, nil
}

// --- Repository type conversion ---

func toDBRepository(r Repository) db.Repository {
	return db.Repository{
		ID:             r.ID,
		UserID:         r.UserID,
		GithubID:       r.GithubID,
		NodeID:         r.NodeID,
		Name:           r.Name,
		FullName:       r.FullName,
		OwnerLogin:     r.OwnerLogin,
		OwnerID:        r.OwnerID,
		Private:        toNullBool(r.Private),
		Description:    r.Description,
		HTMLURL:        r.HtmlUrl,
		Fork:           toNullBool(r.Fork),
		Visibility:     r.Visibility,
		DefaultBranch:  r.DefaultBranch,
		Archived:       toNullBool(r.Archived),
		Disabled:       toNullBool(r.Disabled),
		PushedAt:       parseNullTime(r.PushedAt),
		CreatedAt:      parseNullTime(r.CreatedAt),
		UpdatedAt:      parseNullTime(r.UpdatedAt),
		Raw:            toNullRawMessage(r.Raw),
		OwnerAvatarURL: r.OwnerAvatarUrl,
		OwnerHTMLURL:   r.OwnerHtmlUrl,
	}
}

// --- Tag type conversion ---

func toDBTag(t Tag) db.Tag {
	return db.Tag{
		ID:           t.ID,
		UserID:       t.UserID,
		Name:         t.Name,
		Color:        t.Color,
		Description:  t.Description,
		CreatedAt:    parseTime(t.CreatedAt),
		DisplayOrder: int32(t.DisplayOrder),
		Slug:         t.Slug,
	}
}

func toDBTagAssignment(ta TagAssignment) db.TagAssignment {
	return db.TagAssignment{
		ID:         ta.ID,
		UserID:     ta.UserID,
		TagID:      ta.TagID,
		EntityType: ta.EntityType,
		EntityID:   ta.EntityID,
		CreatedAt:  parseTime(ta.CreatedAt),
	}
}

// --- View type conversion ---

func toDBView(v View) db.View {
	return db.View{
		ID:           v.ID,
		UserID:       v.UserID,
		Name:         v.Name,
		Description:  v.Description,
		IsDefault:    toBool(v.IsDefault),
		CreatedAt:    parseTime(v.CreatedAt),
		Icon:         v.Icon,
		Slug:         v.Slug,
		Query:        v.Query,
		DisplayOrder: int32(v.DisplayOrder),
	}
}

// --- Rule type conversion ---

func toDBRule(r Rule) db.Rule {
	return db.Rule{
		ID:           r.ID,
		UserID:       r.UserID,
		Name:         r.Name,
		Description:  r.Description,
		Query:        r.Query,
		Enabled:      toBool(r.Enabled),
		Actions:      toRawMessage(r.Actions),
		DisplayOrder: int32(r.DisplayOrder),
		CreatedAt:    parseTime(r.CreatedAt),
		UpdatedAt:    parseTime(r.UpdatedAt),
		ViewID:       r.ViewID,
	}
}

// --- User type conversion ---

func toDBUser(u User) db.User {
	return db.User{
		ID:                   u.ID,
		GithubUserID:         u.GithubUserID,
		GithubUsername:       u.GithubUsername,
		GithubTokenEncrypted: u.GithubTokenEncrypted,
		CreatedAt:            parseTime(u.CreatedAt),
		UpdatedAt:            parseTime(u.UpdatedAt),
		SyncSettings:         toNullRawMessage(u.SyncSettings),
		RetentionSettings:    toNullRawMessage(u.RetentionSettings),
		UpdateSettings:       toNullRawMessage(u.UpdateSettings),
		MutedUntil:           parseNullTime(u.MutedUntil),
	}
}

// --- PullRequest type conversion ---

func toDBPullRequest(pr PullRequest) db.PullRequest {
	return db.PullRequest{
		ID:           pr.ID,
		UserID:       pr.UserID,
		RepositoryID: pr.RepositoryID,
		GithubID:     pr.GithubID,
		NodeID:       pr.NodeID,
		Number:       int32(pr.Number),
		Title:        pr.Title,
		State:        pr.State,
		Draft:        toNullBool(pr.Draft),
		Merged:       toNullBool(pr.Merged),
		AuthorLogin:  pr.AuthorLogin,
		AuthorID:     pr.AuthorID,
		CreatedAt:    parseNullTime(pr.CreatedAt),
		UpdatedAt:    parseNullTime(pr.UpdatedAt),
		ClosedAt:     parseNullTime(pr.ClosedAt),
		MergedAt:     parseNullTime(pr.MergedAt),
		Raw:          toNullRawMessage(pr.Raw),
	}
}

// --- SyncState type conversion ---

func toDBGetSyncStateRow(ss SyncState) db.GetSyncStateRow {
	return db.GetSyncStateRow{
		ID:                         ss.ID,
		UserID:                     ss.UserID,
		LastSuccessfulPoll:         parseNullTime(ss.LastSuccessfulPoll),
		LastNotificationEtag:       ss.LastNotificationEtag,
		CreatedAt:                  parseTime(ss.CreatedAt),
		UpdatedAt:                  parseTime(ss.UpdatedAt),
		LatestNotificationAt:       parseNullTime(ss.LatestNotificationAt),
		InitialSyncCompletedAt:     parseNullTime(ss.InitialSyncCompletedAt),
		OldestNotificationSyncedAt: parseNullTime(ss.OldestNotificationSyncedAt),
	}
}

func toDBUpsertSyncStateRow(ss SyncState) db.UpsertSyncStateRow {
	return db.UpsertSyncStateRow{
		ID:                         ss.ID,
		UserID:                     ss.UserID,
		LastSuccessfulPoll:         parseNullTime(ss.LastSuccessfulPoll),
		LastNotificationEtag:       ss.LastNotificationEtag,
		CreatedAt:                  parseTime(ss.CreatedAt),
		UpdatedAt:                  parseTime(ss.UpdatedAt),
		LatestNotificationAt:       parseNullTime(ss.LatestNotificationAt),
		InitialSyncCompletedAt:     parseNullTime(ss.InitialSyncCompletedAt),
		OldestNotificationSyncedAt: parseNullTime(ss.OldestNotificationSyncedAt),
	}
}
