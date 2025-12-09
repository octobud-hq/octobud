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

// Package oauth provides HTTP handlers for GitHub OAuth Device Flow.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	coregithub "github.com/octobud-hq/octobud/backend/internal/core/github"
)

// TokenManager defines the interface for token management operations
type TokenManager interface {
	SetTokenFromOAuth(ctx context.Context, token string) (string, error)
	ClearToken(ctx context.Context) error
	GetStatusConnected() bool
	GetStatusSource() string
	GetStatusMaskedToken() string
	GetStatusGitHubUsername() string
}

// Handler handles OAuth-related HTTP routes
type Handler struct {
	logger       *zap.Logger
	oauthClient  *coregithub.OAuthClient
	tokenManager TokenManager
}

// New creates a new OAuth handler
func New(logger *zap.Logger, tokenManager TokenManager) *Handler {
	return &Handler{
		logger:       logger,
		oauthClient:  coregithub.NewOAuthClient(),
		tokenManager: tokenManager,
	}
}

// Register registers OAuth routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Route("/oauth", func(r chi.Router) {
		r.Post("/device", h.handleStartDeviceFlow)
		r.Post("/poll", h.handlePollForToken)
	})
}

// DeviceFlowResponse is returned when starting the device flow
type DeviceFlowResponse struct {
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
	DeviceCode      string `json:"deviceCode"`
}

// handleStartDeviceFlow initiates the GitHub OAuth Device Flow
func (h *Handler) handleStartDeviceFlow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceCode, err := h.oauthClient.RequestDeviceCode(ctx)
	if err != nil {
		h.logger.Error("failed to request device code", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to start GitHub authorization")
		return
	}

	response := DeviceFlowResponse{
		UserCode:        deviceCode.UserCode,
		VerificationURI: deviceCode.VerificationURI,
		ExpiresIn:       deviceCode.ExpiresIn,
		Interval:        deviceCode.Interval,
		DeviceCode:      deviceCode.DeviceCode,
	}

	shared.WriteJSON(w, http.StatusOK, response)
}

// PollRequest is the request body for polling
type PollRequest struct {
	DeviceCode string `json:"deviceCode"`
}

// PollResponse is returned when polling for the token
type PollResponse struct {
	Status         string `json:"status"` // "pending", "complete", "expired", "denied"
	GitHubUsername string `json:"githubUsername,omitempty"`
}

// handlePollForToken polls GitHub for the access token
func (h *Handler) handlePollForToken(w http.ResponseWriter, r *http.Request) {
	var req PollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("failed to decode poll request", zap.Error(err))
		shared.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DeviceCode == "" {
		shared.WriteError(w, http.StatusBadRequest, "Device code is required")
		return
	}

	ctx := r.Context()
	tokenResp, err := h.oauthClient.PollForAccessToken(ctx, req.DeviceCode)

	if err != nil {
		// Handle expected polling states
		switch {
		case errors.Is(err, coregithub.ErrAuthorizationPending):
			shared.WriteJSON(w, http.StatusOK, PollResponse{Status: "pending"})
			return
		case errors.Is(err, coregithub.ErrSlowDown):
			// Client should slow down polling
			shared.WriteJSON(w, http.StatusOK, PollResponse{Status: "pending"})
			return
		case errors.Is(err, coregithub.ErrExpiredToken):
			shared.WriteJSON(w, http.StatusOK, PollResponse{Status: "expired"})
			return
		case errors.Is(err, coregithub.ErrAccessDenied):
			shared.WriteJSON(w, http.StatusOK, PollResponse{Status: "denied"})
			return
		default:
			h.logger.Error("failed to poll for token", zap.Error(err))
			shared.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to check authorization status",
			)
			return
		}
	}

	// We have a token - store it
	githubUsername, err := h.tokenManager.SetTokenFromOAuth(ctx, tokenResp.AccessToken)
	if err != nil {
		h.logger.Error("failed to store OAuth token", zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "Failed to save GitHub connection")
		return
	}

	h.logger.Info("GitHub OAuth connected",
		zap.String("github_username", githubUsername),
	)

	shared.WriteJSON(w, http.StatusOK, PollResponse{
		Status:         "complete",
		GitHubUsername: githubUsername,
	})
}
