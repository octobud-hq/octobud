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

func TestQuery_InInbox(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create notifications in various states
		inboxNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("inbox-notif").
			Build(t, ctx, ts.Store, userID)

		// Archived - should be excluded
		fixtures.NewNotification(repo.ID).
			WithGithubID("archived-notif").
			WithArchived(true).
			Build(t, ctx, ts.Store, userID)

		// Snoozed (future) - should be excluded
		fixtures.NewNotification(repo.ID).
			WithGithubID("snoozed-notif").
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Muted - should be excluded
		fixtures.NewNotification(repo.ID).
			WithGithubID("muted-notif").
			WithMuted(true).
			Build(t, ctx, ts.Store, userID)

		// Filtered - should be excluded
		fixtures.NewNotification(repo.ID).
			WithGithubID("filtered-notif").
			WithFiltered(true).
			Build(t, ctx, ts.Store, userID)

		// Query inbox
		result := c.ListNotifications(t, "in:inbox", 1, 100)

		// Should only see the inbox notification
		require.Equal(t, int64(1), result.Total)
		require.Len(t, result.Notifications, 1)
		require.Equal(t, inboxNotif.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_InSnoozed(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create a snoozed notification (future)
		snoozedNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("snoozed-notif").
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Create a regular inbox notification
		fixtures.NewNotification(repo.ID).
			WithGithubID("inbox-notif").
			Build(t, ctx, ts.Store, userID)

		// Create an archived snoozed notification (should be excluded)
		fixtures.NewNotification(repo.ID).
			WithGithubID("archived-snoozed-notif").
			WithArchived(true).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Create a muted snoozed notification (should be excluded)
		fixtures.NewNotification(repo.ID).
			WithGithubID("muted-snoozed-notif").
			WithMuted(true).
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Query snoozed
		result := c.ListNotifications(t, "in:snoozed", 1, 100)

		// Should only see the snoozed notification
		require.Equal(t, int64(1), result.Total)
		require.Len(t, result.Notifications, 1)
		require.Equal(t, snoozedNotif.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_InArchive(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create an archived notification
		archivedNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("archived-notif").
			WithArchived(true).
			Build(t, ctx, ts.Store, userID)

		// Create a regular inbox notification
		fixtures.NewNotification(repo.ID).
			WithGithubID("inbox-notif").
			Build(t, ctx, ts.Store, userID)

		// Create an archived + muted notification (should be excluded)
		fixtures.NewNotification(repo.ID).
			WithGithubID("archived-muted-notif").
			WithArchived(true).
			WithMuted(true).
			Build(t, ctx, ts.Store, userID)

		// Query archive
		result := c.ListNotifications(t, "in:archive", 1, 100)

		// Should only see the archived notification
		require.Equal(t, int64(1), result.Total)
		require.Len(t, result.Notifications, 1)
		require.Equal(t, archivedNotif.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_InFiltered(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create a filtered notification
		filteredNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("filtered-notif").
			WithFiltered(true).
			Build(t, ctx, ts.Store, userID)

		// Create a regular inbox notification
		fixtures.NewNotification(repo.ID).
			WithGithubID("inbox-notif").
			Build(t, ctx, ts.Store, userID)

		// Create a filtered + archived notification (should be excluded)
		fixtures.NewNotification(repo.ID).
			WithGithubID("filtered-archived-notif").
			WithFiltered(true).
			WithArchived(true).
			Build(t, ctx, ts.Store, userID)

		// Create a filtered + muted notification (should be excluded)
		fixtures.NewNotification(repo.ID).
			WithGithubID("filtered-muted-notif").
			WithFiltered(true).
			WithMuted(true).
			Build(t, ctx, ts.Store, userID)

		// Query filtered
		result := c.ListNotifications(t, "in:filtered", 1, 100)

		// Should only see the filtered notification
		require.Equal(t, int64(1), result.Total)
		require.Len(t, result.Notifications, 1)
		require.Equal(t, filteredNotif.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_InAnywhere(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create notifications in various states
		fixtures.NewNotification(repo.ID).
			WithGithubID("inbox-notif").
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo.ID).
			WithGithubID("archived-notif").
			WithArchived(true).
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo.ID).
			WithGithubID("snoozed-notif").
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo.ID).
			WithGithubID("muted-notif").
			WithMuted(true).
			Build(t, ctx, ts.Store, userID)

		// Query anywhere
		result := c.ListNotifications(t, "in:anywhere", 1, 100)

		// Should see all notifications
		require.Equal(t, int64(4), result.Total)
	})
}

func TestQuery_SnoozedExpiresReturnsToInbox(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create a notification with an expired snooze (in the past)
		expiredSnoozedNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("expired-snoozed-notif").
			WithSnoozedUntil(time.Now().Add(-1*time.Hour)). // Expired
			Build(t, ctx, ts.Store, userID)

		// Create a notification with a future snooze
		fixtures.NewNotification(repo.ID).
			WithGithubID("future-snoozed-notif").
			WithSnoozedUntil(time.Now().Add(1*time.Hour)).
			Build(t, ctx, ts.Store, userID)

		// Query inbox - expired snooze should appear
		inboxResult := c.ListNotifications(t, "in:inbox", 1, 100)
		require.Equal(t, int64(1), inboxResult.Total)
		require.Equal(t, expiredSnoozedNotif.GithubID, inboxResult.Notifications[0].GithubID)

		// Query snoozed - expired snooze should NOT appear
		snoozedResult := c.ListNotifications(t, "in:snoozed", 1, 100)
		require.Equal(t, int64(1), snoozedResult.Total)
		require.Equal(t, "future-snoozed-notif", snoozedResult.Notifications[0].GithubID)
	})
}

func TestQuery_TagFilter(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().Build(t, ctx, ts.Store, userID)

		// Create a tag
		tag := fixtures.NewTag().
			WithName("Bug").
			WithSlug("bug").
			Build(t, ctx, ts.Store, userID)

		// Create notifications
		taggedNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("tagged-notif").
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo.ID).
			WithGithubID("untagged-notif").
			Build(t, ctx, ts.Store, userID)

		// Assign tag to notification
		fixtures.AssignTag(t, ctx, ts.Store, userID, tag.ID, taggedNotif.ID)

		// Query by tag
		result := c.ListNotifications(t, "tags:bug in:anywhere", 1, 100)

		// Should only see the tagged notification
		require.Equal(t, int64(1), result.Total)
		require.Equal(t, taggedNotif.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_RepoFilter(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create two repositories
		repo1 := fixtures.NewRepository().
			WithFullName("org/repo-one").
			Build(t, ctx, ts.Store, userID)

		repo2 := fixtures.NewRepository().
			WithFullName("org/repo-two").
			Build(t, ctx, ts.Store, userID)

		// Create notifications in each repo
		notif1 := fixtures.NewNotification(repo1.ID).
			WithGithubID("repo-one-notif").
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo2.ID).
			WithGithubID("repo-two-notif").
			Build(t, ctx, ts.Store, userID)

		// Query by repo
		result := c.ListNotifications(t, "repo:repo-one in:anywhere", 1, 100)

		// Should only see the notification from repo-one
		require.Equal(t, int64(1), result.Total)
		require.Equal(t, notif1.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_OrgFilter(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		// Create repositories in different orgs
		repo1 := fixtures.NewRepository().
			WithFullName("org-one/repo").
			Build(t, ctx, ts.Store, userID)

		repo2 := fixtures.NewRepository().
			WithFullName("org-two/repo").
			Build(t, ctx, ts.Store, userID)

		// Create notifications
		notif1 := fixtures.NewNotification(repo1.ID).
			WithGithubID("org-one-notif").
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo2.ID).
			WithGithubID("org-two-notif").
			Build(t, ctx, ts.Store, userID)

		// Query by org
		result := c.ListNotifications(t, "org:org-one in:anywhere", 1, 100)

		// Should only see notifications from org-one
		require.Equal(t, int64(1), result.Total)
		require.Equal(t, notif1.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_CombinedFilters(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().
			WithFullName("test-org/test-repo").
			Build(t, ctx, ts.Store, userID)

		// Create notifications with different properties
		matchingNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("matching-notif").
			WithSubjectType("PullRequest").
			WithReason("review_requested").
			Build(t, ctx, ts.Store, userID)

		// Wrong type
		fixtures.NewNotification(repo.ID).
			WithGithubID("wrong-type-notif").
			WithSubjectType("Issue").
			WithReason("review_requested").
			Build(t, ctx, ts.Store, userID)

		// Wrong reason
		fixtures.NewNotification(repo.ID).
			WithGithubID("wrong-reason-notif").
			WithSubjectType("PullRequest").
			WithReason("subscribed").
			Build(t, ctx, ts.Store, userID)

		// Query with combined filters
		result := c.ListNotifications(t, "type:PullRequest reason:review_requested in:anywhere", 1, 100)

		// Should only see the matching notification
		require.Equal(t, int64(1), result.Total)
		require.Equal(t, matchingNotif.GithubID, result.Notifications[0].GithubID)
	})
}

func TestQuery_FreeTextSearch(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repo := fixtures.NewRepository().
			WithFullName("test-org/test-repo").
			Build(t, ctx, ts.Store, userID)

		// Create notifications with different titles
		matchingNotif := fixtures.NewNotification(repo.ID).
			WithGithubID("matching-notif").
			WithSubjectTitle("Fix important bug in login").
			Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repo.ID).
			WithGithubID("non-matching-notif").
			WithSubjectTitle("Add new feature").
			Build(t, ctx, ts.Store, userID)

		// Free text search
		result := c.ListNotifications(t, "bug in:anywhere", 1, 100)

		// Should only see the notification with "bug" in title
		require.Equal(t, int64(1), result.Total)
		require.Equal(t, matchingNotif.GithubID, result.Notifications[0].GithubID)
	})
}
