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

package views

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	viewcore "github.com/octobud-hq/octobud/backend/internal/core/view"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

type createViewRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	IsDefault   *bool   `json:"isDefault"`
	Query       string  `json:"query"`
}

type updateViewRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	IsDefault   *bool   `json:"isDefault"`
	Query       *string `json:"query"`
}

type reorderViewsRequest struct {
	ViewIDs []string `json:"viewIDs"`
}

func (h *Handler) handleListViews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	views, err := h.viewSvc.ListViewsWithCounts(ctx, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "failed to load views")
		return
	}

	response := make([]ViewResponse, 0, len(views))
	response = append(response, views...)

	shared.WriteJSON(w, http.StatusOK, listViewsResponse{Views: response})
}

func (h *Handler) handleCreateView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req createViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	queryStr := strings.TrimSpace(req.Query)

	view, err := h.viewSvc.CreateView(
		ctx,
		userID,
		req.Name,
		req.Description,
		req.Icon,
		req.IsDefault,
		queryStr,
	)
	if err != nil {
		// Check for unique violation first (both wrapped and unwrapped)
		if models.IsUniqueViolation(err) {
			h.logger.Debug(
				"unique violation detected in create view",
				zap.String("name", req.Name),
				zap.Error(err),
			)
			shared.WriteError(w, http.StatusConflict, "a view with that name already exists")
			return
		}
		if errors.Is(err, viewcore.ErrViewNameAlreadyExists) {
			h.logger.Debug("view name already exists", zap.String("name", req.Name), zap.Error(err))
			shared.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, viewcore.ErrInvalidQuery) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Check for validation errors
		if errors.Is(err, viewcore.ErrNameRequired) ||
			errors.Is(err, viewcore.ErrNameMustContainAlphanumeric) ||
			errors.Is(err, viewcore.ErrSlugReserved) ||
			errors.Is(err, viewcore.ErrQueryRequired) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Log unexpected errors for debugging
		h.logger.Error("failed to create view", zap.String("name", req.Name), zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "failed to create view")
		return
	}

	shared.WriteJSON(w, http.StatusCreated, viewEnvelope{View: view})
}

func (h *Handler) handleUpdateView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	viewID, err := parseViewIDParam(r)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req updateViewRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var name, description, icon, queryStr *string
	if req.Name != nil {
		nameStr := strings.TrimSpace(*req.Name)
		name = &nameStr
	}
	if req.Description != nil {
		descStr := strings.TrimSpace(*req.Description)
		description = &descStr
	}
	if req.Icon != nil {
		iconStr := strings.TrimSpace(*req.Icon)
		icon = &iconStr
	}
	if req.Query != nil {
		queryStrVal := strings.TrimSpace(*req.Query)
		queryStr = &queryStrVal
	}

	view, err := h.viewSvc.UpdateView(
		ctx,
		userID,
		viewID,
		name,
		description,
		icon,
		req.IsDefault,
		queryStr,
	)
	if err != nil {
		if errors.Is(err, viewcore.ErrViewNotFound) {
			shared.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		// Check for unique violation first (both wrapped and unwrapped)
		if models.IsUniqueViolation(err) {
			h.logger.Debug(
				"unique violation detected in update view",
				zap.String("view_id", viewID),
				zap.Error(err),
			)
			shared.WriteError(w, http.StatusConflict, "a view with that name already exists")
			return
		}
		if errors.Is(err, viewcore.ErrViewNameAlreadyExists) {
			h.logger.Debug(
				"view name already exists",
				zap.String("view_id", viewID),
				zap.Error(err),
			)
			shared.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, viewcore.ErrInvalidQuery) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Check for validation errors
		if errors.Is(err, viewcore.ErrNameCannotBeEmpty) ||
			errors.Is(err, viewcore.ErrNameMustContainAlphanumeric) ||
			errors.Is(err, viewcore.ErrSlugReserved) ||
			errors.Is(err, viewcore.ErrQueryCannotBeEmpty) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Log unexpected errors for debugging
		h.logger.Error("failed to update view", zap.String("view_id", viewID), zap.Error(err))
		shared.WriteError(w, http.StatusInternalServerError, "failed to update view")
		return
	}

	shared.WriteJSON(w, http.StatusOK, viewEnvelope{View: view})
}

func (h *Handler) handleDeleteView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	viewID, err := parseViewIDParam(r)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if force parameter is set
	force := r.URL.Query().Get("force") == "true"

	// Check if there are any rules linked to this view
	linkedRuleCount, err := h.viewSvc.DeleteView(ctx, userID, viewID, force)
	if err != nil {
		if errors.Is(err, viewcore.ErrViewNotFound) {
			shared.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete view")
		return
	}

	// If there are linked rules and force is not set, return an error with the count
	// The frontend will show a confirmation dialog
	if linkedRuleCount > 0 && !force {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":           "This view has linked rules that will also be deleted",
			"linkedRuleCount": linkedRuleCount,
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleReorderViews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req reorderViewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.ViewIDs) == 0 {
		shared.WriteError(w, http.StatusBadRequest, "viewIDs cannot be empty")
		return
	}

	// View IDs are now UUIDs (strings), no need to parse as int64
	views, err := h.viewSvc.ReorderViews(ctx, userID, req.ViewIDs)
	if err != nil {
		if errors.Is(err, viewcore.ErrViewNotFound) {
			shared.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		// Check for validation errors
		if errors.Is(err, viewcore.ErrCannotReorderSystemView) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to reorder views")
		return
	}

	response := make([]ViewResponse, 0, len(views))

	response = append(response, views...)

	shared.WriteJSON(w, http.StatusOK, listViewsResponse{Views: response})
}

func parseViewIDParam(r *http.Request) (string, error) {
	rawID := chi.URLParam(r, "id")
	if rawID == "" {
		return "", errors.New("id is required")
	}
	// View IDs are now UUIDs (strings), just validate it's not empty
	return rawID, nil
}
