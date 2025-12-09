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

import { writable, derived, get, type Readable } from "svelte/store";
import { browser } from "$app/environment";
import { ensureSWPolling } from "$lib/utils/serviceWorkerRegistration";
import { debugLog } from "$lib/utils/debug";

// Service worker polls every 10 seconds
// We consider it stale if we haven't heard from it in 2 minutes (12+ missed polls)
const STALE_THRESHOLD_MS = 2 * 60 * 1000; // 2 minutes

// How often to check if SW is stale
const HEALTH_CHECK_INTERVAL_MS = 30 * 1000; // 30 seconds

export interface SWHealthState {
	lastHeartbeat: number | null;
	isStale: boolean;
	recoveryAttempted: boolean;
	showBanner: boolean;
}

/**
 * Service Worker Health Store
 * Tracks SW heartbeats and detects when the SW appears stale/dead
 */
export function createSWHealthStore() {
	const lastHeartbeat = writable<number | null>(null);
	const recoveryAttempted = writable<boolean>(false);
	const showBanner = writable<boolean>(false);
	const bannerDismissed = writable<boolean>(false);

	// Derived: is the SW considered stale?
	const isStale: Readable<boolean> = derived(lastHeartbeat, ($lastHeartbeat) => {
		if ($lastHeartbeat === null) {
			// Never received a heartbeat - not stale yet, SW might be starting up
			return false;
		}
		return Date.now() - $lastHeartbeat > STALE_THRESHOLD_MS;
	});

	let healthCheckInterval: ReturnType<typeof setInterval> | null = null;

	// Record a heartbeat from the service worker
	function recordHeartbeat(timestamp: number) {
		lastHeartbeat.set(timestamp);
		// If we receive a heartbeat, SW is healthy - hide banner and reset recovery state
		showBanner.set(false);
		recoveryAttempted.set(false);
		bannerDismissed.set(false);
		debugLog("[SW Health] Heartbeat received:", new Date(timestamp).toISOString());
	}

	// Check SW health and attempt recovery if stale
	async function checkHealth() {
		if (!browser || !("serviceWorker" in navigator)) {
			return;
		}

		const $lastHeartbeat = get(lastHeartbeat);
		const $recoveryAttempted = get(recoveryAttempted);
		const $bannerDismissed = get(bannerDismissed);

		// Skip check if user dismissed the banner
		if ($bannerDismissed) {
			return;
		}

		// If no heartbeat yet, wait for one before checking
		if ($lastHeartbeat === null) {
			return;
		}

		const timeSinceHeartbeat = Date.now() - $lastHeartbeat;

		if (timeSinceHeartbeat > STALE_THRESHOLD_MS) {
			debugLog("[SW Health] SW appears stale, last heartbeat:", timeSinceHeartbeat, "ms ago");

			if (!$recoveryAttempted) {
				// Try to recover by ensuring SW is polling
				debugLog("[SW Health] Attempting recovery via ensureSWPolling...");
				recoveryAttempted.set(true);

				const success = await ensureSWPolling();
				if (success) {
					debugLog("[SW Health] Recovery attempt sent, waiting for heartbeat...");
					// Wait a bit and check if we got a heartbeat
					setTimeout(() => {
						const newHeartbeat = get(lastHeartbeat);
						if (newHeartbeat === null || Date.now() - newHeartbeat > STALE_THRESHOLD_MS) {
							// Still stale after recovery attempt - show banner
							debugLog("[SW Health] Still stale after recovery, showing banner");
							showBanner.set(true);
						}
					}, 15000); // Wait 15s for a heartbeat (SW polls every 10s)
				} else {
					// Recovery failed immediately - show banner
					debugLog("[SW Health] Recovery failed, showing banner");
					showBanner.set(true);
				}
			}
		}
	}

	// Start periodic health checks
	function startHealthChecks() {
		if (!browser) return;

		// Initial check after a delay to let SW start up
		setTimeout(checkHealth, 30000);

		// Periodic checks
		healthCheckInterval = setInterval(checkHealth, HEALTH_CHECK_INTERVAL_MS);
	}

	// Stop health checks
	function stopHealthChecks() {
		if (healthCheckInterval) {
			clearInterval(healthCheckInterval);
			healthCheckInterval = null;
		}
	}

	// Dismiss the banner (user action)
	function dismissBanner() {
		showBanner.set(false);
		bannerDismissed.set(true);
	}

	// Reset dismissed state (e.g., after page refresh or successful recovery)
	function resetDismissed() {
		bannerDismissed.set(false);
	}

	// Force show banner (for testing) - bypasses recovery logic
	function forceShowBanner() {
		bannerDismissed.set(false);
		showBanner.set(true);
	}

	return {
		// Stores
		lastHeartbeat: lastHeartbeat as Readable<number | null>,
		isStale,
		showBanner: showBanner as Readable<boolean>,

		// Actions
		recordHeartbeat,
		checkHealth,
		startHealthChecks,
		stopHealthChecks,
		dismissBanner,
		resetDismissed,
		forceShowBanner,
	};
}

// Singleton instance
let swHealthStoreInstance: ReturnType<typeof createSWHealthStore> | null = null;

export function getSWHealthStore() {
	if (!swHealthStoreInstance) {
		swHealthStoreInstance = createSWHealthStore();
	}
	return swHealthStoreInstance;
}

// Debug helpers - expose to window in dev mode
if (browser && import.meta.env.DEV) {
	(window as any).swHealth = {
		// Force show the stale SW banner (bypasses recovery)
		showBanner: () => getSWHealthStore().forceShowBanner(),
		// Hide the banner
		hideBanner: () => getSWHealthStore().dismissBanner(),
		// Get current state
		getState: () => {
			const store = getSWHealthStore();
			return {
				lastHeartbeat: get(store.lastHeartbeat),
				isStale: get(store.isStale),
				showBanner: get(store.showBanner),
			};
		},
	};
	debugLog(
		"[SW Health] Dev mode: Use swHealth.showBanner() / swHealth.hideBanner() / swHealth.getState() in console"
	);
}
