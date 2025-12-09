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

package sync

import (
	"context"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/github/types"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// getUserSyncSettings retrieves sync settings from the user
func (s *Service) getUserSyncSettings(ctx context.Context) (*models.SyncSettings, error) {
	user, err := s.userStore.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	return models.SyncSettingsFromJSON(user.SyncSettings.RawMessage)
}

// calculateSyncSinceDate calculates the timestamp for N days ago
func calculateSyncSinceDate(days int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -days)
}

// applyInitialSyncLimitsFromContext applies sync limits from a SyncContext.
// This is the preferred method - it takes pre-computed limits from the context.
// Note: Time-based filtering is handled by the GitHub API `since` parameter,
// so we only apply maxCount and unreadOnly here.
func applyInitialSyncLimitsFromContext(
	threads []types.NotificationThread,
	syncCtx SyncContext,
) []types.NotificationThread {
	// If no limits to apply, return as-is
	if syncCtx.MaxCount == nil && !syncCtx.UnreadOnly {
		return threads
	}

	filtered := []types.NotificationThread{}
	count := 0

	for _, thread := range threads {
		// Apply unread-only filter if specified
		if syncCtx.UnreadOnly && !thread.Unread {
			continue
		}

		// Apply max count limit if specified
		if syncCtx.MaxCount != nil && count >= *syncCtx.MaxCount {
			break
		}

		filtered = append(filtered, thread)
		count++
	}

	return filtered
}
