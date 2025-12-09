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

import { browser } from "$app/environment";
import { debugLog } from "$lib/utils/debug";

const SW_URL = "/sw.js";
const SW_UPDATE_INTERVAL = 60 * 60 * 1000; // Check for updates every hour

export interface ServiceWorkerRegistrationState {
	registration: ServiceWorkerRegistration | null;
	isSupported: boolean;
	isActive: boolean;
	error: string | null;
}

let registrationState: ServiceWorkerRegistrationState = {
	registration: null,
	isSupported: false,
	isActive: false,
	error: null,
};

const stateListeners = new Set<(state: ServiceWorkerRegistrationState) => void>();

export function subscribeToSWState(listener: (state: ServiceWorkerRegistrationState) => void) {
	stateListeners.add(listener);
	// Immediately call with current state
	listener(registrationState);

	return () => {
		stateListeners.delete(listener);
	};
}

function updateState(updates: Partial<ServiceWorkerRegistrationState>) {
	registrationState = { ...registrationState, ...updates };
	stateListeners.forEach((listener) => listener(registrationState));
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
	if (!browser) {
		return null;
	}

	if (!("serviceWorker" in navigator)) {
		debugLog("[SW Registration] Service workers not supported");
		updateState({ isSupported: false, error: "Service workers not supported" });
		return null;
	}

	updateState({ isSupported: true });

	try {
		const registration = await navigator.serviceWorker.register(SW_URL, {
			scope: "/",
		});

		debugLog("[SW Registration] Service Worker registered:", registration.scope);
		updateState({ registration, isActive: !!navigator.serviceWorker.controller });

		// Sync settings with service worker
		syncDebugFlagWithSW();
		syncNotificationSettingWithSW();

		// Handle updates
		registration.addEventListener("updatefound", () => {
			const newWorker = registration.installing;
			if (!newWorker) return;

			newWorker.addEventListener("statechange", () => {
				if (newWorker.state === "installed" && navigator.serviceWorker.controller) {
					// New service worker available
					debugLog("[SW Registration] New service worker available");
					// Could show a toast or prompt to reload
					updateState({ isActive: false });
				} else if (newWorker.state === "activated") {
					debugLog("[SW Registration] New service worker activated");
					updateState({ isActive: true });
					// Sync settings after activation
					syncDebugFlagWithSW();
					syncNotificationSettingWithSW();
				}
			});
		});

		// Listen for controller change (when SW takes control)
		navigator.serviceWorker.addEventListener("controllerchange", () => {
			debugLog("[SW Registration] Service worker controller changed");
			updateState({ isActive: !!navigator.serviceWorker.controller });
			// Sync settings when controller changes
			syncDebugFlagWithSW();
			syncNotificationSettingWithSW();
		});

		// Check for updates periodically
		// In development, check more frequently (every 30 seconds)
		const updateIntervalMs = import.meta.env.DEV ? 30000 : SW_UPDATE_INTERVAL;
		const updateInterval = setInterval(() => {
			registration.update().catch((err) => {
				console.error("[SW Registration] Failed to check for updates:", err);
			});
		}, updateIntervalMs);

		// In development, also check for updates on page load
		if (import.meta.env.DEV) {
			registration.update().catch((err) => {
				console.error("[SW Registration] Failed to check for updates on load:", err);
			});
		}

		// Clean up interval on page unload
		if (browser) {
			window.addEventListener("beforeunload", () => {
				clearInterval(updateInterval);
			});
		}

		return registration;
	} catch (error) {
		console.error("[SW Registration] Service Worker registration failed:", error);
		updateState({ error: error instanceof Error ? error.message : "Registration failed" });
		return null;
	}
}

export async function unregisterServiceWorker(): Promise<boolean> {
	if (!browser || !("serviceWorker" in navigator)) {
		return false;
	}

	try {
		const registration = await navigator.serviceWorker.getRegistration();
		if (registration) {
			const success = await registration.unregister();
			if (success) {
				debugLog("[SW Registration] Service Worker unregistered");
				updateState({ registration: null, isActive: false });
			}
			return success;
		}
		return false;
	} catch (error) {
		console.error("[SW Registration] Failed to unregister:", error);
		return false;
	}
}

