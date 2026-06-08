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

// Package diagnostics provides the in-app debugging panel: log viewer, log
// download, GitHub rate-limit status, and a copy-to-clipboard diagnostics
// bundle. All endpoints are read-only or trigger local OS actions (reveal in
// file manager); none mutate persistent state.
package diagnostics

import (
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
)

// TokenStatusProvider is the slice of TokenManager state the diagnostics page
// surfaces. Kept narrow so tests can fake it without dragging in the full
// TokenManager.
type TokenStatusProvider interface {
	GetStatusConnected() bool
	GetStatusSource() string
	GetStatusMaskedToken() string
	GetStatusGitHubUsername() string
	GetStatusGitHubUserID() string
	GetTokenExpiresAt() *time.Time
}

// Handler serves the /api/diagnostics/* routes.
type Handler struct {
	logger       *zap.Logger
	version      string
	dataDir      string
	logDir       string
	logFile      string
	startedAt    time.Time
	tokenManager TokenStatusProvider
	githubClient githubinterfaces.Client
}

// Config is the bundle of dependencies required to build a Handler.
type Config struct {
	Logger       *zap.Logger
	Version      string
	DataDir      string
	LogDir       string
	LogFileName  string // typically "octobud.log"
	TokenManager TokenStatusProvider
	GitHubClient githubinterfaces.Client
}

// New creates a diagnostics handler. TokenManager and GitHubClient may be nil
// (e.g. during early boot or when GitHub isn't configured) — endpoints that
// require them return a 503 in that case.
func New(cfg Config) *Handler {
	logFileName := cfg.LogFileName
	if logFileName == "" {
		logFileName = "octobud.log"
	}
	return &Handler{
		logger:       cfg.Logger,
		version:      cfg.Version,
		dataDir:      cfg.DataDir,
		logDir:       cfg.LogDir,
		logFile:      filepath.Join(cfg.LogDir, logFileName),
		startedAt:    time.Now(),
		tokenManager: cfg.TokenManager,
		githubClient: cfg.GitHubClient,
	}
}

// Register attaches diagnostics routes to the provided router. The router
// should already be scoped to /api.
func (h *Handler) Register(r chi.Router) {
	r.Route("/diagnostics", func(r chi.Router) {
		r.Get("/info", h.handleInfo)
		r.Get("/logs", h.handleLogs)
		r.Get("/logs/download", h.handleLogsDownload)
		r.Get("/github/status", h.handleGitHubStatus)
		r.Get("/bundle", h.handleBundle)
	})
}
