// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"encoding/json"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Rule represents a rule with all its data
type Rule struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Description  *string     `json:"description,omitempty"`
	Query        string      `json:"query"`
	ViewID       *string     `json:"viewId,omitempty"`
	Actions      RuleActions `json:"actions"`
	Enabled      bool        `json:"enabled"`
	DisplayOrder int         `json:"displayOrder"`
	CreatedAt    string      `json:"createdAt"`
	UpdatedAt    string      `json:"updatedAt"`
}

// CreateRuleParams contains parameters for creating a rule
type CreateRuleParams struct {
	Name            string
	Description     *string
	Query           *string
	ViewID          *string
	Actions         RuleActions
	Enabled         *bool
	ApplyToExisting bool
}

// UpdateRuleParams contains parameters for updating a rule
type UpdateRuleParams struct {
	Name        *string
	Description *string
	Query       *string
	ViewID      *string
	Actions     *RuleActions
	Enabled     *bool
}

// RuleFromDB converts a db.Rule to a models.Rule
func RuleFromDB(rule db.Rule) Rule {
	var actions RuleActions
	if len(rule.Actions) > 0 {
		_ = json.Unmarshal(rule.Actions, &actions)
	}

	var viewID *string
	if rule.ViewID.Valid {
		viewID = &rule.ViewID.String
	}

	// Extract query string from NullString
	queryStr := ""
	if rule.Query.Valid {
		queryStr = rule.Query.String
	}

	return Rule{
		ID:           rule.ID, // Now a UUID string
		Name:         rule.Name,
		Description:  NullStringPtr(rule.Description),
		Query:        queryStr,
		ViewID:       viewID,
		Actions:      actions,
		Enabled:      rule.Enabled,
		DisplayOrder: int(rule.DisplayOrder),
		CreatedAt:    rule.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    rule.UpdatedAt.Format(time.RFC3339),
	}
}
