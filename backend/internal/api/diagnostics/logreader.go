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
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// logQuery captures the filter parameters for a log read.
type logQuery struct {
	// Level filters entries to those at this severity or higher. Empty means no
	// filter. Compared against the lowercase "level" field.
	level string
	// Since filters out entries with a timestamp older than this. Zero value
	// means no filter.
	since time.Time
	// Search filters entries whose message or any string field doesn't contain
	// this substring (case-insensitive). Empty means no filter.
	search string
	// Limit caps the number of entries returned (newest first). 0 falls back
	// to a server default.
	limit int
}

// levelRank maps zap level names (lowercase) to an integer for "or higher"
// comparisons. Unknown levels rank below debug so they pass through unfiltered
// when no level filter is set, but fail strict comparisons.
var levelRank = map[string]int{
	"debug":  0,
	"info":   1,
	"warn":   2,
	"error":  3,
	"dpanic": 4,
	"panic":  5,
	"fatal":  6,
}

// readLogEntries returns up to query.limit log entries matching the query,
// drawn from the current log file and rotated/gzipped backups, newest first.
// The match predicate is applied inside the per-file scan so we don't blindly
// keep only the trailing N raw lines — otherwise a search/level filter would
// miss matches that happen to sit further back in the file.
// truncated indicates whether at least one more matching entry exists past
// the returned slice.
func readLogEntries(
	logFile string,
	query logQuery,
) (entries []LogEntry, truncated bool, err error) {
	if query.limit <= 0 {
		query.limit = 200
	}

	files, err := collectLogFiles(logFile)
	if err != nil {
		return nil, false, err
	}

	match := func(e LogEntry) bool { return matchesQuery(e, query) }
	out := make([]LogEntry, 0, query.limit)
	for _, f := range files {
		// Ask for one extra so a returned slice of size (need+1) tells us
		// there's at least one more match past what we have room for.
		need := query.limit - len(out) + 1
		fileEntries, fileErr := readJSONLinesReverse(f, need, match)
		if fileErr != nil {
			// Treat an unreadable rotated file as best-effort — surface the
			// problem in logs but continue with what we already have.
			continue
		}
		for _, e := range fileEntries {
			if len(out) >= query.limit {
				return out, true, nil
			}
			out = append(out, e)
		}
	}
	return out, false, nil
}

// collectLogFiles returns the current log file followed by rotated backups in
// reverse chronological order (newest backup first). Missing files are skipped.
func collectLogFiles(logFile string) ([]string, error) {
	dir := filepath.Dir(logFile)
	base := filepath.Base(logFile)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	out := []string{}
	if _, err := os.Stat(logFile); err == nil {
		out = append(out, logFile)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Current file present but dir read failed — return what we have.
		if len(out) > 0 {
			return out, nil
		}
		return nil, err
	}

	type rotated struct {
		path string
		mod  time.Time
	}
	var rotations []rotated
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == base {
			continue
		}
		// lumberjack rotated names are of the form "<stem>-<timestamp><ext>" or
		// the same with ".gz" appended for compressed backups.
		if !strings.HasPrefix(name, stem+"-") {
			continue
		}
		if !strings.HasSuffix(name, ext) && !strings.HasSuffix(name, ext+".gz") {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		rotations = append(rotations, rotated{path: filepath.Join(dir, name), mod: info.ModTime()})
	}

	sort.Slice(rotations, func(i, j int) bool {
		return rotations[i].mod.After(rotations[j].mod)
	})
	for _, r := range rotations {
		out = append(out, r.path)
	}
	return out, nil
}

// openLogFile opens a regular file by path, refusing to follow symlinks.
// Defense-in-depth: callers obtain path from collectLogFiles which enumerates
// a controlled directory, but rejecting symlinks closes the only realistic
// substitution attack (a planted symlink pointing outside the log dir).
func openLogFile(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to open symlink: %s", path)
	}
	//nolint:gosec // G304: path comes from collectLogFiles enumeration; symlinks rejected via Lstat above
	return os.Open(path)
}

