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

import type { NotificationPageController } from "$lib/state/types";

interface SyncRefreshState {
	previousLatestNotificationAt: string | null;
	isHandlingSyncRefresh: boolean;
}

/**
 * Create state for sync refresh tracking.
 */
export function createSyncRefreshState(): SyncRefreshState {
	return {
		previousLatestNotificationAt: null,
		isHandlingSyncRefresh: false,
	};
}

/**
 * Handle sync refresh logic.
 * Call this from a reactive statement in your component.
 */
export function handleSyncRefresh(
	state: SyncRefreshState,
	currentLatestNotificationAt: string | null,
	pageController: NotificationPageController
): void {
	// Check if we have new notifications (latestNotificationAt changed)
	if (
		currentLatestNotificationAt &&
		currentLatestNotificationAt !== state.previousLatestNotificationAt &&
		!state.isHandlingSyncRefresh
	) {
		state.previousLatestNotificationAt = currentLatestNotificationAt;
		state.isHandlingSyncRefresh = true;

		// Delegate to page controller method
		void pageController.actions.handleSyncNewNotifications().finally(() => {
			state.isHandlingSyncRefresh = false;
		});
	} else if (!currentLatestNotificationAt) {
		// Reset tracking if latestNotificationAt is cleared
		state.previousLatestNotificationAt = null;
	}
}
