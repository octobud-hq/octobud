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

// Package pullrequest provides the business logic for pull requests.
package pullrequest

import (
	"context"
	"errors"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// UpsertPullRequest creates or updates a pull request
func (s *Service) UpsertPullRequest(
	ctx context.Context,
	userID string,
	params db.UpsertPullRequestParams,
) (db.PullRequest, error) {
	pr, err := s.queries.UpsertPullRequest(ctx, userID, params)
	if err != nil {
		return db.PullRequest{}, errors.Join(ErrFailedToUpsertPullRequest, err)
	}
	return pr, nil
}
