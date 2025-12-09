// Copyright (C) Austin Beattie
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

import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { debugLog } from "$lib/utils/debug";
import { getSWHealthStore } from "$lib/stores/swHealthStore";
import type { NotificationPageController } from "$lib/state/types";

interface ServiceWorkerHandlerParams {
	getPageController: () => NotificationPageController | null;
}

/**
 * Set up all service worker event listeners.
 * This should be called early in the app lifecycle (e.g., in +layout.svelte)
 * to ensure messages aren't missed.
 *
 * Handles:
 * - Service worker messages (NEW_NOTIFICATIONS, OPEN_NOTIFICATION, TOKEN_EXPIRED)
 * - Page visibility changes (sync notifications when page becomes visible)
 *
 * @param params - Function to get pageController when available
 * @returns Cleanup function to remove event listeners
 */
export function setupServiceWorkerHandlers(params: ServiceWorkerHandlerParams): () => void {
	if (!browser) {
		return () => {}; // No-op cleanup
	}

	const { getPageController } = params;

	// Get SW health store for heartbeat tracking
	const swHealthStore = getSWHealthStore();

	// Handle service worker messages
	const handleSWMessage = async (event: MessageEvent) => {
		debugLog("[App] Service worker message received:", event.data?.type, event.data);

		if (event.data?.type === "SW_HEARTBEAT") {
			// Service worker is alive and polling - record the heartbeat
			const timestamp = event.data.timestamp || Date.now();
			swHealthStore.recordHeartbeat(timestamp);
		} else if (event.data?.type === "NEW_NOTIFICATIONS") {
			// Service worker detected new notifications
			// Trigger the existing sync handler to refresh the UI
			const pageController = getPageController();
			if (pageController) {
				void pageController.actions.handleSyncNewNotifications();
			}
		} else if (event.data?.type === "OPEN_NOTIFICATION") {
			// Service worker wants to open a specific notification
			// Navigate to the URL - the page's reactive statement will handle opening the detail
			const notificationId = event.data.notificationId;
			debugLog("[App] Received OPEN_NOTIFICATION message:", { notificationId });

			if (notificationId) {
				const notificationIdStr = String(notificationId);
				const targetUrl = `/views/inbox?id=${encodeURIComponent(notificationIdStr)}`;

				// Navigate to the URL - the reactive statement will handle opening the detail
				const urlObj = new URL(targetUrl, window.location.origin);
				const resolvedPath = resolve(urlObj.pathname as any) + urlObj.search;
				debugLog("[App] Navigating to:", resolvedPath);

				// Use goto() to navigate - this should update the URL and trigger reactive statements
				// If we're already on the same route, goto() should still update the query params
				// Use replaceState: true to avoid creating a new history entry
				void goto(resolvedPath, {
					noScroll: true,
					keepFocus: true,
					replaceState: true,
					invalidateAll: false,
				}).catch((error) => {
					console.error("[App] Failed to navigate to notification:", error);
				});
			}
		}
	};

	// Handle page visibility changes to refresh when page wakes up
	// This ensures we load latest notifications when returning to a suspended tab
	const handleVisibilityChange = async () => {
		if (!document.hidden) {
			const pageController = getPageController();

			if (pageController) {
				// Sync notifications to get the latest state
				await pageController.actions.handleSyncNewNotifications();
			}
		}
	};

	// Set up listeners
	const cleanupFunctions: (() => void)[] = [];

	if ("serviceWorker" in navigator) {
		// Service worker message listener
		navigator.serviceWorker.addEventListener("message", handleSWMessage);
		cleanupFunctions.push(() => {
			navigator.serviceWorker.removeEventListener("message", handleSWMessage);
		});

		// Controller change listener
		const handleControllerChange = () => {
			debugLog("[App] Service worker controller changed");
		};
		navigator.serviceWorker.addEventListener("controllerchange", handleControllerChange);
		cleanupFunctions.push(() => {
			navigator.serviceWorker.removeEventListener("controllerchange", handleControllerChange);
		});
	}

	// Visibility change listener
	document.addEventListener("visibilitychange", handleVisibilityChange);
	cleanupFunctions.push(() => {
		document.removeEventListener("visibilitychange", handleVisibilityChange);
	});

	// Return cleanup function
	return () => {
		cleanupFunctions.forEach((cleanup) => cleanup());
	};
}
