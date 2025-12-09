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

// Package rules provides the rule service.
package rules

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/shared"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	rulescore "github.com/octobud-hq/octobud/backend/internal/core/rules"
	"github.com/octobud-hq/octobud/backend/internal/core/view"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToQueueApplyRuleJob = errors.New("failed to queue apply_rule job")
)

// Handler handles rule-related HTTP routes
type Handler struct {
	logger    *zap.Logger
	ruleSvc   rulescore.RuleService
	viewSvc   view.ViewService
	scheduler jobs.Scheduler
	authSvc   authsvc.AuthService
}

// NewWithScheduler creates a new rules handler with a scheduler
func NewWithScheduler(
	logger *zap.Logger,
	ruleSvc rulescore.RuleService,
	viewSvc view.ViewService,
	scheduler jobs.Scheduler,
	authSvc authsvc.AuthService,
) *Handler {
	return &Handler{
		logger:    logger,
		ruleSvc:   ruleSvc,
		viewSvc:   viewSvc,
		scheduler: scheduler,
		authSvc:   authSvc,
	}
}

// Register registers rule routes on the provided router
func (h *Handler) Register(r chi.Router) {
	r.Route("/rules", func(r chi.Router) {
		r.Get("/", h.handleListRules)
		r.Post("/", h.handleCreateRule)
		r.Post("/reorder", h.handleReorderRules)
		r.Get("/{id}", h.handleGetRule)
		r.Put("/{id}", h.handleUpdateRule)
		r.Delete("/{id}", h.handleDeleteRule)
	})
}

type createRuleRequest struct {
	Name            string      `json:"name"`
	Description     *string     `json:"description"`
	Query           *string     `json:"query,omitempty"`
	ViewID          *string     `json:"viewId,omitempty"`
	Actions         RuleActions `json:"actions"`
	Enabled         *bool       `json:"enabled"`
	ApplyToExisting bool        `json:"applyToExisting,omitempty"`
}

type updateRuleRequest struct {
	Name        *string      `json:"name"`
	Description *string      `json:"description"`
	Query       *string      `json:"query"`
	ViewID      *string      `json:"viewId,omitempty"`
	Actions     *RuleActions `json:"actions"`
	Enabled     *bool        `json:"enabled"`
}

type reorderRulesRequest struct {
	RuleIDs []string `json:"ruleIDs"`
}

func (h *Handler) handleListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	rules, err := h.ruleSvc.ListRules(ctx, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "failed to load rules")
		return
	}

	response := make([]RuleResponse, 0, len(rules))
	response = append(response, rules...)

	shared.WriteJSON(w, http.StatusOK, listRulesResponse{Rules: response})
}

func (h *Handler) handleGetRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		shared.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	rule, err := h.ruleSvc.GetRule(ctx, userID, ruleID)
	if err != nil {
		if errors.Is(err, rulescore.ErrRuleNotFound) {
			shared.WriteError(w, http.StatusNotFound, "rule not found")
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to get rule")
		return
	}

	shared.WriteJSON(w, http.StatusOK, ruleEnvelope{Rule: rule})
}

func (h *Handler) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req createRuleRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	// Convert request to service params
	params := models.CreateRuleParams{
		Name:            req.Name,
		Description:     req.Description,
		Query:           req.Query,
		ViewID:          req.ViewID,
		Actions:         req.Actions,
		Enabled:         req.Enabled,
		ApplyToExisting: req.ApplyToExisting,
	}

	createdRule, err := h.ruleSvc.CreateRule(ctx, userID, params)
	if err != nil {
		if errors.Is(err, rulescore.ErrRuleNameAlreadyExists) {
			shared.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, rulescore.ErrViewNotFound) || errors.Is(err, rulescore.ErrInvalidQuery) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Check for validation errors
		if errors.Is(err, rulescore.ErrNameRequired) ||
			errors.Is(err, rulescore.ErrQueryOrViewIDRequired) ||
			errors.Is(err, rulescore.ErrQueryAndViewIDMutuallyExclusive) ||
			errors.Is(err, rulescore.ErrQueryCannotBeEmpty) ||
			errors.Is(err, rulescore.ErrInvalidViewID) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}

	// If ApplyToExisting is true, enqueue job to apply rule retroactively
	if req.ApplyToExisting && h.scheduler != nil {
		h.logger.Info("enqueuing apply_rule job", zap.String("rule_id", createdRule.ID))
		if enqueueErr := h.scheduler.EnqueueApplyRule(ctx, userID, createdRule.ID); enqueueErr != nil {
			h.logger.Error("failed to queue apply_rule job",
				zap.String("rule_id", createdRule.ID),
				zap.Error(errors.Join(ErrFailedToQueueApplyRuleJob, enqueueErr)))
			// Don't fail the entire request - rule was created successfully
		} else {
			h.logger.Info("successfully queued apply_rule job", zap.String("rule_id", createdRule.ID))
		}
	}

	shared.WriteJSON(w, http.StatusCreated, ruleEnvelope{Rule: createdRule})
}

func (h *Handler) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		shared.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req updateRuleRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert request to service params
	updateParams := models.UpdateRuleParams{
		Name:        req.Name,
		Description: req.Description,
		Query:       req.Query,
		ViewID:      req.ViewID,
		Enabled:     req.Enabled,
	}
	if req.Actions != nil {
		actions := *req.Actions
		updateParams.Actions = &actions
	}

	updatedRule, err := h.ruleSvc.UpdateRule(ctx, userID, ruleID, updateParams)
	if err != nil {
		if errors.Is(err, rulescore.ErrRuleNotFound) {
			shared.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, rulescore.ErrRuleNameAlreadyExists) {
			shared.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, rulescore.ErrViewNotFound) || errors.Is(err, rulescore.ErrInvalidQuery) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Check for validation errors
		if errors.Is(err, rulescore.ErrNameCannotBeEmpty) ||
			errors.Is(err, rulescore.ErrQueryCannotBeEmpty) ||
			errors.Is(err, rulescore.ErrInvalidViewID) {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to update rule")
		return
	}

	shared.WriteJSON(w, http.StatusOK, ruleEnvelope{Rule: updatedRule})
}

func (h *Handler) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		shared.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.ruleSvc.DeleteRule(ctx, userID, ruleID); err != nil {
		if errors.Is(err, rulescore.ErrRuleNotFound) {
			shared.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleReorderRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := shared.RequireUserID(ctx, w, h.authSvc)
	if !ok {
		return
	}

	var req reorderRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.RuleIDs) == 0 {
		shared.WriteError(w, http.StatusBadRequest, "ruleIDs cannot be empty")
		return
	}

	// Rule IDs are now UUIDs (strings), no need to parse as int64
	rules, err := h.ruleSvc.ReorderRules(ctx, userID, req.RuleIDs)
	if err != nil {
		if errors.Is(err, rulescore.ErrRuleNotFound) {
			shared.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "failed to reorder rules")
		return
	}

	response := make([]RuleResponse, 0, len(rules))
	response = append(response, rules...)

	shared.WriteJSON(w, http.StatusOK, listRulesResponse{Rules: response})
}
