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
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite" // Embedded.
)

// OpenDatabase opens a SQLite database connection.
// Returns the database connection and any error.
func OpenDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// Enable foreign keys and WAL mode for SQLite
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	// Set busy timeout - SQLite will wait and retry on lock contention
	// instead of immediately returning SQLITE_BUSY
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout = 2000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}
	return db, nil
}

// NewStore creates a new Store implementation for SQLite.
// All operations automatically retry on SQLITE_BUSY errors (handled within the store implementation).
func NewStore(dbConn *sql.DB) Store {
	if sqliteStoreFactory != nil {
		return sqliteStoreFactory(dbConn)
	}
	panic("sqlite store factory not registered")
}

// sqliteStoreFactory is set by the sqlite package to create stores without import cycles
var sqliteStoreFactory func(*sql.DB) Store

// RegisterSQLiteStore registers the SQLite store factory
func RegisterSQLiteStore(factory func(*sql.DB) Store) {
	sqliteStoreFactory = factory
}

// GetDefaultDataDir returns the default data directory for the current platform
func GetDefaultDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "Octobud"), nil
	case "linux":
		// Follow XDG spec
		if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
			return filepath.Join(xdgData, "octobud"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "octobud"), nil
	case "windows":
		if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
			return filepath.Join(appData, "Octobud"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Local", "Octobud"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".octobud"), nil
	}
}
