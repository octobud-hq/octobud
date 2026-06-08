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
	"time"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
)

// handleBundle returns a small, paste-safe diagnostic payload: app/runtime
// info plus token source and expiry. Anything that could leak repo/org names,
// usernames, or API URLs (logs, GitHub status responses, rate-limit error
// strings) is intentionally absent — for that, users can use the in-app log
// viewer and copy whatever they're comfortable disclosing themselves.
func (h *Handler) handleBundle(w http.ResponseWriter, _ *http.Request) {
	info := h.buildInfo()
	info.DataDir = sanitizePath(info.DataDir)
	info.LogDir = sanitizePath(info.LogDir)
	info.LogFile = sanitizePath(info.LogFile)

	resp := BundleResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Info:        info,
	}
	if h.tokenManager != nil {
		resp.TokenSource = h.tokenManager.GetStatusSource()
		if expiresAt := h.tokenManager.GetTokenExpiresAt(); expiresAt != nil {
			formatted := expiresAt.UTC().Format(time.RFC3339)
			resp.TokenExpiresAt = &formatted
		}
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}
