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

// Package syncstate provides the sync state service.
package syncstate

import (
	"context"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// SyncStateService is the interface for the sync state service.
type SyncStateService interface {
	GetSyncState(ctx context.Context, userID string) (models.SyncState, error)
	UpsertSyncState(
		ctx context.Context,
		userID string,
		lastSuccessfulPoll *time.Time,
		latestNotificationAt *time.Time,
	) (models.SyncState, error)
	UpsertSyncStateWithInitialSync(
		ctx context.Context,
		userID string,
		lastSuccessfulPoll *time.Time,
		latestNotificationAt *time.Time,
		initialSyncCompletedAt *time.Time,
		oldestNotificationSyncedAt *time.Time,
	) (models.SyncState, error)
}

// Service provides business logic for sync state operations
type Service struct {
	queries db.Store
}

// NewSyncStateService constructs a Service backed by the provided queries
func NewSyncStateService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}
