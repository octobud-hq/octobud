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

// Package rules provides the business logic for rules.
package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
)

// Error definitions
var (
	ErrFailedToGetRulesByViewID      = errors.New("failed to get rules by view ID")
	ErrFailedToLoadRules             = errors.New("failed to load rules")
	ErrRuleNotFound                  = errors.New("rule not found")
	ErrFailedToGetRule               = errors.New("failed to get rule")
	ErrFailedToVerifyView            = errors.New("failed to verify view")
	ErrViewNotFound                  = errors.New("view not found")
	ErrInvalidQuery                  = errors.New("invalid query")
	ErrFailedToProcessActions        = errors.New("failed to process actions")
	ErrFailedToDetermineDisplayOrder = errors.New("failed to determine display order")
	ErrFailedToCreateRule            = errors.New("failed to create rule")
	ErrFailedToUpdateRule            = errors.New("failed to update rule")
	ErrFailedToDeleteRule            = errors.New("failed to delete rule")
	ErrFailedToReorderRules          = errors.New("failed to reorder rules")
	ErrRuleNameAlreadyExists         = errors.New("a rule with that name already exists")
	// Validation errors
	ErrNameRequired                    = errors.New("name is required")
	ErrNameCannotBeEmpty               = errors.New("name cannot be empty")
	ErrQueryCannotBeEmpty              = errors.New("query cannot be empty")
	ErrQueryOrViewIDRequired           = errors.New("either query or viewId is required")
	ErrQueryAndViewIDMutuallyExclusive = errors.New("only one of query or viewId can be provided")
	ErrInvalidViewID                   = errors.New("invalid viewId")
)

// GetRulesByViewID returns all rules linked to a view
func (s *Service) GetRulesByViewID(
	ctx context.Context,
	userID string,
	viewID string,
) ([]db.Rule, error) {
	rules, err := s.queries.GetRulesByViewID(
		ctx,
		userID,
		sql.NullString{String: viewID, Valid: true},
	)
	if err != nil {
		return nil, errors.Join(ErrFailedToGetRulesByViewID, err)
	}
	return rules, nil
}

// ListRules returns all rules
func (s *Service) ListRules(ctx context.Context, userID string) ([]models.Rule, error) {
	rules, err := s.queries.ListRules(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadRules, err)
	}

	response := make([]models.Rule, 0, len(rules))
	for _, rule := range rules {
		response = append(response, models.RuleFromDB(rule))
	}

	return response, nil
}

// GetRule returns a single rule by ID
func (s *Service) GetRule(ctx context.Context, userID string, ruleID string) (models.Rule, error) {
	rule, err := s.queries.GetRule(ctx, userID, ruleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Rule{}, errors.Join(ErrRuleNotFound, err)
		}
		return models.Rule{}, errors.Join(ErrFailedToGetRule, err)
	}

	return models.RuleFromDB(rule), nil
}

// CreateRule creates a new rule
func (s *Service) CreateRule(
	ctx context.Context,
	userID string,
	params models.CreateRuleParams,
) (models.Rule, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return models.Rule{}, ErrNameRequired
	}

	// Validate that exactly one of query or viewId is provided
	hasQuery := params.Query != nil
	hasViewID := params.ViewID != nil

	if !hasQuery && !hasViewID {
		return models.Rule{}, ErrQueryOrViewIDRequired
	}

	if hasQuery && hasViewID {
		return models.Rule{}, ErrQueryAndViewIDMutuallyExclusive
	}

	var viewID sql.NullString
	var queryStr string

	if hasViewID {
		viewIDStr := strings.TrimSpace(*params.ViewID)
		if viewIDStr == "" {
			return models.Rule{}, ErrInvalidViewID
		}

		// Verify the view exists
		_, err := s.queries.GetView(ctx, userID, viewIDStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return models.Rule{}, errors.Join(ErrViewNotFound, err)
			}
			return models.Rule{}, errors.Join(ErrFailedToVerifyView, err)
		}

		viewID = sql.NullString{String: viewIDStr, Valid: true}
	} else {
		queryStr = strings.TrimSpace(*params.Query)
		if queryStr == "" {
			return models.Rule{}, ErrQueryCannotBeEmpty
		}
		// Validate the query by attempting to parse and validate it
		if _, err := query.ParseAndValidate(queryStr); err != nil {
			return models.Rule{}, errors.Join(ErrInvalidQuery, err)
		}
	}

	// Marshal actions to JSON
	actionsJSON, err := json.Marshal(params.Actions)
	if err != nil {
		return models.Rule{}, errors.Join(ErrFailedToProcessActions, err)
	}

	enabled := true
	if params.Enabled != nil {
		enabled = *params.Enabled
	}

	// Get the next display order (max + 1)
	rules, err := s.queries.ListRules(ctx, userID)
	if err != nil {
		return models.Rule{}, errors.Join(ErrFailedToDetermineDisplayOrder, err)
	}

	maxOrder := int32(0)
	for _, r := range rules {
		if r.DisplayOrder > maxOrder {
			maxOrder = r.DisplayOrder
		}
	}
	displayOrder := maxOrder + 100

	dbParams := db.CreateRuleParams{
		Name:         name,
		Description:  models.StringPtrToNull(params.Description),
		Query:        sql.NullString{String: queryStr, Valid: hasQuery},
		ViewID:       viewID,
		Enabled:      enabled,
		Actions:      actionsJSON,
		DisplayOrder: displayOrder,
	}

	rule, err := s.queries.CreateRule(ctx, userID, dbParams)
	if err != nil {
		if models.IsUniqueViolation(err) {
			return models.Rule{}, errors.Join(ErrRuleNameAlreadyExists, err)
		}
		return models.Rule{}, errors.Join(ErrFailedToCreateRule, err)
	}

	return models.RuleFromDB(rule), nil
}

