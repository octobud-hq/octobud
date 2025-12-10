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

// Package user provides the HTTP handler for user-related routes.
package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/helpers"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/syncstate"
	"github.com/octobud-hq/octobud/backend/internal/core/update"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/osactions"
)

// Handler handles user-related HTTP routes
type Handler struct {
	logger        *zap.Logger
	authSvc       authsvc.AuthService
	store         db.Store                   // For database operations
	scheduler     jobs.Scheduler             // For queueing sync jobs
	syncStateSvc  syncstate.SyncStateService // For sync state operations
	tokenManager  TokenManagerInterface      // For GitHub token management
	updateService *update.Service            // For update checking
	osActionsSvc  osactions.Service          // For OS-specific actions (restart, browser tabs, etc.)
}

// New creates a new user handler
func New(
	logger *zap.Logger,
	authSvc authsvc.AuthService,
) *Handler {
	return &Handler{
		logger:  logger,
		authSvc: authSvc,
	}
}

// WithScheduler sets the scheduler for queueing jobs.
func (h *Handler) WithScheduler(scheduler jobs.Scheduler) *Handler {
	h.scheduler = scheduler
	return h
}

// WithSyncStateService sets the sync state service for sync operations
func (h *Handler) WithSyncStateService(svc syncstate.SyncStateService) *Handler {
	h.syncStateSvc = svc
	return h
}

// TokenManagerInterface defines the interface for GitHub token management
type TokenManagerInterface interface {
	GetStatusConnected() bool
	GetStatusSource() string
	GetStatusMaskedToken() string
	GetStatusGitHubUsername() string
	GetStatusGitHubUserID() string
	IsConnected() bool
	SetToken(ctx context.Context, token string) (string, error)
	SetTokenFromOAuth(ctx context.Context, token string) (string, error)
	ClearToken(ctx context.Context) error
}

// WithTokenManager sets the token manager for GitHub token operations
func (h *Handler) WithTokenManager(tm TokenManagerInterface) *Handler {
	h.tokenManager = tm
	return h
}

// WithStore sets the database store for database operations
func (h *Handler) WithStore(store db.Store) *Handler {
	h.store = store
	return h
}

// WithUpdateService sets the update service for update checking
func (h *Handler) WithUpdateService(service *update.Service) *Handler {
	h.updateService = service
	return h
}

// WithOSActionsService sets the OS actions service for platform-specific operations
func (h *Handler) WithOSActionsService(service osactions.Service) *Handler {
	h.osActionsSvc = service
	return h
}

// Register registers user routes on the provided router.
func (h *Handler) Register(r chi.Router) {
	r.Route("/user", func(r chi.Router) {
		// User info (based on GitHub identity)
		r.Get("/me", h.HandleGetCurrentUser)

		// Sync settings
		r.Get("/sync-settings", h.HandleGetSyncSettings)
		r.Put("/sync-settings", h.HandleUpdateSyncSettings)
		r.Get("/sync-state", h.HandleGetSyncState)
		r.Post("/sync-older", h.HandleSyncOlder)

		// GitHub token management
		r.Get("/github-status", h.HandleGetGitHubStatus)
		r.Put("/github-token", h.HandleSetGitHubToken)
		r.Delete("/github-token", h.HandleClearGitHubToken)

		// Storage management
		r.Get("/storage-stats", h.HandleGetStorageStats)
		r.Get("/retention-settings", h.HandleGetRetentionSettings)
		r.Put("/retention-settings", h.HandleUpdateRetentionSettings)
		r.Post("/cleanup", h.HandleRunCleanup)
		r.Get("/cleanup-preview", h.HandleGetEligibleForCleanup)
		r.Delete("/github-data", h.HandleDeleteAllGitHubData)

		// Mute management
		r.Get("/mute-status", h.HandleGetMuteStatus)
		r.Put("/mute", h.HandleSetMute)
		r.Delete("/mute", h.HandleClearMute)

		// Update management
		r.Get("/update-settings", h.HandleGetUpdateSettings)
		r.Put("/update-settings", h.HandleUpdateUpdateSettings)
		r.Get("/update-check", h.HandleCheckForUpdates)
		r.Get("/version", h.HandleGetVersion)
		r.Get("/installed-version", h.HandleGetInstalledVersion)
		r.Post("/restart", h.HandleRestart)
	})
}

