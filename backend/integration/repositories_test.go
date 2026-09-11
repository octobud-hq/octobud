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

//go:build test && integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/octobud-hq/octobud/backend/integration/client"
	"github.com/octobud-hq/octobud/backend/integration/fixtures"
	"github.com/octobud-hq/octobud/backend/integration/testserver"
)

func TestRepositoryCounts_GroupsByRepositoryUnderQuery(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repoA := fixtures.NewRepository().WithFullName("org/alpha").Build(t, ctx, ts.Store, userID)
		repoB := fixtures.NewRepository().WithFullName("org/beta").Build(t, ctx, ts.Store, userID)
		repoC := fixtures.NewRepository().WithFullName("org/gamma").Build(t, ctx, ts.Store, userID)

		// alpha: 3 in inbox (1 read), 1 archived
		fixtures.NewNotification(repoA.ID).WithGithubID("a1").Build(t, ctx, ts.Store, userID)
		fixtures.NewNotification(repoA.ID).WithGithubID("a2").Build(t, ctx, ts.Store, userID)
		fixtures.NewNotification(repoA.ID).WithGithubID("a3").WithIsRead(true).Build(t, ctx, ts.Store, userID)
		fixtures.NewNotification(repoA.ID).WithGithubID("a4").WithArchived(true).Build(t, ctx, ts.Store, userID)

		// beta: 1 in inbox
		fixtures.NewNotification(repoB.ID).WithGithubID("b1").Build(t, ctx, ts.Store, userID)

		// gamma: only archived, so absent from inbox counts
		fixtures.NewNotification(repoC.ID).WithGithubID("c1").WithArchived(true).Build(t, ctx, ts.Store, userID)

		inbox := c.ListRepositoryCounts(t, "in:inbox")
		require.Len(t, inbox.Repositories, 2)
		require.Equal(t, "org/alpha", inbox.Repositories[0].Repository.FullName)
		require.Equal(t, repoA.ID, inbox.Repositories[0].Repository.ID)
		require.Equal(t, int64(3), inbox.Repositories[0].Total)
		require.Equal(t, int64(2), inbox.Repositories[0].Unread)
		require.Equal(t, "org/beta", inbox.Repositories[1].Repository.FullName)
		require.Equal(t, int64(1), inbox.Repositories[1].Total)
		require.Equal(t, int64(1), inbox.Repositories[1].Unread)

		// Included repositories with no matches come last with zero counts; unknown ids are skipped.
		withInclude := c.ListRepositoryCounts(t, "in:inbox", repoC.ID, repoA.ID, 999999)
		require.Len(t, withInclude.Repositories, 3)
		require.Equal(t, "org/gamma", withInclude.Repositories[2].Repository.FullName)
		require.Equal(t, int64(0), withInclude.Repositories[2].Total)

		// A repo: term in the query joins repositories itself; the counts join must not clash.
		scoped := c.ListRepositoryCounts(t, "in:inbox repo:alpha")
		require.Len(t, scoped.Repositories, 1)
		require.Equal(t, "org/alpha", scoped.Repositories[0].Repository.FullName)

		// Query narrows the counts: only unread notifications.
		unread := c.ListRepositoryCounts(t, "in:inbox is:unread")
		require.Len(t, unread.Repositories, 2)
		require.Equal(t, int64(2), unread.Repositories[0].Total)

		// in:anywhere sees gamma too.
		anywhere := c.ListRepositoryCounts(t, "in:anywhere")
		require.Len(t, anywhere.Repositories, 3)
	})
}

func TestRepositoryFilter_ScopesListAndBulk(t *testing.T) {
	RunWithBackends(t, func(t *testing.T, ts *testserver.TestServer, c *client.Client) {
		ctx := context.Background()
		userID := ts.UserID

		repoA := fixtures.NewRepository().WithFullName("org/alpha").Build(t, ctx, ts.Store, userID)
		repoB := fixtures.NewRepository().WithFullName("org/beta").Build(t, ctx, ts.Store, userID)
		repoC := fixtures.NewRepository().WithFullName("org/gamma").Build(t, ctx, ts.Store, userID)

		fixtures.NewNotification(repoA.ID).WithGithubID("a1").Build(t, ctx, ts.Store, userID)
		fixtures.NewNotification(repoA.ID).WithGithubID("a2").Build(t, ctx, ts.Store, userID)
		fixtures.NewNotification(repoB.ID).WithGithubID("b1").Build(t, ctx, ts.Store, userID)
		fixtures.NewNotification(repoC.ID).WithGithubID("c1").Build(t, ctx, ts.Store, userID)

		// Unscoped list sees everything.
		all := c.ListNotifications(t, "in:inbox", 1, 100)
		require.Equal(t, int64(4), all.Total)

		// Scoped to two repositories.
		scoped := c.ListNotificationsInRepositories(t, "in:inbox", []int64{repoA.ID, repoB.ID}, 1, 100)
		require.Equal(t, int64(3), scoped.Total)
		for _, n := range scoped.Notifications {
			require.NotEqual(t, repoC.ID, n.RepositoryID)
		}

		// Bulk archive scoped to alpha only touches alpha.
		archived := c.BulkArchiveInRepositories(t, "in:inbox", []int64{repoA.ID})
		require.Equal(t, 2, archived.Count)

		remaining := c.ListNotifications(t, "in:inbox", 1, 100)
		require.Equal(t, int64(2), remaining.Total)
		for _, n := range remaining.Notifications {
			require.NotEqual(t, repoA.ID, n.RepositoryID)
		}

		// Counts reflect the archive.
		counts := c.ListRepositoryCounts(t, "in:inbox")
		require.Len(t, counts.Repositories, 2)
	})
}
