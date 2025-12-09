//go:build test && integration

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

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/octobud-hq/octobud/backend/integration/client"
	"github.com/octobud-hq/octobud/backend/integration/fixtures"
	"github.com/octobud-hq/octobud/backend/integration/testserver"
	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Note: These tests simulate sync behavior by directly calling the store's UpsertNotification
// method, which is what the sync service uses. This tests the business logic embedded in
// the upsert SQL queries.

func TestUpsert_PreservesSnoozeOnUpdate(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a repository and notification
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		originalGithubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC()
		notif := fixtures.NewNotification(repo.ID).
			WithGithubID("test-notif").
			WithGithubUpdatedAt(originalGithubUpdatedAt).
			Build(t, ctx, ts.Store, userID)

		// Snooze the notification via API
		snoozedUntil := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second)
		c.SnoozeNotification(t, notif.GithubID, snoozedUntil)

		// Verify it's snoozed
		snoozedNotif := c.GetNotification(t, notif.GithubID)
		require.NotNil(t, snoozedNotif.Notification.SnoozedUntil)

		// Simulate a sync update with a new github_updated_at
		newGithubUpdatedAt := time.Now().UTC()
		_, err := ts.Store.UpsertNotification(ctx, userID, db.UpsertNotificationParams{
			GithubID:        notif.GithubID,
			RepositoryID:    repo.ID,
			SubjectType:     "PullRequest",
			SubjectTitle:    "Updated title from sync",
			GithubUpdatedAt: sql.NullTime{Time: newGithubUpdatedAt, Valid: true},
		})
		require.NoError(t, err)

		// Verify snooze is still preserved after sync update
		updatedNotif := c.GetNotification(t, notif.GithubID)
		require.NotNil(t, updatedNotif.Notification.SnoozedUntil, "Snooze should be preserved after sync update")
		require.WithinDuration(t, snoozedUntil, *updatedNotif.Notification.SnoozedUntil, 2*time.Second)

		// Verify effective_sort_date still reflects the snooze time (not the new github_updated_at)
		require.WithinDuration(t, snoozedUntil, updatedNotif.Notification.EffectiveSortDate, 2*time.Second)

		// Verify title was updated
		require.Equal(t, "Updated title from sync", updatedNotif.Notification.SubjectTitle)
	})
}

func TestUpsert_UpdatesEffectiveSortDateWhenNotSnoozed(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a repository and notification
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		originalGithubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC()
		notif := fixtures.NewNotification(repo.ID).
			WithGithubID("test-notif").
			WithGithubUpdatedAt(originalGithubUpdatedAt).
			Build(t, ctx, ts.Store, userID)

		// Get original effective_sort_date
		originalNotif := c.GetNotification(t, notif.GithubID)
		originalSortDate := originalNotif.Notification.EffectiveSortDate

		// Simulate a sync update with a new github_updated_at
		newGithubUpdatedAt := time.Now().UTC().Truncate(time.Second)
		_, err := ts.Store.UpsertNotification(ctx, userID, db.UpsertNotificationParams{
			GithubID:        notif.GithubID,
			RepositoryID:    repo.ID,
			SubjectType:     "PullRequest",
			SubjectTitle:    "Updated title",
			GithubUpdatedAt: sql.NullTime{Time: newGithubUpdatedAt, Valid: true},
		})
		require.NoError(t, err)

		// Verify effective_sort_date is updated to new github_updated_at
		updatedNotif := c.GetNotification(t, notif.GithubID)
		require.NotEqual(t, originalSortDate, updatedNotif.Notification.EffectiveSortDate)
		require.WithinDuration(t, newGithubUpdatedAt, updatedNotif.Notification.EffectiveSortDate, 2*time.Second)
	})
}

