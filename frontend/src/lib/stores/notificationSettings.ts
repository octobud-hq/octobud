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

import { writable, type Writable } from "svelte/store";
import { browser } from "$app/environment";

const NOTIFICATIONS_ENABLED_KEY = "octobud:notifications:enabled";
const FAVICON_BADGE_ENABLED_KEY = "octobud:notifications:faviconBadge";

/**
 * Notification Settings Store
 * Manages notification preferences persisted to localStorage
 */
export function createNotificationSettingsStore() {
	// Load from localStorage, default to true (enabled)
	const storedEnabled =
		typeof window !== "undefined"
			? localStorage.getItem(NOTIFICATIONS_ENABLED_KEY) !== "false" // Default to true
			: true;

	const storedFaviconBadgeEnabled =
		typeof window !== "undefined"
			? localStorage.getItem(FAVICON_BADGE_ENABLED_KEY) !== "false" // Default to true
			: true;

	const notificationsEnabled = writable<boolean>(storedEnabled);
	const faviconBadgeEnabled = writable<boolean>(storedFaviconBadgeEnabled);

	// Persist to localStorage and notify service worker
	if (browser) {
		notificationsEnabled.subscribe((value) => {
			localStorage.setItem(NOTIFICATIONS_ENABLED_KEY, value.toString());

			// Notify service worker of setting change
			if ("serviceWorker" in navigator && navigator.serviceWorker.controller) {
				navigator.serviceWorker.controller.postMessage({
					type: "NOTIFICATION_SETTING_CHANGED",
					enabled: value,
				});
			}
		});

		faviconBadgeEnabled.subscribe((value) => {
			localStorage.setItem(FAVICON_BADGE_ENABLED_KEY, value.toString());
		});
	}

	return {
		notificationsEnabled: notificationsEnabled as Writable<boolean>,
		faviconBadgeEnabled: faviconBadgeEnabled as Writable<boolean>,
		setEnabled: (enabled: boolean) => {
			notificationsEnabled.set(enabled);
		},
		setFaviconBadgeEnabled: (enabled: boolean) => {
			faviconBadgeEnabled.set(enabled);
		},
		toggle: () => {
			notificationsEnabled.update((enabled) => !enabled);
		},
		toggleFaviconBadge: () => {
			faviconBadgeEnabled.update((enabled) => !enabled);
		},
	};
}

// Singleton instance
let notificationSettingsStoreInstance: ReturnType<typeof createNotificationSettingsStore> | null =
	null;

export function getNotificationSettingsStore() {
	if (!notificationSettingsStoreInstance) {
		notificationSettingsStoreInstance = createNotificationSettingsStore();
	}
	return notificationSettingsStoreInstance;
}

// Helper to check if notifications are enabled (for service worker)
export function isNotificationEnabled(): boolean {
	if (!browser) return true; // Default to enabled on server
	const stored = localStorage.getItem(NOTIFICATIONS_ENABLED_KEY);
	return stored !== "false"; // Default to true
}

// Helper to check if favicon badge is enabled
export function isFaviconBadgeEnabled(): boolean {
	if (!browser) return true; // Default to enabled on server
	const stored = localStorage.getItem(FAVICON_BADGE_ENABLED_KEY);
	return stored !== "false"; // Default to true
}
