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

package view

import (
	"context"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

var reservedSlugs = map[string]struct{}{
	"inbox":          {},
	"everything":     {},
	"done":           {},
	"archive":        {},
	"snoozed":        {},
	"starred":        {},
	"search_results": {},
}

// ViewService is the interface for the view service.
//
//nolint:revive // exported type name stutters with package name
type ViewService interface {
	ListViewsWithCounts(ctx context.Context, userID string) ([]models.View, error)
	GetView(ctx context.Context, userID string, id string) (db.View, error)
	CreateView(
		ctx context.Context,
		userID, name string,
		description, icon *string,
		isDefault *bool,
		queryStr string,
	) (models.View, error)
	UpdateView(
		ctx context.Context,
		userID string,
		viewID string,
		name, description, icon *string,
		isDefault *bool,
		queryStr *string,
	) (models.View, error)
	DeleteView(
		ctx context.Context,
		userID string,
		viewID string,
		force bool,
	) (linkedRuleCount int, err error)
	ReorderViews(ctx context.Context, userID string, viewIDs []string) ([]models.View, error)
}

// Service implements the ViewService interface, providing
// business logic for view operations.
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
