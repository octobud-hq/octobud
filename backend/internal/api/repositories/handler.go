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

// Package repositories provides the repositories handler.
package repositories

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/repository"
)

// Error definitions
var (
	ErrFailedToLoadRepositories = errors.New("failed to load repositories")
)

// Handler handles repository-related HTTP routes
type Handler struct {
	logger        *zap.Logger
	repositorySvc repository.RepositoryService
	authSvc       authsvc.AuthService
}

// New creates a new repositories handler
func New(
	logger *zap.Logger,
	repositorySvc repository.RepositoryService,
	authSvc authsvc.AuthService,
) *Handler {
	return &Handler{
		logger:        logger,
		repositorySvc: repositorySvc,
		authSvc:       authSvc,
	}
}

// Register registers repository routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Get("/repositories", h.handleListRepositories)
}

func (h *Handler) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	repos, err := h.repositorySvc.ListRepositories(ctx, userID)
	if err != nil {
		h.logger.Error(
			"failed to load repositories",
			zap.Error(errors.Join(ErrFailedToLoadRepositories, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to load repositories")
		return
	}

	shared.WriteJSON(w, http.StatusOK, listRepositoriesResponse{Repositories: repos})
}
