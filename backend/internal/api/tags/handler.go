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

// Package tags provides the tag service.
package tags

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/tag"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToListTags             = errors.New("failed to list tags")
	ErrFailedToCalculateUnreadCount = errors.New("failed to calculate unread count")
	ErrFailedToDecodeRequest        = errors.New("failed to decode request")
	ErrFailedToCreateTag            = errors.New("failed to create tag")
	ErrFailedToReorderTags          = errors.New("failed to reorder tags")
	ErrFailedToUpdateTag            = errors.New("failed to update tag")
	ErrFailedToDeleteTag            = errors.New("failed to delete tag")
)

// Handler handles tag-related HTTP routes
type Handler struct {
	logger  *zap.Logger
	tagSvc  tag.TagService
	authSvc authsvc.AuthService
}

// New creates a new tags handler
func New(logger *zap.Logger, tagSvc tag.TagService, authSvc authsvc.AuthService) *Handler {
	return &Handler{
		logger:  logger,
		tagSvc:  tagSvc,
		authSvc: authSvc,
	}
}

// Register registers tag routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Route("/tags", func(r chi.Router) {
		r.Get("/", h.handleListAllTags)
		r.Post("/", h.handleCreateTag)
		r.Post("/reorder", h.handleReorderTags)
		r.Put("/{id}", h.handleUpdateTag)
		r.Delete("/{id}", h.handleDeleteTag)
	})
}

type createTagRequest struct {
	Name        string  `json:"name"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

type updateTagRequest struct {
	Name        string  `json:"name"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

type reorderTagsRequest struct {
	TagIDs []string `json:"tagIDs"`
}

// handleListAllTags returns all available tags
func (h *Handler) handleListAllTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	tags, err := h.tagSvc.ListTagsWithUnreadCounts(ctx, userID)
	if err != nil {
		h.logger.Error(
			"handleListAllTags - failed to list tags",
			zap.Error(errors.Join(ErrFailedToListTags, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to list tags")
		return
	}

	shared.WriteJSON(w, http.StatusOK, listTagsResponse{Tags: tags})
}

// handleCreateTag creates a new tag
func (h *Handler) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		shared.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	tag, err := h.tagSvc.CreateTag(ctx, userID, req.Name, req.Color, req.Description)
	if err != nil {
		h.logger.Error(
			"handleCreateTag - failed to create tag",
			zap.Error(errors.Join(ErrFailedToCreateTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to create tag")
		return
	}

	// Convert db.Tag to TagResponse
	tagResp := models.TagFromDB(tag)
	shared.WriteJSON(w, http.StatusCreated, tagEnvelope{Tag: tagResp})
}

// handleUpdateTag updates an existing tag
func (h *Handler) handleUpdateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	tagID := chi.URLParam(r, "id")
	if tagID == "" {
		shared.WriteError(w, http.StatusBadRequest, "tag ID is required")
		return
	}

	var req updateTagRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		shared.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	tag, err := h.tagSvc.UpdateTag(ctx, userID, tagID, req.Name, req.Color, req.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusNotFound, "tag not found")
			return
		}
		h.logger.Error(
			"handleUpdateTag - failed to update tag",
			zap.Error(errors.Join(ErrFailedToUpdateTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to update tag")
		return
	}

	// Convert db.Tag to TagResponse
	tagResp := models.TagFromDB(tag)
	shared.WriteJSON(w, http.StatusOK, tagEnvelope{Tag: tagResp})
}

// handleDeleteTag deletes a tag
func (h *Handler) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	tagID := chi.URLParam(r, "id")
	if tagID == "" {
		shared.WriteError(w, http.StatusBadRequest, "tag ID is required")
		return
	}

	err := h.tagSvc.DeleteTag(ctx, userID, tagID)
	if err != nil {
		h.logger.Error(
			"handleDeleteTag - failed to delete tag",
			zap.Error(errors.Join(ErrFailedToDeleteTag, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete tag")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleReorderTags updates the display order of tags
func (h *Handler) handleReorderTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req reorderTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.TagIDs) == 0 {
		shared.WriteError(w, http.StatusBadRequest, "tagIDs is required")
		return
	}

	// Tag IDs are now UUIDs (strings), no need to parse as int64
	tags, err := h.tagSvc.ReorderTags(ctx, userID, req.TagIDs)
	if err != nil {
		h.logger.Error(
			"handleReorderTags - failed to reorder tags",
			zap.Error(errors.Join(ErrFailedToReorderTags, err)),
		)
		shared.WriteError(w, http.StatusInternalServerError, "failed to reorder tags")
		return
	}

	tagResponses := make([]TagResponse, len(tags))
	for i, tag := range tags {
		tagResponses[i] = models.TagFromDB(tag)
	}

	shared.WriteJSON(w, http.StatusOK, listTagsResponse{Tags: tagResponses})
}
