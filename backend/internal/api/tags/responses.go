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

package tags

import (
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// TagResponse is the response type for a tag.
type TagResponse = models.Tag

// listTagsResponse is the response type for a list of tags.
type listTagsResponse struct {
	Tags []TagResponse `json:"tags"`
}

// tagEnvelope is the envelope type for a tag.
type tagEnvelope struct {
	Tag TagResponse `json:"tag"`
}
