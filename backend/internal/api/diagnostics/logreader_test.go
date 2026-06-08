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

package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeLog writes the given lines to <dir>/octobud.log and returns the path.
func writeLog(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "octobud.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	defer func() { _ = f.Close() }()
	for _, line := range lines {
		if _, err := fmt.Fprintln(f, line); err != nil {
			t.Fatalf("write line: %v", err)
		}
	}
	return path
}

// TestReadLogEntries_LevelFilterFindsMatchesBeyondTrailingWindow is the
// regression test for the bug where filtering happened after trimming to the
// trailing N raw lines: if no errors sat in the last 200 lines, the viewer
// reported "no errors" even when the file clearly contained some.
func TestReadLogEntries_LevelFilterFindsMatchesBeyondTrailingWindow(t *testing.T) {
	var lines []string
	// One error early in the file...
	lines = append(lines, `{"level":"error","time":"2026-01-01T00:00:00Z","msg":"the needle"}`)
	// ...followed by 500 routine info entries that would have completely
	// crowded out the error under the old trailing-window behavior.
	for i := 0; i < 500; i++ {
		lines = append(lines,
			fmt.Sprintf(`{"level":"info","time":"2026-01-01T00:00:00Z","msg":"routine %d"}`, i))
	}
	path := writeLog(t, lines)

	entries, truncated, err := readLogEntries(path, logQuery{level: "error", limit: 200})
	if err != nil {
		t.Fatalf("readLogEntries: %v", err)
	}
	if truncated {
		t.Errorf("did not expect truncated=true with a single match")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 matching entry, got %d", len(entries))
	}
	if entries[0].Message != "the needle" {
		t.Errorf("got wrong entry: %+v", entries[0])
	}
}

func TestReadLogEntries_SearchFindsMatchesBeyondTrailingWindow(t *testing.T) {
	lines := []string{
		`{"level":"info","time":"2026-01-01T00:00:00Z","msg":"contains needle here"}`,
	}
	for i := 0; i < 500; i++ {
		lines = append(lines,
			fmt.Sprintf(`{"level":"info","time":"2026-01-01T00:00:00Z","msg":"haystack %d"}`, i))
	}
	path := writeLog(t, lines)

	entries, _, err := readLogEntries(path, logQuery{search: "needle", limit: 200})
	if err != nil {
		t.Fatalf("readLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 matching entry, got %d", len(entries))
	}
}

func TestReadLogEntries_NoFilterReturnsNewestFirst(t *testing.T) {
	lines := []string{
		`{"level":"info","time":"2026-01-01T00:00:00Z","msg":"oldest"}`,
		`{"level":"info","time":"2026-01-01T00:00:01Z","msg":"middle"}`,
		`{"level":"info","time":"2026-01-01T00:00:02Z","msg":"newest"}`,
	}
	path := writeLog(t, lines)

	entries, _, err := readLogEntries(path, logQuery{limit: 10})
	if err != nil {
		t.Fatalf("readLogEntries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Message != "newest" || entries[2].Message != "oldest" {
		t.Errorf("entries not in newest-first order: %+v", entries)
	}
}

// TestReadLogEntries_TruncatedFlagOnExcessMatches verifies the limit boundary
// and that the truncated flag correctly signals there's more behind it.
func TestReadLogEntries_TruncatedFlagOnExcessMatches(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines,
			fmt.Sprintf(`{"level":"error","time":"2026-01-01T00:00:00Z","msg":"err %d"}`, i))
	}
	path := writeLog(t, lines)

	entries, truncated, err := readLogEntries(path, logQuery{level: "error", limit: 10})
	if err != nil {
		t.Fatalf("readLogEntries: %v", err)
	}
	if len(entries) != 10 {
		t.Fatalf("expected exactly limit=10 entries, got %d", len(entries))
	}
	if !truncated {
		t.Error("expected truncated=true when matches exceed limit")
	}
	// Newest matches should be returned, not oldest.
	if entries[0].Message != "err 49" {
		t.Errorf("expected newest match at index 0, got %q", entries[0].Message)
	}
}
