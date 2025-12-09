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
	"github.com/octobud-hq/octobud/backend/internal/db"
)

// View represents a view with unread count
type View struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Description  *string `json:"description,omitempty"`
	Icon         *string `json:"icon,omitempty"`
	IsDefault    bool    `json:"isDefault"`
	SystemView   bool    `json:"systemView,omitempty"`
	Query        string  `json:"query"`
	UnreadCount  int64   `json:"unreadCount"`
	DisplayOrder int     `json:"displayOrder"`
}

// SystemView represents a system view (inbox, everything, etc.)
type SystemView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Icon        string `json:"icon"`
	Query       string `json:"query"`
	UnreadCount int64  `json:"unreadCount"`
}

// ViewFromDB converts a db.View to a View
func ViewFromDB(view db.View) View {
	query := ""
	if view.Query.Valid {
		query = view.Query.String
	}

	return View{
		ID:           view.ID, // Now a UUID string
		Name:         view.Name,
		Slug:         view.Slug,
		Description:  NullStringPtr(view.Description),
		Icon:         NullStringPtr(view.Icon),
		IsDefault:    view.IsDefault,
		Query:        query,
		DisplayOrder: int(view.DisplayOrder),
	}
}
