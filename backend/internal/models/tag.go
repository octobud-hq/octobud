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

package models

import (
	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Tag represents tag data in domain models and API responses
type Tag struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
	UnreadCount *int64  `json:"unreadCount,omitempty"`
}

// TagFromDB converts a db.Tag to a models.Tag
func TagFromDB(tag db.Tag) Tag {
	resp := Tag{
		ID:   tag.ID, // tag.ID is now a UUID string
		Name: tag.Name,
		Slug: tag.Slug,
	}
	if tag.Color.Valid {
		resp.Color = &tag.Color.String
	}
	if tag.Description.Valid {
		resp.Description = &tag.Description.String
	}
	return resp
}
