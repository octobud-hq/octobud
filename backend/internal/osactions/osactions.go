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

// Package osactions provides platform-specific OS actions (app restart, browser tab activation, etc.).
// On macOS, this uses AppleScript. On other platforms, it provides no-op implementations.
package osactions

import "context"

// Service provides platform-specific OS actions.
// On macOS, this uses AppleScript. On other platforms, it provides no-op implementations.
type Service interface {
	// RestartApp quits the current application and launches the installed app.
	// If baseURL is provided, after restart it will attempt to find and refresh
	// an existing browser tab with that URL instead of opening a new one.
	// Returns an error if the restart fails.
	RestartApp(ctx context.Context, baseURL string) error

	// ActivateBrowserTab attempts to find and activate a browser tab with the given URL.
	// browser should be "safari" or "chrome".
	// If navigateTo is provided and a tab is found, it will navigate that tab to navigateTo
	// (useful for forcing a refresh by navigating to the baseURL).
	// Returns true if a tab was found and activated, false otherwise.
	// Returns an error if the operation fails.
	ActivateBrowserTab(
		ctx context.Context,
		browser string,
		url string,
		navigateTo string,
	) (bool, error)

	// GetInstalledAppVersion returns the version of the installed application bundle.
	// On macOS, this reads from the app's Info.plist. On other platforms, this may
	// return an error if the version cannot be determined from the installed bundle.
	GetInstalledAppVersion(ctx context.Context) (string, error)
}
