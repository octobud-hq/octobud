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

// Package views provides the views handler.
package views

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/view"
)

// Handler handles view-related HTTP routes
type Handler struct {
	logger  *zap.Logger
	viewSvc view.ViewService
	authSvc authsvc.AuthService
}

// New creates a new views handler
func New(logger *zap.Logger, viewSvc view.ViewService, authSvc authsvc.AuthService) *Handler {
	return &Handler{
		logger:  logger,
		viewSvc: viewSvc,
		authSvc: authSvc,
	}
}

// Register registers view routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Route("/views", func(r chi.Router) {
		r.Get("/", h.handleListViews)
		r.Post("/", h.handleCreateView)
		r.Post("/reorder", h.handleReorderViews)
		r.Put("/{id}", h.handleUpdateView)
		r.Delete("/{id}", h.handleDeleteView)
	})
}