// Request notification permission
export async function requestNotificationPermission(): Promise<NotificationPermission> {
	if (!browser || !("Notification" in window)) {
		debugLog("[SW Registration] Notifications not supported in this environment");
		return "denied";
	}

	const currentPermission = Notification.permission;
	debugLog("[SW Registration] Current notification permission:", currentPermission);

	if (currentPermission === "granted") {
		// Also notify service worker of permission status
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission: "granted" });
		return "granted";
	}

	if (currentPermission === "denied") {
		debugLog("[SW Registration] Notification permission is denied");
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission: "denied" });
		return "denied";
	}

	try {
		debugLog("[SW Registration] Requesting notification permission...");
		const permission = await Notification.requestPermission();
		debugLog("[SW Registration] Notification permission result:", permission);

		// Notify service worker of permission status
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission });

		return permission;
	} catch (error) {
		console.error("[SW Registration] Failed to request notification permission:", error);
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission: "denied" });
		return "denied";
	}
}

// Get current notification permission
export function getNotificationPermission(): NotificationPermission {
	if (!browser || !("Notification" in window)) {
		return "denied";
	}
	return Notification.permission;
}

// Sync debug flag with service worker
function syncDebugFlagWithSW() {
	if (!browser || !("serviceWorker" in navigator) || !navigator.serviceWorker.controller) {
		return;
	}

	// Check if debug is enabled (check localStorage or dev mode)
	const DEBUG_KEY = "octobud:debug";
	let debugEnabled = false;

	if (import.meta.env.DEV) {
		debugEnabled = true;
	} else {
		try {
			debugEnabled = localStorage.getItem(DEBUG_KEY) === "true";
		} catch {
			// Ignore localStorage errors
		}
	}

	// Send debug flag to service worker
	sendMessageToSW({
		type: "DEBUG_ENABLED",
		enabled: debugEnabled,
	});
}

// Sync notification enabled setting with service worker
function syncNotificationSettingWithSW() {
	if (!browser || !("serviceWorker" in navigator) || !navigator.serviceWorker.controller) {
		return;
	}

	// Read the notification enabled setting from localStorage
	const NOTIFICATIONS_ENABLED_KEY = "octobud:notifications:enabled";
	let notificationsEnabled = true; // Default to enabled

	try {
		const stored = localStorage.getItem(NOTIFICATIONS_ENABLED_KEY);
		// If explicitly set to "false", disable; otherwise default to true
		notificationsEnabled = stored !== "false";
	} catch {
		// Ignore localStorage errors, use default
	}

	debugLog("[SW Registration] Syncing notification setting to SW:", notificationsEnabled);

	// Send notification setting to service worker
	sendMessageToSW({
		type: "NOTIFICATION_SETTING_CHANGED",
		enabled: notificationsEnabled,
	});
}

// Send message to service worker
export async function sendMessageToSW(message: any, transfer?: Transferable[]): Promise<void> {
	if (!browser || !("serviceWorker" in navigator) || !navigator.serviceWorker.controller) {
		return;
	}

	try {
		if (transfer) {
			navigator.serviceWorker.controller.postMessage(message, transfer);
		} else {
			navigator.serviceWorker.controller.postMessage(message);
		}
	} catch (error) {
		console.error("[SW Registration] Failed to send message to SW:", error);
	}
}

// Send notification setting to service worker
export async function sendNotificationSettingToSW(enabled: boolean): Promise<void> {
	await sendMessageToSW({
		type: "NOTIFICATION_SETTING_CHANGED",
		enabled,
	});
}

// Force service worker to poll immediately
export async function forceSWPoll(): Promise<void> {
	await sendMessageToSW({ type: "FORCE_POLL" });
}

