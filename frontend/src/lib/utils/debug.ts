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

/**
 * Debug logging utility
 *
 * Only logs in development mode (import.meta.env.DEV) or when explicitly enabled
 * via localStorage by setting 'octobud:debug' to 'true'.
 *
 * To enable debug logging in production, run in browser console:
 *   localStorage.setItem('octobud:debug', 'true')
 *
 * To disable debug logging:
 *   localStorage.removeItem('octobud:debug')
 *
 * The debug flag is automatically synced with the service worker.
 */

const DEBUG_KEY = "octobud:debug";

function isDebugEnabled(): boolean {
	if (typeof window === "undefined") {
		return false;
	}

	// Always enabled in development mode
	if (import.meta.env.DEV) {
		return true;
	}

	// Check localStorage for explicit debug flag
	try {
		return localStorage.getItem(DEBUG_KEY) === "true";
	} catch {
		return false;
	}
}

/**
 * Debug logger - only logs if debug mode is enabled
 */
export function debugLog(...args: unknown[]): void {
	if (isDebugEnabled()) {
		console.log(...args);
	}
}

/**
 * Enable debug logging programmatically
 */
export function enableDebugLogging(): void {
	if (typeof window !== "undefined") {
		try {
			localStorage.setItem(DEBUG_KEY, "true");
		} catch {
			// Ignore localStorage errors
		}
	}
}

/**
 * Disable debug logging programmatically
 */
export function disableDebugLogging(): void {
	if (typeof window !== "undefined") {
		try {
			localStorage.removeItem(DEBUG_KEY);
		} catch {
			// Ignore localStorage errors
		}
	}
}