func TestUpsert_UnarchivesOnNewUpdate(t *testing.T) {
	// This test verifies that notifications are unarchived when github_updated_at changes
	// (and they're not muted).
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create an archived notification
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		originalGithubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC()
		notif := fixtures.NewNotification(repo.ID).
			WithGithubID("test-notif").
			WithGithubUpdatedAt(originalGithubUpdatedAt).
			WithArchived(true).
			Build(t, ctx, ts.Store, userID)

		// Verify it's archived
		archivedNotif := c.GetNotification(t, notif.GithubID)
		require.True(t, archivedNotif.Notification.Archived)

		// Simulate a sync update with a NEW github_updated_at (different from original)
		newGithubUpdatedAt := time.Now().UTC()
		_, err := ts.Store.UpsertNotification(ctx, userID, db.UpsertNotificationParams{
			GithubID:        notif.GithubID,
			RepositoryID:    repo.ID,
			SubjectType:     "PullRequest",
			SubjectTitle:    "New activity - updated title",
			GithubUpdatedAt: sql.NullTime{Time: newGithubUpdatedAt, Valid: true},
		})
		require.NoError(t, err)

		// Verify notification is unarchived (new activity detected)
		updatedNotif := c.GetNotification(t, notif.GithubID)
		require.False(t, updatedNotif.Notification.Archived, "Notification should be unarchived after new activity")
	})
}

func TestUpsert_MutedNotificationsIgnoreStatusUpdates(t *testing.T) {
	// This tests that muted notifications preserve their archived/read state
	// even when github_updated_at changes.
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a muted and archived notification
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		originalGithubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC()
		notif := fixtures.NewNotification(repo.ID).
			WithGithubID("test-notif").
			WithGithubUpdatedAt(originalGithubUpdatedAt).
			WithArchived(true).
			WithMuted(true).
			WithIsRead(true).
			Build(t, ctx, ts.Store, userID)

		// Verify initial state
		initial := c.GetNotification(t, notif.GithubID)
		require.True(t, initial.Notification.Archived)
		require.True(t, initial.Notification.Muted)
		require.True(t, initial.Notification.IsRead)

		// Simulate a sync update with a NEW github_updated_at
		newGithubUpdatedAt := time.Now().UTC()
		_, err := ts.Store.UpsertNotification(ctx, userID, db.UpsertNotificationParams{
			GithubID:        notif.GithubID,
			RepositoryID:    repo.ID,
			SubjectType:     "PullRequest",
			SubjectTitle:    "New activity",
			GithubUpdatedAt: sql.NullTime{Time: newGithubUpdatedAt, Valid: true},
		})
		require.NoError(t, err)

		// Verify muted notification preserves its archived/read state
		updatedNotif := c.GetNotification(t, notif.GithubID)
		require.True(t, updatedNotif.Notification.Muted, "Should still be muted")
		require.True(t, updatedNotif.Notification.Archived, "Muted notifications should stay archived")
		require.True(t, updatedNotif.Notification.IsRead, "Muted notifications should stay read")
	})
}

func TestUpsert_SameUpdateDoesNotUnarchive(t *testing.T) {
	// This tests that when github_updated_at is unchanged, archived status is preserved.
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create an archived notification
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		githubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		notif := fixtures.NewNotification(repo.ID).
			WithGithubID("test-notif").
			WithGithubUpdatedAt(githubUpdatedAt).
			WithArchived(true).
			Build(t, ctx, ts.Store, userID)

		// Verify it's archived
		archivedNotif := c.GetNotification(t, notif.GithubID)
		require.True(t, archivedNotif.Notification.Archived)

		// Simulate a sync update with the SAME github_updated_at
		_, err := ts.Store.UpsertNotification(ctx, userID, db.UpsertNotificationParams{
			GithubID:        notif.GithubID,
			RepositoryID:    repo.ID,
			SubjectType:     "PullRequest",
			SubjectTitle:    "Same title - no new activity",
			GithubUpdatedAt: sql.NullTime{Time: githubUpdatedAt, Valid: true},
		})
		require.NoError(t, err)

		// Verify notification stays archived (no new activity detected)
		updatedNotif := c.GetNotification(t, notif.GithubID)
		require.True(t, updatedNotif.Notification.Archived, "Notification should stay archived when github_updated_at is unchanged")
	})
}
