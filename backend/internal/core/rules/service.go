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

package rules

import (
	"context"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// RuleService is the interface for the rule service.
type RuleService interface {
	GetRulesByViewID(ctx context.Context, userID string, viewID string) ([]db.Rule, error)
	ListRules(ctx context.Context, userID string) ([]models.Rule, error)
	GetRule(ctx context.Context, userID string, ruleID string) (models.Rule, error)
	CreateRule(
		ctx context.Context,
		userID string,
		params models.CreateRuleParams,
	) (models.Rule, error)
	UpdateRule(
		ctx context.Context,
		userID string,
		ruleID string,
		params models.UpdateRuleParams,
	) (models.Rule, error)
	DeleteRule(ctx context.Context, userID string, ruleID string) error
	ReorderRules(ctx context.Context, userID string, ruleIDs []string) ([]models.Rule, error)
}

// Service provides business logic for rule operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
