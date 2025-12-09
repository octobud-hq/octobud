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
	"github.com/octobud-hq/octobud/backend/internal/models"
)

const (
	defaultListPage     = 1
	defaultListPageSize = 50
	maxListPageSize     = 200
)

// normalizedPagination extracts and normalizes pagination parameters
func normalizedPagination(opts models.ListOptions) (limit, offset int32, page, pageSize int) {
	page = opts.Page
	if page < 1 {
		page = defaultListPage
	}

	pageSize = opts.PageSize
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	}
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}

	limit = int32(pageSize)

	offset = int32((page - 1) * pageSize)
	return
}
