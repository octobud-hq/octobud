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
	"encoding/json"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

// Repository represents repository data in domain models and API responses
type Repository struct {
	ID          int64   `json:"id"`
	GithubID    *int64  `json:"githubId,omitempty"`
	NodeID      *string `json:"nodeId,omitempty"`
	Name        string  `json:"name"`
	FullName    string  `json:"fullName"`
	OwnerLogin  *string `json:"ownerLogin,omitempty"`
	OwnerID     *int64  `json:"ownerId,omitempty"`
	Private     *bool   `json:"private,omitempty"`
	Description *string `json:"description,omitempty"`

	HTMLURL        *string         `json:"htmlURL,omitempty"`
	Fork           *bool           `json:"fork,omitempty"`
	Visibility     *string         `json:"visibility,omitempty"`
	DefaultBranch  *string         `json:"defaultBranch,omitempty"`
	Archived       *bool           `json:"archived,omitempty"`
	Disabled       *bool           `json:"disabled,omitempty"`
	PushedAt       *time.Time      `json:"pushedAt,omitempty"`
	CreatedAt      *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt      *time.Time      `json:"updatedAt,omitempty"`
	Raw            json.RawMessage `json:"raw,omitempty"`
	OwnerAvatarURL *string         `json:"ownerAvatarURL,omitempty"`

	OwnerHTMLURL *string `json:"ownerHTMLURL,omitempty"`
}

// RepositoryFromDB converts a db.Repository to a Repository
func RepositoryFromDB(repository db.Repository) Repository {
	var raw json.RawMessage
	if repository.Raw.Valid {
		raw = repository.Raw.RawMessage
	}

	return Repository{
		ID:             repository.ID,
		GithubID:       NullInt64Ptr(repository.GithubID),
		NodeID:         NullStringPtr(repository.NodeID),
		Name:           repository.Name,
		FullName:       repository.FullName,
		OwnerLogin:     NullStringPtr(repository.OwnerLogin),
		OwnerID:        NullInt64Ptr(repository.OwnerID),
		Private:        NullBoolPtr(repository.Private),
		Description:    NullStringPtr(repository.Description),
		HTMLURL:        NullStringPtr(repository.HTMLURL),
		Fork:           NullBoolPtr(repository.Fork),
		Visibility:     NullStringPtr(repository.Visibility),
		DefaultBranch:  NullStringPtr(repository.DefaultBranch),
		Archived:       NullBoolPtr(repository.Archived),
		Disabled:       NullBoolPtr(repository.Disabled),
		PushedAt:       NullTimePtr(repository.PushedAt),
		CreatedAt:      NullTimePtr(repository.CreatedAt),
		UpdatedAt:      NullTimePtr(repository.UpdatedAt),
		Raw:            raw,
		OwnerAvatarURL: NullStringPtr(repository.OwnerAvatarURL),
		OwnerHTMLURL:   NullStringPtr(repository.OwnerHTMLURL),
	}
}
