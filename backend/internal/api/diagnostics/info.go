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
	"net/http"
	"runtime"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
)

// handleInfo returns process/build metadata for the diagnostics panel.
func (h *Handler) handleInfo(w http.ResponseWriter, _ *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, h.buildInfo())
}

// buildInfo composes the InfoResponse so it can be reused by /bundle.
func (h *Handler) buildInfo() InfoResponse {
	return InfoResponse{
		Version:       h.version,
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		DataDir:       h.dataDir,
		LogDir:        h.logDir,
		LogFile:       h.logFile,
		StartedAt:     h.startedAt.UTC().Format(time.RFC3339),
		UptimeSeconds: int64(time.Since(h.startedAt).Seconds()),
	}
}
