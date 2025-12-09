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

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
)

// RuleMatcher applies rules to notifications
type RuleMatcher struct {
	store db.Store
}

// NewRuleMatcher creates a new rule matcher
func NewRuleMatcher(store db.Store) *RuleMatcher {
	return &RuleMatcher{
		store: store,
	}
}

// MatchAndApplyRules checks notification against all enabled rules and applies matching rules
// Returns true if any rule matched
func (rm *RuleMatcher) MatchAndApplyRules(
	ctx context.Context,
	userID string,
	notificationID int64,
) (bool, error) {
	// Get the notification
	notification, err := rm.store.GetNotificationByID(ctx, userID, notificationID)
	if err != nil {
		return false, fmt.Errorf("failed to get notification: %w", err)
	}

	// Fetch all enabled rules ordered by display_order
	rules, err := rm.store.ListEnabledRulesOrdered(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to list enabled rules: %w", err)
	}

	anyMatched := false

	// Check each rule
	for _, rule := range rules {
		matched, err := rm.checkRuleMatch(ctx, userID, notification, rule)
		if err != nil {
			// Skip rules that fail to match - don't fail the entire job
			continue
		}

		if matched {
			anyMatched = true

			// Parse and apply actions
			var actions models.RuleActions
			if len(rule.Actions) > 0 {
				if err := json.Unmarshal(rule.Actions, &actions); err != nil {
					continue
				}
			}

			if err := rm.ApplyRuleActions(ctx, userID, notification.GithubID, actions); err != nil {
				// Continue processing other rules even if one fails
				continue
			}
		}
	}

	return anyMatched, nil
}

// checkRuleMatch checks if a notification matches a rule's query
func (rm *RuleMatcher) checkRuleMatch(
	ctx context.Context,
	userID string,
	notification db.Notification,
	rule db.Rule,
) (bool, error) {
	// Determine the query to use - prefer viewId if both are defined
	var queryStr string

	if rule.ViewID.Valid {
		// Rule is linked to a view - resolve the view's query dynamically
		view, err := rm.store.GetView(ctx, userID, rule.ViewID.String)
		if err != nil {
			return false, fmt.Errorf("failed to get view for rule: %w", err)
		}

		if !view.Query.Valid || view.Query.String == "" {
			return false, fmt.Errorf("view %s has no query defined", view.ID)
		}

		queryStr = view.Query.String
	} else {
		// Rule has its own query (only use if viewId is not set)
		if !rule.Query.Valid || rule.Query.String == "" {
			return false, fmt.Errorf("rule %s has neither query nor viewId set", rule.ID)
		}
		queryStr = rule.Query.String
	}

	// Parse and build the query
	dbQuery, err := query.BuildQuery(queryStr, 1, 0)
	if err != nil {
		return false, fmt.Errorf("failed to build query: %w", err)
	}

	// Add constraint that id must match the notification we're checking
	dbQuery.Where = append(dbQuery.Where, "n.id = ?")
	dbQuery.Args = append(dbQuery.Args, notification.ID)

	// Execute the query
	result, err := rm.store.ListNotificationsFromQuery(ctx, userID, dbQuery)
	if err != nil {
		return false, fmt.Errorf("failed to execute query: %w", err)
	}

	// If we got results, the notification matches
	return result.Total > 0, nil
}

//go:generate mockgen -source=rule_matcher.go -destination=mocks/mock_rule_matcher.go -package=mocks

// RuleMatcherInterface defines the interface for applying rule actions to notifications
type RuleMatcherInterface interface {
	ApplyRuleActions(
		ctx context.Context,
		userID string,
		githubID string,
		actions models.RuleActions,
	) error
}

// ApplyRuleActions applies the actions specified by a rule to a notification
func (rm *RuleMatcher) ApplyRuleActions(
	ctx context.Context,
	userID string,
	githubID string,
	actions models.RuleActions,
) error {
	var errs []error

	if actions.SkipInbox {
		if _, err := rm.store.MarkNotificationFiltered(ctx, userID, githubID); err != nil {
			errs = append(errs, fmt.Errorf("failed to mark filtered: %w", err))
		}
	}

	if actions.MarkRead {
		if _, err := rm.store.MarkNotificationRead(ctx, userID, githubID); err != nil {
			errs = append(errs, fmt.Errorf("failed to mark read: %w", err))
		}
	}

	if actions.Star {
		if _, err := rm.store.StarNotification(ctx, userID, githubID); err != nil {
			errs = append(errs, fmt.Errorf("failed to star: %w", err))
		}
	}

	if actions.Archive {
		if _, err := rm.store.ArchiveNotification(ctx, userID, githubID); err != nil {
			errs = append(errs, fmt.Errorf("failed to archive: %w", err))
		}
	}

	if actions.Mute {
		if _, err := rm.store.MuteNotification(ctx, userID, githubID); err != nil {
			errs = append(errs, fmt.Errorf("failed to mute: %w", err))
		}
	}

	// Get notification ID for tag operations
	if len(actions.AssignTags) > 0 || len(actions.RemoveTags) > 0 {
		notification, err := rm.store.GetNotificationByGithubID(ctx, userID, githubID)
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to get notification for tag operations: %w", err),
			)
		} else {
			// Assign tags by ID (tags now use UUID strings)
			for _, tagID := range actions.AssignTags {
				// Verify tag exists
				_, err = rm.store.GetTag(ctx, userID, tagID)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						errs = append(errs, fmt.Errorf("tag with ID %s not found", tagID))
						continue
					}
					errs = append(errs, fmt.Errorf("failed to get tag %s: %w", tagID, err))
					continue
				}

				// Assign tag to notification
				_, err = rm.store.AssignTagToEntity(ctx, userID, db.AssignTagToEntityParams{
					TagID:      tagID,
					EntityType: "notification",
					EntityID:   notification.ID,
				})
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to assign tag %s: %w", tagID, err))
				}
			}

			// Remove tags by ID (tags now use UUID strings)
			for _, tagID := range actions.RemoveTags {
				// Verify tag exists (optional check, but good for error messages)
				_, err = rm.store.GetTag(ctx, userID, tagID)
				if err != nil {
					if !errors.Is(err, sql.ErrNoRows) {
						errs = append(errs, fmt.Errorf("failed to get tag %s for removal: %w", tagID, err))
					}
					continue
				}

				err = rm.store.RemoveTagAssignment(ctx, userID, db.RemoveTagAssignmentParams{
					TagID:      tagID,
					EntityType: "notification",
					EntityID:   notification.ID,
				})
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to remove tag %s: %w", tagID, err))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some actions failed: %v", errs)
	}

	return nil
}

// MatchAndApplyRulesWithDB is a convenience wrapper that creates a RuleMatcher and applies rules
func MatchAndApplyRulesWithDB(
	ctx context.Context,
	store db.Store,
	userID string,
	notificationID int64,
) (bool, error) {
	matcher := NewRuleMatcher(store)
	return matcher.MatchAndApplyRules(ctx, userID, notificationID)
}
