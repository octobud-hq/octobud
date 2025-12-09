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

// Package integration contains API-level integration tests that run against
// the SQLite backend.
package integration

import (
	"os"
	"sync"
	"testing"

	"github.com/octobud-hq/octobud/backend/integration/client"
	"github.com/octobud-hq/octobud/backend/integration/testserver"
)

var (
	// Lazy-initialized server
	sqliteServer *testserver.TestServer

	// Mutex for thread-safe lazy initialization
	sqliteMu sync.Mutex
)

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup
	if sqliteServer != nil {
		sqliteServer.Cleanup()
	}

	os.Exit(code)
}

// getSQLiteServer returns the SQLite test server, creating it if necessary.
func getSQLiteServer(t *testing.T) *testserver.TestServer {
	sqliteMu.Lock()
	defer sqliteMu.Unlock()

	if sqliteServer == nil {
		sqliteServer = testserver.NewSQLite(t)
	}
	return sqliteServer
}

// RunWithBackends runs a test function against the SQLite backend.
func RunWithBackends(t *testing.T, testFn func(t *testing.T, ts *testserver.TestServer, c *client.Client)) {
	t.Helper()

	t.Run("sqlite", func(t *testing.T) {
		ts := getSQLiteServer(t)

		// Reset database state
		ts.Reset(t)

		// Create a new client (no authentication needed - API trusts localhost)
		c := client.New(ts.Server.URL)

		// Run the test with userID available via ts.UserID
		testFn(t, ts, c)
	})
}
