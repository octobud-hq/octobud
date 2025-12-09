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

package repository

import (
	"context"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// RepositoryService is the interface for the repository service.
//
//nolint:revive // exported type name stutters with package name
type RepositoryService interface {
	ListRepositories(ctx context.Context, userID string) ([]models.Repository, error)
	ListRepositoriesAsMap(ctx context.Context, userID string) (map[int64]db.Repository, error)
	GetRepositoryByID(ctx context.Context, userID string, id int64) (db.Repository, error)
	UpsertRepository(
		ctx context.Context,
		userID string,
		params db.UpsertRepositoryParams,
	) (db.Repository, error)
}

// Service provides business logic for repository operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
