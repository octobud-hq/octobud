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

import { writable, derived, get } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import { checkForUpdates, getUpdateSettings, updateUpdateSettings, type UpdateSettings, type UpdateCheckResponse } from "$lib/api/user";

/**
 * Update Store
 * Manages update checking state, polling, and settings
 */
export function createUpdateStore() {
	const updateInfo = writable<UpdateCheckResponse | null>(null);
	const isLoading = writable<boolean>(false);
	const isChecking = writable<boolean>(false);
	const error = writable<string | null>(null);
	const settings = writable<UpdateSettings | null>(null);
	const lastCheckedAt = writable<Date | null>(null);

	// Polling interval ID
	let pollIntervalId: ReturnType<typeof setInterval> | null = null;
	let pollTimeoutId: ReturnType<typeof setTimeout> | null = null;

	// Derived store for whether an update is available
	const updateAvailable = derived(updateInfo, ($updateInfo) => {
		return $updateInfo?.updateAvailable ?? false;
	});

	/**
	 * Load update settings from the API
	 */
	async function loadSettings(): Promise<void> {
		try {
			const currentSettings = await getUpdateSettings();
			settings.set(currentSettings);
			if (currentSettings.lastCheckedAt) {
				lastCheckedAt.set(new Date(currentSettings.lastCheckedAt));
			}
		} catch (err) {
			console.error("Failed to load update settings:", err);
			error.set(err instanceof Error ? err.message : "Failed to load update settings");
		}
	}

	/**
	 * Update settings and save to API
	 */
	async function saveSettings(updates: Partial<UpdateSettings>): Promise<void> {
		const currentSettings = get(settings);
		if (!currentSettings) {
			await loadSettings();
			return;
		}

		try {
			const updated = await updateUpdateSettings(updates);
			settings.set(updated);
			if (updated.lastCheckedAt) {
				lastCheckedAt.set(new Date(updated.lastCheckedAt));
			}

			// Restart polling if settings changed in a way that affects polling
			if (updates.enabled !== undefined || updates.checkFrequency !== undefined) {
				stopPolling();
				startPolling();
			}
		} catch (err) {
			console.error("Failed to save update settings:", err);
			error.set(err instanceof Error ? err.message : "Failed to save update settings");
			throw err;
		}
	}

	/**
	 * Manually check for updates (called by user or polling)
	 */
	async function checkForUpdate(force: boolean = false): Promise<UpdateCheckResponse | null> {
		const currentSettings = get(settings);
		
		// If settings aren't loaded yet, load them first
		if (!currentSettings) {
			await loadSettings();
			const loadedSettings = get(settings);
			if (!loadedSettings) {
				return null;
			}
		}

		// Skip if updates are disabled and not forced
		if (!force && !get(settings)?.enabled) {
			return null;
		}

		isChecking.set(true);
		error.set(null);

		try {
			const response = await checkForUpdates();
			updateInfo.set(response);
			
			// Update last checked time from settings
			const updatedSettings = await getUpdateSettings();
			settings.set(updatedSettings);
			if (updatedSettings.lastCheckedAt) {
				lastCheckedAt.set(new Date(updatedSettings.lastCheckedAt));
			}

			return response;
		} catch (err) {
			console.error("Failed to check for updates:", err);
			error.set(err instanceof Error ? err.message : "Failed to check for updates");
			return null;
		} finally {
			isChecking.set(false);
		}
	}

	/**
	 * Dismiss update banner for a period of time
	 */
	async function dismissUpdate(untilHours: number = 24): Promise<void> {
		const until = new Date();
		until.setHours(until.getHours() + untilHours);
		
		await saveSettings({
			dismissedUntil: until.toISOString(),
		});
	}

	/**
	 * Clear dismissed state (show update banner again)
	 */
	async function clearDismissed(): Promise<void> {
		await saveSettings({
			dismissedUntil: undefined,
		});
	}

	/**
	 * Start polling for updates based on settings
	 */
	function startPolling(): void {
		stopPolling(); // Clear any existing polling

		const currentSettings = get(settings);
		if (!currentSettings?.enabled) {
			return;
		}

		// Don't poll if frequency is "never" or "on_startup"
		if (currentSettings.checkFrequency === "never" || currentSettings.checkFrequency === "on_startup") {
			return;
		}

		// Calculate polling interval based on frequency
		let intervalMs: number;
		switch (currentSettings.checkFrequency) {
			case "daily":
				intervalMs = 24 * 60 * 60 * 1000; // 24 hours
				break;
			case "weekly":
				intervalMs = 7 * 24 * 60 * 60 * 1000; // 7 days
				break;
			default:
				intervalMs = 60 * 60 * 1000; // Default: 1 hour
		}

		// Check immediately on start
		void checkForUpdate();

		// Then check at the configured interval
		pollIntervalId = setInterval(() => {
			void checkForUpdate();
		}, intervalMs);
	}

	/**
	 * Stop polling for updates
	 */
	function stopPolling(): void {
		if (pollIntervalId !== null) {
			clearInterval(pollIntervalId);
			pollIntervalId = null;
		}
		if (pollTimeoutId !== null) {
			clearTimeout(pollTimeoutId);
			pollTimeoutId = null;
		}
	}

	/**
	 * Initialize the store - load settings and start polling
	 */
	async function initialize(): Promise<void> {
		await loadSettings();
		startPolling();
	}

	/**
	 * Cleanup - stop polling
	 */
	function destroy(): void {
		stopPolling();
	}

	return {
		// Stores
		updateInfo: updateInfo as Readable<UpdateCheckResponse | null>,
		isLoading: isLoading as Readable<boolean>,
		isChecking: isChecking as Readable<boolean>,
		error: error as Readable<string | null>,
		settings: settings as Readable<UpdateSettings | null>,
		lastCheckedAt: lastCheckedAt as Readable<Date | null>,

		// Derived stores
		updateAvailable: updateAvailable as Readable<boolean>,

		// Actions
		loadSettings,
		saveSettings,
		checkForUpdate,
		dismissUpdate,
		clearDismissed,
		startPolling,
		stopPolling,
		initialize,
		destroy,
	};
}

// Singleton instance
let updateStoreInstance: ReturnType<typeof createUpdateStore> | null = null;

/**
 * Get the singleton update store instance
 */
export function getUpdateStore(): ReturnType<typeof createUpdateStore> {
	if (!updateStoreInstance) {
		updateStoreInstance = createUpdateStore();
		
		// Expose store globally in browser for testing/debugging
		if (typeof window !== "undefined") {
			(window as any).__octobudUpdateStore = updateStoreInstance;
		}
	}
	return updateStoreInstance;
}

