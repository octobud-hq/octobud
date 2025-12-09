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

// Package handlers provides the apply_rule job handler.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
)

// ApplyRuleHandler handles applying a rule to existing notifications.
type ApplyRuleHandler struct {
	store   db.Store
	matcher *RuleMatcher
	logger  *zap.Logger
}

// NewApplyRuleHandler creates a new ApplyRuleHandler.
func NewApplyRuleHandler(store db.Store, logger *zap.Logger) *ApplyRuleHandler {
	return &ApplyRuleHandler{
		store:   store,
		matcher: NewRuleMatcher(store),
		logger:  logger,
	}
}

// Handle applies a rule to all matching notifications.
func (h *ApplyRuleHandler) Handle(ctx context.Context, userID string, ruleID string) error {
	// Fetch the rule
	rule, err := h.store.GetRule(ctx, userID, ruleID)
	if err != nil {
		return fmt.Errorf("failed to get rule %s: %w", ruleID, err)
	}

	// Skip applying if rule is disabled
	if !rule.Enabled {
		return nil
	}

	// Determine the query to use - prefer viewId if both are defined
	var queryStr string

	if rule.ViewID.Valid {
		// Rule is linked to a view - resolve the view's query dynamically
		view, err := h.store.GetView(ctx, userID, rule.ViewID.String)
		if err != nil {
			return fmt.Errorf("failed to get view for rule: %w", err)
		}

		if !view.Query.Valid || view.Query.String == "" {
			return fmt.Errorf("view %s has no query defined", view.ID)
		}

		queryStr = view.Query.String
	} else {
		// Rule has its own query (only use if viewId is not set)
		if !rule.Query.Valid || rule.Query.String == "" {
			return fmt.Errorf("rule %s has neither query nor viewId set", ruleID)
		}
		queryStr = rule.Query.String
	}

	// Build query from rule query string, using in:anywhere to include all notifications
	fullQueryStr := fmt.Sprintf("(%s) AND in:anywhere", queryStr)

	// Build the query - use a reasonable page size for batch processing
	const pageSize = 100
	offset := int32(0)
	totalProcessed := 0

	for {
		dbQuery, err := query.BuildQuery(fullQueryStr, pageSize, offset)
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}

		// Execute query to get matching notifications
		result, err := h.store.ListNotificationsFromQuery(ctx, userID, dbQuery)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}

		if len(result.Notifications) == 0 {
			break // No more notifications to process
		}

		// Apply rule actions to each notification
		for _, notification := range result.Notifications {
			// Parse and apply actions
			var actions models.RuleActions
			if len(rule.Actions) > 0 {
				if err := json.Unmarshal(rule.Actions, &actions); err != nil {
					continue
				}
			}

			if err := h.matcher.ApplyRuleActions(ctx, userID, notification.GithubID, actions); err != nil {
				// Continue processing other notifications even if one fails
				continue
			}
			totalProcessed++
		}

		// Move to next page
		offset += pageSize

		// Safety check to avoid infinite loops
		if totalProcessed >= int(result.Total) {
			break
		}
	}

	return nil
}
