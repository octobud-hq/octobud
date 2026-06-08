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
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

// InfoResponse is the payload returned by GET /api/diagnostics/info.
type InfoResponse struct {
	Version       string `json:"version"`
	GoVersion     string `json:"goVersion"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	DataDir       string `json:"dataDir"`
	LogDir        string `json:"logDir"`
	LogFile       string `json:"logFile"`
	StartedAt     string `json:"startedAt"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
}

// LogEntry is one parsed log line from the JSON log file. Fields beyond the
// fixed set are preserved in Fields so the UI can show structured context
// (zap.Field values) without us enumerating every key.
type LogEntry struct {
	Time       string                 `json:"time"`
	Level      string                 `json:"level"`
	Logger     string                 `json:"logger,omitempty"`
	Caller     string                 `json:"caller,omitempty"`
	Message    string                 `json:"msg"`
	Stacktrace string                 `json:"stacktrace,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

// LogsResponse is the payload returned by GET /api/diagnostics/logs.
type LogsResponse struct {
	Entries  []LogEntry `json:"entries"`
	Returned int        `json:"returned"`
	// Truncated is true when the requested window was capped by the server limit.
	Truncated bool `json:"truncated"`
}

// GitHubStatusResponse is the payload returned by GET /api/diagnostics/github/status.
type GitHubStatusResponse struct {
	Connected      bool             `json:"connected"`
	GitHubUsername string           `json:"githubUsername,omitempty"`
	GitHubUserID   string           `json:"githubUserId,omitempty"`
	MaskedToken    string           `json:"maskedToken,omitempty"`
	Source         string           `json:"source"`
	TokenExpiresAt *string          `json:"tokenExpiresAt,omitempty"`
	RateLimit      *types.RateLimit `json:"rateLimit,omitempty"`
	// RateLimitError carries a fetch-side error message when the rate-limit call
	// itself failed (e.g. network, expired token). The rest of the payload is
	// still populated so the UI can show what it has.
	RateLimitError string `json:"rateLimitError,omitempty"`
}

// BundleResponse is the payload returned by GET /api/diagnostics/bundle —
// deliberately limited to information that is safe to share verbatim in a
// public bug report. Anything that could carry repo/org names, usernames,
// API URLs, or PR/issue titles (i.e. logs and GitHub API responses) lives in
// the in-app viewer instead, where the user can choose what to disclose.
type BundleResponse struct {
	GeneratedAt    string       `json:"generatedAt"`
	Info           InfoResponse `json:"info"`
	TokenSource    string       `json:"tokenSource,omitempty"`
	TokenExpiresAt *string      `json:"tokenExpiresAt,omitempty"`
}
