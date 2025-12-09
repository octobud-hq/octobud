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

package notification

import (
	"errors"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
	"github.com/octobud-hq/octobud/backend/internal/query/eval"
)

// Error definitions
var (
	ErrFailedToFetchTags = errors.New("failed to fetch tags")
)

// computeActionHintsForNotification computes action hints for a single notification
// using a pre-created evaluator (can be nil if query parsing failed)
func computeActionHintsForNotification(
	notification *db.Notification,
	repo *db.Repository,
	evaluator *eval.Evaluator,
) *models.ActionHints {
	// If evaluator is nil (query parsing failed), return conservative hints
	if evaluator == nil {
		return &models.ActionHints{DismissedOn: []string{}}
	}

	// Use the pre-created evaluator to compute hints efficiently
	hints := query.ComputeActionHintsWithEvaluator(notification, repo, evaluator)

	return &models.ActionHints{
		DismissedOn: hints.DismissedOn,
	}
}
