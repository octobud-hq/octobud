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

// Package repository provides the business logic for repositories.
package repository

import (
	"context"
	"errors"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrFailedToLoadRepositories      = errors.New("failed to load repositories")
	ErrFailedToListRepositoriesAsMap = errors.New("failed to list repositories as map")
	ErrFailedToUpsertRepository      = errors.New("failed to upsert repository")
)

// ListRepositories returns all repositories
func (s *Service) ListRepositories(
	ctx context.Context,
	userID string,
) ([]models.Repository, error) {
	repositories, err := s.queries.ListRepositories(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadRepositories, err)
	}

	response := make([]models.Repository, 0, len(repositories))
	for _, repository := range repositories {
		response = append(response, models.RepositoryFromDB(repository))
	}

	return response, nil
}

// ListRepositoriesAsMap returns all repositories as a map keyed by ID for efficient lookup
func (s *Service) ListRepositoriesAsMap(
	ctx context.Context,
	userID string,
) (map[int64]db.Repository, error) {
	repositories, err := s.queries.ListRepositories(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToListRepositoriesAsMap, err)
	}

	result := make(map[int64]db.Repository, len(repositories))
	for _, repository := range repositories {
		result[repository.ID] = repository
	}

	return result, nil
}

// GetRepositoryByID returns a repository by ID
func (s *Service) GetRepositoryByID(
	ctx context.Context,
	userID string,
	id int64,
) (db.Repository, error) {
	return s.queries.GetRepositoryByID(ctx, userID, id)
}

// UpsertRepository creates or updates a repository
func (s *Service) UpsertRepository(
	ctx context.Context,
	userID string,
	params db.UpsertRepositoryParams,
) (db.Repository, error) {
	repo, err := s.queries.UpsertRepository(ctx, userID, params)
	if err != nil {
		return db.Repository{}, errors.Join(ErrFailedToUpsertRepository, err)
	}
	return repo, nil
}
