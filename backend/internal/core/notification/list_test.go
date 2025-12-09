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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestListOptions_normalizedPagination(t *testing.T) {
	tests := []struct {
		name           string
		opts           models.ListOptions
		expectedLimit  int32
		expectedOffset int32
		expectedPage   int
		expectedSize   int
	}{
		{
			name:           "default values",
			opts:           models.ListOptions{},
			expectedLimit:  50,
			expectedOffset: 0,
			expectedPage:   1,
			expectedSize:   50,
		},
		{
			name:           "valid page and page size",
			opts:           models.ListOptions{Page: 2, PageSize: 25},
			expectedLimit:  25,
			expectedOffset: 25,
			expectedPage:   2,
			expectedSize:   25,
		},
		{
			name:           "page less than 1 defaults to 1",
			opts:           models.ListOptions{Page: 0, PageSize: 25},
			expectedLimit:  25,
			expectedOffset: 0,
			expectedPage:   1,
			expectedSize:   25,
		},
		{
			name:           "page size less than 1 defaults to 50",
			opts:           models.ListOptions{Page: 1, PageSize: 0},
			expectedLimit:  50,
			expectedOffset: 0,
			expectedPage:   1,
			expectedSize:   50,
		},
		{
			name:           "page size greater than max defaults to max",
			opts:           models.ListOptions{Page: 1, PageSize: 500},
			expectedLimit:  200,
			expectedOffset: 0,
			expectedPage:   1,
			expectedSize:   200,
		},
		{
			name:           "page 3 with page size 30",
			opts:           models.ListOptions{Page: 3, PageSize: 30},
			expectedLimit:  30,
			expectedOffset: 60,
			expectedPage:   3,
			expectedSize:   30,
		},
		{
			name:           "negative page defaults to 1",
			opts:           models.ListOptions{Page: -1, PageSize: 25},
			expectedLimit:  25,
			expectedOffset: 0,
			expectedPage:   1,
			expectedSize:   25,
		},
		{
			name:           "negative page size defaults to 50",
			opts:           models.ListOptions{Page: 1, PageSize: -10},
			expectedLimit:  50,
			expectedOffset: 0,
			expectedPage:   1,
			expectedSize:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset, page, pageSize := normalizedPagination(tt.opts)
			require.Equal(t, tt.expectedLimit, limit)
			require.Equal(t, tt.expectedOffset, offset)
			require.Equal(t, tt.expectedPage, page)
			require.Equal(t, tt.expectedSize, pageSize)
		})
	}
}
