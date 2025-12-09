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

//go:build integration

// Package fixtures provides test data factories for integration tests.
package fixtures

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Counter for generating unique IDs
var counter int64

func nextID() int64 {
	return atomic.AddInt64(&counter, 1)
}

// RepositoryBuilder builds test repositories.
type RepositoryBuilder struct {
	githubID   int64
	name       string
	fullName   string
	ownerLogin string
}

// NewRepository creates a new repository builder with defaults.
func NewRepository() *RepositoryBuilder {
	id := nextID()
	return &RepositoryBuilder{
		githubID:   id,
		name:       fmt.Sprintf("test-repo-%d", id),
		fullName:   fmt.Sprintf("test-org/test-repo-%d", id),
		ownerLogin: "test-org",
	}
}

// WithGithubID sets the GitHub ID.
func (b *RepositoryBuilder) WithGithubID(id int64) *RepositoryBuilder {
	b.githubID = id
	return b
}

// WithName sets the repository name.
func (b *RepositoryBuilder) WithName(name string) *RepositoryBuilder {
	b.name = name
	return b
}

// WithFullName sets the full repository name.
func (b *RepositoryBuilder) WithFullName(fullName string) *RepositoryBuilder {
	b.fullName = fullName
	return b
}

// WithOwnerLogin sets the owner login.
func (b *RepositoryBuilder) WithOwnerLogin(ownerLogin string) *RepositoryBuilder {
	b.ownerLogin = ownerLogin
	return b
}

