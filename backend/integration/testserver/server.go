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

// Package testserver provides test server setup for integration tests.
package testserver

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/octobud-hq/octobud/backend/internal/api"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/sqlite"
	"github.com/octobud-hq/octobud/backend/internal/server"

	// SQLite driver
	_ "modernc.org/sqlite"
)

// TestServer wraps a test HTTP server with database access.
type TestServer struct {
	Server  *httptest.Server
	Store   db.Store
	DB      *sql.DB
	UserID  string // GitHub user ID for test user
	Cleanup func()
}

// NewSQLite creates a test server backed by in-memory SQLite.
func NewSQLite(t *testing.T) *TestServer {
	t.Helper()

	ctx := context.Background()

	// Open in-memory SQLite database
	dbConn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}

	// Enable foreign keys and WAL mode
	if _, err := dbConn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		dbConn.Close()
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run migrations
	if err := goose.SetDialect("sqlite3"); err != nil {
		dbConn.Close()
		t.Fatalf("Failed to set goose dialect: %v", err)
	}
	goose.SetBaseFS(sqlite.MigrationsFS)
	if err := goose.Up(dbConn, "migrations"); err != nil {
		dbConn.Close()
		t.Fatalf("Failed to run SQLite migrations: %v", err)
	}

	// Create store
	store := db.NewStore(dbConn)

	// Initialize user record with GitHub identity
	authService := authsvc.NewService(store)
	if err := authService.EnsureUser(ctx); err != nil {
		dbConn.Close()
		t.Fatalf("Failed to ensure user: %v", err)
	}

	// Set up GitHub identity for the test user
	// Use a test GitHub user ID and username
	testUserID := "test-user-12345"
	testUsername := "testuser"
	if err := authService.UpdateGitHubIdentity(ctx, testUserID, testUsername); err != nil {
		dbConn.Close()
		t.Fatalf("Failed to set GitHub identity: %v", err)
	}

	// Create API handler (trusts localhost - no JWT auth needed)
	apiHandler := api.NewHandler(store)

	// Set up router
	router := server.NewRouter(server.DefaultConfig())
	router.Route("/api", apiHandler.RegisterAllRoutes)

	// Create test server
	ts := httptest.NewServer(router)

	return &TestServer{
		Server: ts,
		Store:  store,
		DB:     dbConn,
		UserID: testUserID,
		Cleanup: func() {
			ts.Close()
			dbConn.Close()
		},
	}
}

// Reset clears all data from the database while preserving the schema.
// This is useful for resetting state between tests.
func (ts *TestServer) Reset(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	// Delete all data in reverse order of dependencies
	tables := []string{
		"tag_assignments",
		"notifications",
		"pull_requests",
		"repositories",
		"tags",
		"views",
		"rules",
		"sync_state",
		// Don't delete users - we need the user record
	}

	for _, table := range tables {
		if _, err := ts.DB.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("Failed to clear table %s: %v", table, err)
		}
	}
}
