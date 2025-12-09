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
	"database/sql"
	"time"
)

// SyncState represents the current sync state
type SyncState struct {
	LastSuccessfulPoll         sql.NullTime
	LatestNotificationAt       sql.NullTime
	UpdatedAt                  time.Time
	InitialSyncCompletedAt     sql.NullTime
	OldestNotificationSyncedAt sql.NullTime
}
