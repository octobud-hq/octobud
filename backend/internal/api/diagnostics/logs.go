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
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
)

// maxLogLimit caps how many entries one /logs call can return. Generous enough
// for "show me everything from the last hour" but bounded so a misbehaving
// caller can't ask for the whole file.
const maxLogLimit = 2000

// defaultLogLimit is the limit applied when the caller doesn't specify one.
const defaultLogLimit = 200

// handleLogs returns recent log entries, newest first, filtered by query params.
// Query params: level (min level), since (RFC3339), search (substring), limit (int).
func (h *Handler) handleLogs(w http.ResponseWriter, r *http.Request) {
	q := logQuery{
		level:  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("level"))),
		search: strings.TrimSpace(r.URL.Query().Get("search")),
		limit:  defaultLogLimit,
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			q.since = t
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			q.limit = n
		}
	}
	if q.limit > maxLogLimit {
		q.limit = maxLogLimit
	}

	entries, truncated, err := readLogEntries(h.logFile, q)
	if err != nil {
		// File not existing yet (fresh install) is a normal condition — return
		// an empty list rather than an error so the UI can show "no logs yet".
		if os.IsNotExist(err) {
			helpers.WriteJSON(w, http.StatusOK, LogsResponse{Entries: []LogEntry{}, Returned: 0})
			return
		}
		h.logger.Error("failed to read log entries", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "failed to read logs")
		return
	}

	sanitizeEntries(entries)
	helpers.WriteJSON(w, http.StatusOK, LogsResponse{
		Entries:   entries,
		Returned:  len(entries),
		Truncated: truncated,
	})
}

// handleLogsDownload streams a zip of every file in the logs directory. We
// include rotated/gzipped backups as-is so the bundle reflects what's actually
// on disk.
func (h *Handler) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	files, err := collectLogFiles(h.logFile)
	if err != nil {
		h.logger.Error("failed to enumerate log files", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "failed to read log directory")
		return
	}
	if len(files) == 0 {
		helpers.WriteError(w, http.StatusNotFound, "no log files available")
		return
	}

	filename := fmt.Sprintf("octobud-logs-%s.zip", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	zw := zip.NewWriter(w)
	defer func() {
		// A Close failure here means the central directory wasn't flushed,
		// producing a truncated archive. Headers are already sent so we can't
		// surface this to the client — log it so it's visible on the server.
		if closeErr := zw.Close(); closeErr != nil {
			h.logger.Warn("failed to close log archive writer", zap.Error(closeErr))
		}
	}()

	for _, path := range files {
		if err := addFileToZip(zw, path); err != nil {
			h.logger.Warn("failed to add file to log archive",
				zap.String("path", path), zap.Error(err))
			// Continue with remaining files — a partial archive is better than none.
		}
		if r.Context().Err() != nil {
			return
		}
	}
}

func addFileToZip(zw *zip.Writer, path string) error {
	f, err := openLogFile(path)
	if err != nil {
		return err
	}
	//nolint:errcheck // read-only file close: failure has no actionable handling
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(path)
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, f); err != nil {
		return err
	}
	return nil
}
