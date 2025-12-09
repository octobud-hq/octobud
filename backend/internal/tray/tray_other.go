//go:build !darwin

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

// Package tray provides system tray support. On non-macOS platforms,
// this is a no-op implementation that runs the app directly.
package tray

import (
	"time"

	"go.uber.org/zap"
)

// Tray is a no-op on non-darwin platforms.
type Tray struct{}

// RunWithApp on non-darwin platforms just calls onAppReady directly.
// There is no system tray, so the app runs normally.
// baseURL is ignored on non-darwin platforms.
// logger is ignored on non-darwin platforms.
func RunWithApp(baseURL string, logger *zap.Logger, onAppReady func(t *Tray), onAppExit func()) {
	t := &Tray{}
	if onAppReady != nil {
		onAppReady(t)
	}
	// On non-macOS, we don't block here - the app should manage its own lifecycle
}

// Quit is a no-op on non-darwin platforms.
func (t *Tray) Quit() {}

// SetOnQuit is a no-op on non-darwin platforms.
func (t *Tray) SetOnQuit(onQuit func()) {}

// SetUnreadCount is a no-op on non-darwin platforms.
func (t *Tray) SetUnreadCount(count int) {}

// SetLastSync is a no-op on non-darwin platforms.
func (t *Tray) SetLastSync(syncTime time.Time) {}

// SetSyncing is a no-op on non-darwin platforms.
func (t *Tray) SetSyncing(syncing bool) {}

// SetNavigationBroadcaster is a no-op on non-darwin platforms.
func (t *Tray) SetNavigationBroadcaster(broadcaster interface{}) {}
