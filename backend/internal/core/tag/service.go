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

// Package tag provides the tag service.
package tag

import (
	"context"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// TagService is the interface for the tag service.
//
//nolint:revive // exported type name stutters with package name
type TagService interface {
	ListTagsWithUnreadCounts(ctx context.Context, userID string) ([]models.Tag, error)
	CreateTag(ctx context.Context, userID, name string, color, description *string) (db.Tag, error)
	UpdateTag(
		ctx context.Context,
		userID, tagID, name string,
		color, description *string,
	) (db.Tag, error)
	DeleteTag(ctx context.Context, userID, tagID string) error
	ReorderTags(ctx context.Context, userID string, tagIDs []string) ([]db.Tag, error)
	GetTag(ctx context.Context, userID, tagID string) (db.Tag, error)
	GetTagByName(ctx context.Context, userID, name string) (db.Tag, error)
	ListTags(ctx context.Context, userID string) ([]db.Tag, error)
}

// Service provides business logic for tag operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
