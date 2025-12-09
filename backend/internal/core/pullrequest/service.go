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

package pullrequest

import (
	"context"
	"errors"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Error definitions
var (
	ErrFailedToUpsertPullRequest = errors.New("failed to upsert pull request")
)

// PullRequestService is the interface for the pull request service.
//
//nolint:revive // exported type name stutters with package name
type PullRequestService interface {
	UpsertPullRequest(
		ctx context.Context,
		userID string,
		params db.UpsertPullRequestParams,
	) (db.PullRequest, error)
}

// Service provides business logic for pull request operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