// HandleGetCurrentUser handles GET /api/user/me
// Returns user info based on GitHub identity
func (h *Handler) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := h.authSvc.GetUser(ctx)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := UserResponse{
		GithubUserID:   user.GithubUserID,
		GithubUsername: user.GithubUsername,
	}

	// Include sync settings if they exist
	if user.SyncSettings != nil {
		response.SyncSettings = &SyncSettingsResponse{
			InitialSyncDays:       user.SyncSettings.InitialSyncDays,
			InitialSyncMaxCount:   user.SyncSettings.InitialSyncMaxCount,
			InitialSyncUnreadOnly: user.SyncSettings.InitialSyncUnreadOnly,
			SetupCompleted:        user.SyncSettings.SetupCompleted,
		}
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// HandleGetGitHubStatus handles GET /api/user/github-status
func (h *Handler) HandleGetGitHubStatus(w http.ResponseWriter, _ *http.Request) {
	if h.tokenManager == nil {
		helpers.WriteError(
			w,
			http.StatusServiceUnavailable,
			"GitHub token management not configured",
		)
		return
	}

	response := GitHubStatusResponse{
		Connected:      h.tokenManager.GetStatusConnected(),
		Source:         h.tokenManager.GetStatusSource(),
		MaskedToken:    h.tokenManager.GetStatusMaskedToken(),
		GitHubUsername: h.tokenManager.GetStatusGitHubUsername(),
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// HandleSetGitHubToken handles PUT /api/user/github-token
func (h *Handler) HandleSetGitHubToken(w http.ResponseWriter, r *http.Request) {
	if h.tokenManager == nil {
		helpers.WriteError(
			w,
			http.StatusServiceUnavailable,
			"GitHub token management not configured",
		)
		return
	}

	var req SetGitHubTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode set token request", zap.Error(err))
		helpers.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Token == "" {
		helpers.WriteError(w, http.StatusBadRequest, "Token is required")
		return
	}

	ctx := r.Context()
	githubUsername, err := h.tokenManager.SetToken(ctx, req.Token)
	if err != nil {
		h.logger.Debug("failed to set GitHub token", zap.Error(err))
		// Provide user-friendly error messages
		errMsg := err.Error()
		if strings.Contains(errMsg, "unauthorized") {
			helpers.WriteError(w, http.StatusBadRequest, "Invalid token: authentication failed")
			return
		}
		if strings.Contains(errMsg, "rate limit") {
			helpers.WriteError(
				w,
				http.StatusTooManyRequests,
				"GitHub API rate limit exceeded. Please try again later.",
			)
			return
		}
		if strings.Contains(errMsg, "permissions") {
			helpers.WriteError(
				w,
				http.StatusBadRequest,
				"Token does not have required permissions (needs 'repo', 'notifications', and 'read:discussions' scopes)",
			)
			return
		}
		helpers.WriteError(w, http.StatusBadRequest, "Failed to validate token: "+errMsg)
		return
	}

	h.logger.Info("GitHub token configured",
		zap.String("github_username", githubUsername),
		zap.String("ip", getClientIP(r)),
	)

	response := SetGitHubTokenResponse{
		GitHubUsername: githubUsername,
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// HandleClearGitHubToken handles DELETE /api/user/github-token
func (h *Handler) HandleClearGitHubToken(w http.ResponseWriter, r *http.Request) {
	if h.tokenManager == nil {
		helpers.WriteError(
			w,
			http.StatusServiceUnavailable,
			"GitHub token management not configured",
		)
		return
	}

	ctx := r.Context()
	if err := h.tokenManager.ClearToken(ctx); err != nil {
		h.logger.Error("failed to clear GitHub token", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to clear token")
		return
	}

	h.logger.Info("GitHub token cleared",
		zap.String("ip", getClientIP(r)),
	)

	// Return current status after clearing
	response := GitHubStatusResponse{
		Connected:      h.tokenManager.GetStatusConnected(),
		Source:         h.tokenManager.GetStatusSource(),
		MaskedToken:    h.tokenManager.GetStatusMaskedToken(),
		GitHubUsername: h.tokenManager.GetStatusGitHubUsername(),
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// HandleDeleteAllGitHubData handles DELETE /api/user/github-data
func (h *Handler) HandleDeleteAllGitHubData(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database store not configured")
		return
	}

	ctx := r.Context()
	userID, ok := helpers.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	err := h.store.DeleteAllGitHubData(ctx, userID)
	if err != nil {
		h.logger.Error("failed to delete all GitHub data", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to delete GitHub data")
		return
	}

	h.logger.Info("Deleted all GitHub data",
		zap.String("user_id", userID),
		zap.String("ip", getClientIP(r)),
	)

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// getClientIP extracts the client IP address from the request
// Checks X-Forwarded-For, X-Real-IP, and RemoteAddr
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by reverse proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (set by some reverse proxies)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// HandleGetMuteStatus handles GET /api/user/mute-status
func (h *Handler) HandleGetMuteStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := h.authSvc.GetUser(ctx)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := MuteStatusResponse{
		IsMuted: user.MutedUntil.Valid && user.MutedUntil.Time.After(time.Now()),
	}

	if user.MutedUntil.Valid && user.MutedUntil.Time.After(time.Now()) {
		mutedUntilStr := user.MutedUntil.Time.Format(time.RFC3339)
		response.MutedUntil = &mutedUntilStr
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// HandleSetMute handles PUT /api/user/mute
func (h *Handler) HandleSetMute(w http.ResponseWriter, r *http.Request) {
	var req SetMuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode set mute request", zap.Error(err))
		helpers.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	now := time.Now()
	var mutedUntil time.Time

	switch req.Duration {
	case "30m":
		mutedUntil = now.Add(30 * time.Minute)
	case "1h":
		mutedUntil = now.Add(1 * time.Hour)
	case "rest_of_day":
		// Set to end of today (23:59:59)
		today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		mutedUntil = today
	default:
		helpers.WriteError(
			w,
			http.StatusBadRequest,
			"Invalid duration. Valid values: 30m, 1h, rest_of_day",
		)
		return
	}

	mutedUntilNullTime := sql.NullTime{Time: mutedUntil, Valid: true}
	updatedUser, err := h.authSvc.UpdateUserMutedUntil(ctx, mutedUntilNullTime)
	if err != nil {
		h.logger.Error("failed to update mute status", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to update mute status")
		return
	}

	mutedUntilStr := updatedUser.MutedUntil.Time.Format(time.RFC3339)
	response := MuteStatusResponse{
		MutedUntil: &mutedUntilStr,
		IsMuted:    true,
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// HandleClearMute handles DELETE /api/user/mute
func (h *Handler) HandleClearMute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	mutedUntilNullTime := sql.NullTime{Valid: false}
	_, err := h.authSvc.UpdateUserMutedUntil(ctx, mutedUntilNullTime)
	if err != nil {
		h.logger.Error("failed to clear mute status", zap.Error(err))
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to clear mute status")
		return
	}

	response := MuteStatusResponse{
		IsMuted: false,
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}