// Build creates the repository in the database.
func (b *RepositoryBuilder) Build(t *testing.T, ctx context.Context, store db.Store, userID string) db.Repository {
	t.Helper()

	repo, err := store.UpsertRepository(ctx, userID, db.UpsertRepositoryParams{
		GithubID:   sql.NullInt64{Int64: b.githubID, Valid: true},
		Name:       b.name,
		FullName:   b.fullName,
		OwnerLogin: sql.NullString{String: b.ownerLogin, Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	return repo
}

// NotificationBuilder builds test notifications.
type NotificationBuilder struct {
	githubID        string
	repositoryID    int64
	subjectType     string
	subjectTitle    string
	reason          string
	archived        bool
	isRead          bool
	muted           bool
	starred         bool
	filtered        bool
	snoozedUntil    sql.NullTime
	githubUpdatedAt sql.NullTime
}

// NewNotification creates a new notification builder with defaults.
func NewNotification(repositoryID int64) *NotificationBuilder {
	id := nextID()
	return &NotificationBuilder{
		githubID:        fmt.Sprintf("test-notif-%d", id),
		repositoryID:    repositoryID,
		subjectType:     "PullRequest",
		subjectTitle:    fmt.Sprintf("Test Notification %d", id),
		reason:          "subscribed",
		githubUpdatedAt: sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true},
	}
}

// WithGithubID sets the GitHub ID.
func (b *NotificationBuilder) WithGithubID(id string) *NotificationBuilder {
	b.githubID = id
	return b
}

// WithSubjectType sets the subject type.
func (b *NotificationBuilder) WithSubjectType(t string) *NotificationBuilder {
	b.subjectType = t
	return b
}

// WithSubjectTitle sets the subject title.
func (b *NotificationBuilder) WithSubjectTitle(title string) *NotificationBuilder {
	b.subjectTitle = title
	return b
}

// WithReason sets the reason.
func (b *NotificationBuilder) WithReason(reason string) *NotificationBuilder {
	b.reason = reason
	return b
}

// WithArchived sets the archived flag.
func (b *NotificationBuilder) WithArchived(archived bool) *NotificationBuilder {
	b.archived = archived
	return b
}

// WithIsRead sets the is_read flag.
func (b *NotificationBuilder) WithIsRead(isRead bool) *NotificationBuilder {
	b.isRead = isRead
	return b
}

// WithMuted sets the muted flag.
func (b *NotificationBuilder) WithMuted(muted bool) *NotificationBuilder {
	b.muted = muted
	return b
}

// WithStarred sets the starred flag.
func (b *NotificationBuilder) WithStarred(starred bool) *NotificationBuilder {
	b.starred = starred
	return b
}

// WithFiltered sets the filtered flag.
func (b *NotificationBuilder) WithFiltered(filtered bool) *NotificationBuilder {
	b.filtered = filtered
	return b
}

// WithSnoozedUntil sets the snooze time.
func (b *NotificationBuilder) WithSnoozedUntil(t time.Time) *NotificationBuilder {
	b.snoozedUntil = sql.NullTime{Time: t, Valid: true}
	return b
}

// WithGithubUpdatedAt sets the github_updated_at time.
func (b *NotificationBuilder) WithGithubUpdatedAt(t time.Time) *NotificationBuilder {
	b.githubUpdatedAt = sql.NullTime{Time: t, Valid: true}
	return b
}

// Build creates the notification in the database.
func (b *NotificationBuilder) Build(t *testing.T, ctx context.Context, store db.Store, userID string) db.Notification {
	t.Helper()

	// Create notification via upsert
	notif, err := store.UpsertNotification(ctx, userID, db.UpsertNotificationParams{
		GithubID:        b.githubID,
		RepositoryID:    b.repositoryID,
		SubjectType:     b.subjectType,
		SubjectTitle:    b.subjectTitle,
		Reason:          sql.NullString{String: b.reason, Valid: true},
		GithubUpdatedAt: b.githubUpdatedAt,
	})
	if err != nil {
		t.Fatalf("Failed to create notification: %v", err)
	}

	// Apply additional states if needed
	if b.archived {
		notif, err = store.ArchiveNotification(ctx, userID, b.githubID)
		if err != nil {
			t.Fatalf("Failed to archive notification: %v", err)
		}
	}

	if b.isRead {
		notif, err = store.MarkNotificationRead(ctx, userID, b.githubID)
		if err != nil {
			t.Fatalf("Failed to mark notification read: %v", err)
		}
	}

	if b.muted {
		notif, err = store.MuteNotification(ctx, userID, b.githubID)
		if err != nil {
			t.Fatalf("Failed to mute notification: %v", err)
		}
	}

	if b.starred {
		notif, err = store.StarNotification(ctx, userID, b.githubID)
		if err != nil {
			t.Fatalf("Failed to star notification: %v", err)
		}
	}

	if b.filtered {
		notif, err = store.MarkNotificationFiltered(ctx, userID, b.githubID)
		if err != nil {
			t.Fatalf("Failed to filter notification: %v", err)
		}
	}

	if b.snoozedUntil.Valid {
		notif, err = store.SnoozeNotification(ctx, userID, db.SnoozeNotificationParams{
			GithubID:     b.githubID,
			SnoozedUntil: b.snoozedUntil,
		})
		if err != nil {
			t.Fatalf("Failed to snooze notification: %v", err)
		}
	}

	return notif
}

// TagBuilder builds test tags.
type TagBuilder struct {
	name        string
	slug        string
	color       string
	description string
}

// NewTag creates a new tag builder with defaults.
func NewTag() *TagBuilder {
	id := nextID()
	return &TagBuilder{
		name:  fmt.Sprintf("Test Tag %d", id),
		slug:  fmt.Sprintf("test-tag-%d", id),
		color: "#FF5733",
	}
}

// WithName sets the tag name.
func (b *TagBuilder) WithName(name string) *TagBuilder {
	b.name = name
	return b
}

// WithSlug sets the tag slug.
func (b *TagBuilder) WithSlug(slug string) *TagBuilder {
	b.slug = slug
	return b
}

// WithColor sets the tag color.
func (b *TagBuilder) WithColor(color string) *TagBuilder {
	b.color = color
	return b
}

// WithDescription sets the tag description.
func (b *TagBuilder) WithDescription(desc string) *TagBuilder {
	b.description = desc
	return b
}

// Build creates the tag in the database.
func (b *TagBuilder) Build(t *testing.T, ctx context.Context, store db.Store, userID string) db.Tag {
	t.Helper()

	tag, err := store.UpsertTag(ctx, userID, db.UpsertTagParams{
		Name:        b.name,
		Slug:        b.slug,
		Color:       sql.NullString{String: b.color, Valid: true},
		Description: sql.NullString{String: b.description, Valid: b.description != ""},
	})
	if err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	return tag
}

// AssignTag assigns a tag to a notification.
func AssignTag(t *testing.T, ctx context.Context, store db.Store, userID string, tagID string, notificationID int64) {
	t.Helper()

	_, err := store.AssignTagToEntity(ctx, userID, db.AssignTagToEntityParams{
		TagID:      tagID,
		EntityType: "notification",
		EntityID:   notificationID,
	})
	if err != nil {
		t.Fatalf("Failed to assign tag: %v", err)
	}

	// Note: SQLite uses a junction table, so no need to update a denormalized tag_ids column
}