// readJSONLinesReverse reads up to wantedEntries JSON log lines from the tail
// of a file (gzip-aware) that satisfy the match predicate, returning them
// newest-first. We tolerate non-JSON lines (mixed-format historical content)
// by skipping them rather than erroring. A nil match predicate keeps every
// decoded entry.
func readJSONLinesReverse(
	path string,
	wantedEntries int,
	match func(LogEntry) bool,
) ([]LogEntry, error) {
	f, err := openLogFile(path)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck // read-only file close: failure has no actionable handling
	defer f.Close()

	var reader io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, gzErr := gzip.NewReader(f)
		if gzErr != nil {
			return nil, gzErr
		}
		//nolint:errcheck // read-only gzip close: failure has no actionable handling
		defer gz.Close()
		reader = gz
	}

	// Streaming the whole file forward and keeping a ring of the last N matches
	// is fine for our sizing (10MB per file). Avoids the complexity of reverse
	// byte scans, and gzip-wrapped files can't be seeked anyway.
	scanner := bufio.NewScanner(reader)
	// Allow long stacktrace lines (zap stacktraces can be sizeable).
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	ring := make([]LogEntry, 0, wantedEntries)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		entry, ok := decodeLogLine(line)
		if !ok {
			continue
		}
		if match != nil && !match(entry) {
			continue
		}
		if len(ring) < wantedEntries {
			ring = append(ring, entry)
			continue
		}
		copy(ring, ring[1:])
		ring[len(ring)-1] = entry
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	// Reverse to newest-first.
	for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
		ring[i], ring[j] = ring[j], ring[i]
	}
	return ring, nil
}

// decodeLogLine parses one JSON log line into a LogEntry, putting any
// non-standard keys into Fields. Returns false if the line isn't a JSON object.
func decodeLogLine(line []byte) (LogEntry, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return LogEntry{}, false
	}
	entry := LogEntry{}
	extractField(raw, "time", &entry.Time)
	extractField(raw, "level", &entry.Level)
	extractField(raw, "logger", &entry.Logger)
	extractField(raw, "caller", &entry.Caller)
	extractField(raw, "msg", &entry.Message)
	extractField(raw, "stacktrace", &entry.Stacktrace)
	if len(raw) > 0 {
		entry.Fields = make(map[string]interface{}, len(raw))
		for k, v := range raw {
			var decoded interface{}
			if err := json.Unmarshal(v, &decoded); err == nil {
				entry.Fields[k] = decoded
			} else {
				entry.Fields[k] = string(v)
			}
		}
	}
	return entry, true
}

// extractField unmarshals raw[key] into dst and removes the key. Tolerates
// malformed values by leaving dst untouched — log lines from older app
// versions may carry fields of unexpected types and shouldn't fail the parse.
func extractField(raw map[string]json.RawMessage, key string, dst interface{}) {
	v, ok := raw[key]
	if !ok {
		return
	}
	delete(raw, key)
	if err := json.Unmarshal(v, dst); err != nil {
		return
	}
}

// matchesQuery returns true if the entry passes the level/since/search filters.
func matchesQuery(entry LogEntry, q logQuery) bool {
	if q.level != "" {
		entryRank, ok := levelRank[strings.ToLower(entry.Level)]
		if !ok {
			return false
		}
		minRank, ok := levelRank[strings.ToLower(q.level)]
		if !ok {
			return false
		}
		if entryRank < minRank {
			return false
		}
	}
	if !q.since.IsZero() && entry.Time != "" {
		// Try a couple of formats — JSON encoder is ISO8601, but tolerate
		// RFC3339Nano in case a future config emits sub-second precision.
		t, err := time.Parse(time.RFC3339, entry.Time)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05.000-0700", entry.Time)
		}
		if err == nil && t.Before(q.since) {
			return false
		}
	}
	if q.search != "" {
		needle := strings.ToLower(q.search)
		if strings.Contains(strings.ToLower(entry.Message), needle) {
			return true
		}
		if strings.Contains(strings.ToLower(entry.Stacktrace), needle) {
			return true
		}
		for _, v := range entry.Fields {
			if s, ok := v.(string); ok && strings.Contains(strings.ToLower(s), needle) {
				return true
			}
		}
		return false
	}
	return true
}
