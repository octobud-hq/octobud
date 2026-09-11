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

package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/octobud-hq/octobud/backend/internal/db"
)

func TestApplyRepositoryFilter(t *testing.T) {
	t.Run("no repository IDs leaves query untouched", func(t *testing.T) {
		q := db.NotificationQuery{Where: []string{"n.is_read = 0"}, Args: []interface{}{}}
		got := ApplyRepositoryFilter(q, nil)
		require.Equal(t, []string{"n.is_read = 0"}, got.Where)
		require.Empty(t, got.Args)
	})

	t.Run("appends IN clause and args after existing conditions", func(t *testing.T) {
		q := db.NotificationQuery{
			Where: []string{"r.full_name LIKE ?"},
			Args:  []interface{}{"%cli%"},
		}
		got := ApplyRepositoryFilter(q, []int64{7, 42})
		require.Equal(t, []string{"r.full_name LIKE ?", "n.repository_id IN (?, ?)"}, got.Where)
		require.Equal(t, []interface{}{"%cli%", int64(7), int64(42)}, got.Args)
	})

	t.Run("does not mutate the caller's slices", func(t *testing.T) {
		where := make([]string, 0, 4)
		where = append(where, "n.muted = 0")
		args := make([]interface{}, 0, 4)
		q := db.NotificationQuery{Where: where, Args: args}
		_ = ApplyRepositoryFilter(q, []int64{1})
		require.Equal(t, []string{"n.muted = 0"}, where)
		require.Empty(t, args)
	})
}
