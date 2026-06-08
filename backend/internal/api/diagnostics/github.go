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
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
)

// handleGitHubStatus returns the connected account, token health, and a
// freshly-fetched rate-limit snapshot. The rate-limit fetch failing degrades
// to RateLimitError rather than failing the whole response.
func (h *Handler) handleGitHubStatus(w http.ResponseWriter, r *http.Request) {
	resp := h.buildGitHubStatus(r.Context())
	helpers.WriteJSON(w, http.StatusOK, resp)
}

// buildGitHubStatus composes the response so /bundle can reuse it.
func (h *Handler) buildGitHubStatus(ctx context.Context) GitHubStatusResponse {
	if h.tokenManager == nil {
		return GitHubStatusResponse{Connected: false, Source: "none"}
	}

	resp := GitHubStatusResponse{
		Connected:      h.tokenManager.GetStatusConnected(),
		Source:         h.tokenManager.GetStatusSource(),
		MaskedToken:    h.tokenManager.GetStatusMaskedToken(),
		GitHubUsername: h.tokenManager.GetStatusGitHubUsername(),
		GitHubUserID:   h.tokenManager.GetStatusGitHubUserID(),
	}
	if expiresAt := h.tokenManager.GetTokenExpiresAt(); expiresAt != nil {
		formatted := expiresAt.UTC().Format(time.RFC3339)
		resp.TokenExpiresAt = &formatted
	}

	if !resp.Connected || h.githubClient == nil {
		return resp
	}

	// Bounded budget for the passthrough so a slow GitHub call doesn't block
	// the diagnostics page UI loading.
	rlCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	rl, err := h.githubClient.FetchRateLimit(rlCtx)
	if err != nil {
		h.logger.Debug("rate limit fetch failed", zap.Error(err))
		resp.RateLimitError = err.Error()
		return resp
	}
	resp.RateLimit = rl
	return resp
}