// UpdateRule updates an existing rule
func (s *Service) UpdateRule(
	ctx context.Context,
	userID string,
	ruleID string,
	params models.UpdateRuleParams,
) (models.Rule, error) {
	dbParams := db.UpdateRuleParams{
		ID: ruleID,
	}

	if params.Name != nil {
		name := strings.TrimSpace(*params.Name)
		if name == "" {
			return models.Rule{}, ErrNameCannotBeEmpty
		}
		dbParams.Name = sql.NullString{String: name, Valid: true}
	}
	if params.Description != nil {
		dbParams.Description = sql.NullString{
			String: strings.TrimSpace(*params.Description),
			Valid:  true,
		}
	}
	if params.Query != nil {
		queryStr := strings.TrimSpace(*params.Query)
		if queryStr == "" {
			return models.Rule{}, ErrQueryCannotBeEmpty
		}
		// Validate the query by attempting to parse and validate it
		if _, err := query.ParseAndValidate(queryStr); err != nil {
			return models.Rule{}, errors.Join(ErrInvalidQuery, err)
		}
		dbParams.Query = sql.NullString{String: queryStr, Valid: true}
		// Clear viewId when setting query (mutual exclusivity)
		dbParams.ClearViewID = sql.NullBool{Bool: true, Valid: true}
	}
	if params.ViewID != nil {
		viewIDStr := strings.TrimSpace(*params.ViewID)
		if viewIDStr == "" {
			dbParams.ClearViewID = sql.NullBool{Bool: true, Valid: true}
		} else {
			// Verify the view exists
			_, err := s.queries.GetView(ctx, userID, viewIDStr)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return models.Rule{}, errors.Join(ErrViewNotFound, err)
				}
				return models.Rule{}, errors.Join(ErrFailedToVerifyView, err)
			}
			dbParams.ViewID = sql.NullString{String: viewIDStr, Valid: true}
			// Clear query when setting viewId (mutual exclusivity)
			dbParams.ClearQuery = sql.NullBool{Bool: true, Valid: true}
		}
	}
	if params.Actions != nil {
		actionsJSON, err := json.Marshal(*params.Actions)
		if err != nil {
			return models.Rule{}, errors.Join(ErrFailedToProcessActions, err)
		}
		dbParams.Actions = db.NullRawMessage{RawMessage: actionsJSON, Valid: true}
	}
	if params.Enabled != nil {
		dbParams.Enabled = sql.NullBool{Bool: *params.Enabled, Valid: true}
	}

	rule, err := s.queries.UpdateRule(ctx, userID, dbParams)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Rule{}, errors.Join(ErrRuleNotFound, err)
		}
		if models.IsUniqueViolation(err) {
			return models.Rule{}, errors.Join(ErrRuleNameAlreadyExists, err)
		}
		return models.Rule{}, errors.Join(ErrFailedToUpdateRule, err)
	}

	return models.RuleFromDB(rule), nil
}

// DeleteRule deletes a rule
func (s *Service) DeleteRule(ctx context.Context, userID string, ruleID string) error {
	err := s.queries.DeleteRule(ctx, userID, ruleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.Join(ErrRuleNotFound, err)
		}
		return errors.Join(ErrFailedToDeleteRule, err)
	}
	return nil
}

// ReorderRules updates the display order of rules
func (s *Service) ReorderRules(
	ctx context.Context,
	userID string,
	ruleIDs []string,
) ([]models.Rule, error) {
	if len(ruleIDs) == 0 {
		return nil, fmt.Errorf("ruleIDs cannot be empty")
	}

	// Update display order for each rule
	// Use increments of 100 to allow for future insertions
	for i, ruleID := range ruleIDs {
		displayOrder := int32((i + 1) * 100)
		err := s.queries.UpdateRuleOrder(ctx, userID, db.UpdateRuleOrderParams{
			ID:           ruleID,
			DisplayOrder: displayOrder,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.Join(ErrRuleNotFound, err)
			}
			return nil, errors.Join(ErrFailedToReorderRules, err)
		}
	}

	// Return the updated rules list
	rules, err := s.queries.ListRules(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadRules, err)
	}

	ruleResponses := make([]models.Rule, len(rules))
	for i, rule := range rules {
		ruleResponses[i] = models.RuleFromDB(rule)
	}

	return ruleResponses, nil
}
