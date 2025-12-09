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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/octobud-hq/octobud/backend/integration/client"
	"github.com/octobud-hq/octobud/backend/integration/fixtures"
	"github.com/octobud-hq/octobud/backend/integration/testserver"
)

func TestSnooze_SetsEffectiveSortDate(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a repository and notification
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(time.Now().Add(-2*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Record the original effective_sort_date
		originalNotif := c.GetNotification(t, notif.GithubID)
		originalSortDate := originalNotif.Notification.EffectiveSortDate

		// Snooze the notification for 1 hour from now
		snoozedUntil := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second)
		result := c.SnoozeNotification(t, notif.GithubID, snoozedUntil)

		// Verify snoozed_until is set
		require.NotNil(t, result.Notification.SnoozedUntil)
		require.WithinDuration(t, snoozedUntil, *result.Notification.SnoozedUntil, 2*time.Second)

		// Verify effective_sort_date is set to snoozed_until (not the original date)
		require.NotEqual(t, originalSortDate, result.Notification.EffectiveSortDate)
		require.WithinDuration(t, snoozedUntil, result.Notification.EffectiveSortDate, 2*time.Second)

		// Verify snoozed_at is set
		require.NotNil(t, result.Notification.SnoozedAt)
	})
}

func TestUnsnooze_RestoresEffectiveSortDate(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a repository and notification with a known github_updated_at
		githubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			Build(t, ctx, ts.Store, userID)

		// Snooze the notification
		snoozedUntil := time.Now().Add(1 * time.Hour).UTC()
		c.SnoozeNotification(t, notif.GithubID, snoozedUntil)

		// Verify it's snoozed
		snoozedNotif := c.GetNotification(t, notif.GithubID)
		require.NotNil(t, snoozedNotif.Notification.SnoozedUntil)

		// Unsnooze the notification
		result := c.UnsnoozeNotification(t, notif.GithubID)

		// Verify snoozed_until is cleared
		require.Nil(t, result.Notification.SnoozedUntil)
		require.Nil(t, result.Notification.SnoozedAt)

		// Verify effective_sort_date is restored to github_updated_at
		require.WithinDuration(t, githubUpdatedAt, result.Notification.EffectiveSortDate, 2*time.Second)
	})
}

func TestArchive_ClearsSnooze(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a snoozed notification
		githubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Verify it's snoozed
		snoozedNotif := c.GetNotification(t, notif.GithubID)
		require.NotNil(t, snoozedNotif.Notification.SnoozedUntil)

		// Archive the notification
		result := c.ArchiveNotification(t, notif.GithubID)

		// Verify it's archived
		require.True(t, result.Notification.Archived)

		// Verify snooze is cleared
		require.Nil(t, result.Notification.SnoozedUntil)
		require.Nil(t, result.Notification.SnoozedAt)

		// Verify effective_sort_date is restored to github_updated_at
		require.WithinDuration(t, githubUpdatedAt, result.Notification.EffectiveSortDate, 2*time.Second)
	})
}

func TestMute_ClearsSnooze(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create a snoozed notification
		githubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Verify it's snoozed
		snoozedNotif := c.GetNotification(t, notif.GithubID)
		require.NotNil(t, snoozedNotif.Notification.SnoozedUntil)

		// Mute the notification
		result := c.MuteNotification(t, notif.GithubID)

		// Verify it's muted
		require.True(t, result.Notification.Muted)

		// Verify snooze is cleared
		require.Nil(t, result.Notification.SnoozedUntil)
		require.Nil(t, result.Notification.SnoozedAt)

		// Verify effective_sort_date is restored to github_updated_at
		require.WithinDuration(t, githubUpdatedAt, result.Notification.EffectiveSortDate, 2*time.Second)
	})
}

func TestBulkSnooze_SetsEffectiveSortDate(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create multiple notifications
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif1 := fixtures.NewNotification(repo.ID).Build(t, ctx, ts.Store, userID)
		notif2 := fixtures.NewNotification(repo.ID).Build(t, ctx, ts.Store, userID)

		// Bulk snooze
		snoozedUntil := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
		result := c.BulkSnooze(t, client.BulkSnoozeRequest{
			GithubIDs:    []string{notif1.GithubID, notif2.GithubID},
			SnoozedUntil: snoozedUntil,
		})

		require.Equal(t, 2, result.Count)

		// Verify both notifications are snoozed with correct effective_sort_date
		n1 := c.GetNotification(t, notif1.GithubID)
		n2 := c.GetNotification(t, notif2.GithubID)

		require.NotNil(t, n1.Notification.SnoozedUntil)
		require.NotNil(t, n2.Notification.SnoozedUntil)
		require.WithinDuration(t, snoozedUntil, n1.Notification.EffectiveSortDate, 2*time.Second)
		require.WithinDuration(t, snoozedUntil, n2.Notification.EffectiveSortDate, 2*time.Second)
	})
}

func TestBulkArchive_ClearsSnooze(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create multiple snoozed notifications
		githubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif1 := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)
		notif2 := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Bulk archive
		result := c.BulkArchive(t, []string{notif1.GithubID, notif2.GithubID}, "")
		require.Equal(t, 2, result.Count)

		// Verify both notifications have snooze cleared
		n1 := c.GetNotification(t, notif1.GithubID)
		n2 := c.GetNotification(t, notif2.GithubID)

		require.True(t, n1.Notification.Archived)
		require.True(t, n2.Notification.Archived)
		require.Nil(t, n1.Notification.SnoozedUntil)
		require.Nil(t, n2.Notification.SnoozedUntil)
		require.WithinDuration(t, githubUpdatedAt, n1.Notification.EffectiveSortDate, 2*time.Second)
		require.WithinDuration(t, githubUpdatedAt, n2.Notification.EffectiveSortDate, 2*time.Second)
	})
}

func TestBulkUnsnooze_RestoresEffectiveSortDate(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create multiple snoozed notifications
		githubUpdatedAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)
		notif1 := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)
		notif2 := fixtures.NewNotification(repo.ID).
			WithGithubUpdatedAt(githubUpdatedAt).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Bulk unsnooze
		result := c.BulkUnsnooze(t, []string{notif1.GithubID, notif2.GithubID}, "")
		require.Equal(t, 2, result.Count)

		// Verify both notifications have snooze cleared and effective_sort_date restored
		n1 := c.GetNotification(t, notif1.GithubID)
		n2 := c.GetNotification(t, notif2.GithubID)

		require.Nil(t, n1.Notification.SnoozedUntil)
		require.Nil(t, n2.Notification.SnoozedUntil)
		require.WithinDuration(t, githubUpdatedAt, n1.Notification.EffectiveSortDate, 2*time.Second)
		require.WithinDuration(t, githubUpdatedAt, n2.Notification.EffectiveSortDate, 2*time.Second)
	})
}