// Wait for service worker controller to be available
// Returns true if controller is available, false if timed out
export async function waitForSWController(timeoutMs: number = 5000): Promise<boolean> {
	if (!browser || !("serviceWorker" in navigator)) {
		return false;
	}

	// Already have a controller
	if (navigator.serviceWorker.controller) {
		return true;
	}

	// Wait for controller to become available
	// Use both event listener AND polling as a fallback for race conditions
	return new Promise((resolve) => {
		let resolved = false;

		const cleanup = () => {
			if (resolved) return;
			resolved = true;
			clearTimeout(timeout);
			clearInterval(pollInterval);
			navigator.serviceWorker.removeEventListener("controllerchange", handler);
		};

		const timeout = setTimeout(() => {
			cleanup();
			resolve(false);
		}, timeoutMs);

		const handler = () => {
			cleanup();
			resolve(true);
		};

		// Poll every 100ms as a fallback in case we miss the controllerchange event
		// This handles race conditions where the event fires between checks
		const pollInterval = setInterval(() => {
			if (navigator.serviceWorker.controller) {
				cleanup();
				resolve(true);
			}
		}, 100);

		navigator.serviceWorker.addEventListener("controllerchange", handler);

		// Also check immediately in case controller appeared between our check and adding listener
		if (navigator.serviceWorker.controller) {
			cleanup();
			resolve(true);
		}
	});
}

// Ensure service worker polling is active
// This is safe to call multiple times - the SW will ignore if already polling
// Registers SW if needed, waits for controller, then sends message to ensure polling
export async function ensureSWPolling(): Promise<boolean> {
	if (!browser || !("serviceWorker" in navigator)) {
		return false;
	}

	// If no controller yet, we may need to register the SW first
	if (!navigator.serviceWorker.controller) {
		// Check if SW is registered
		const existingRegistration = await navigator.serviceWorker.getRegistration();
		if (!existingRegistration) {
			// SW not registered - register it now
			debugLog("[SW Registration] No SW registered, registering now for ensureSWPolling");
			await registerServiceWorker();
		} else {
			// Registration exists - check if there's a waiting worker that needs to be activated
			if (existingRegistration.waiting) {
				debugLog("[SW Registration] Found waiting worker, sending SKIP_WAITING");
				existingRegistration.waiting.postMessage({ type: "SKIP_WAITING" });
			} else if (existingRegistration.installing) {
				// Worker is still installing, wait for it
				debugLog("[SW Registration] Worker is still installing, waiting...");
			}
		}

		// Wait for controller (with timeout)
		const hasController = await waitForSWController(5000);
		if (!hasController) {
			debugLog("[SW Registration] No controller available after waiting, cannot ensure polling");
			return false;
		}
	}

	// Send message to ensure polling is started
	// SW will ignore if already polling
	try {
		navigator.serviceWorker.controller!.postMessage({ type: "FORCE_POLL" });
		debugLog("[SW Registration] Sent FORCE_POLL to ensure SW is polling");
		return true;
	} catch (error) {
		console.error("[SW Registration] Failed to send polling message:", error);
		return false;
	}
}

// Reset the service worker's poll state (clears lastMaxEffectiveSortDate)
// Call this after login or initial setup so all notifications are treated as new
export async function resetSWPollState(): Promise<void> {
	await sendMessageToSW({ type: "RESET_POLL_STATE" });
}

// Send test notification request to service worker
export async function sendTestNotificationToSW(): Promise<void> {
	await sendMessageToSW({ type: "TEST_NOTIFICATION" });
}

// Get current registration state
export function getSWState(): ServiceWorkerRegistrationState {
	return registrationState;
}

// Force service worker to update immediately
export async function forceServiceWorkerUpdate(): Promise<void> {
	if (!browser || !("serviceWorker" in navigator)) {
		return;
	}

	try {
		const registration = await navigator.serviceWorker.getRegistration();
		if (registration) {
			debugLog("[SW Registration] Forcing service worker update...");
			await registration.update();
			debugLog("[SW Registration] Service worker update check completed");

			// If there's a waiting service worker, skip waiting and reload
			if (registration.waiting) {
				debugLog("[SW Registration] New service worker waiting, activating...");
				registration.waiting.postMessage({ type: "SKIP_WAITING" });
				// The service worker will activate and we'll get a controllerchange event
			}
		}
	} catch (error) {
		console.error("[SW Registration] Failed to force update:", error);
	}
}

// Expose force update function to window for easy console access in dev
if (browser && import.meta.env.DEV) {
	(window as any).forceSWUpdate = forceServiceWorkerUpdate;
	debugLog(
		"[SW Registration] Dev mode: Call forceSWUpdate() in console to update service worker"
	);
}
