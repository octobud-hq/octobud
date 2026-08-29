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
 * API fetch utilities.
 *
 * Adapted from frontend/src/lib/api/fetch.ts. The exported surface is identical
 * so the vendored api/ modules import it unchanged (see src/lib/VENDORED.md).
 * The one behavioural difference: the desktop app is served from the same origin
 * as the backend and bakes its base URL in at build time, whereas the panel is a
 * chrome-extension:// / moz-extension:// origin talking to localhost, and the
 * port is only known at runtime. So `buildApiUrl` reads the resolved base URL
 * from ext/backendUrl instead of `import.meta.env.VITE_API_BASE_URL`.
 */

import { getBackendUrl } from "$lib/ext/backendUrl";

/**
 * Custom error class for when the API is unreachable (network errors).
 * This is different from API errors (which have HTTP status codes).
 */
export class ApiUnreachableError extends Error {
	constructor(message: string = "Unable to reach the API server") {
		super(message);
		this.name = "ApiUnreachableError";
	}
}

/**
 * Thrown for API responses with a non-2xx status. Carries the HTTP status and
 * the backend's machine-readable error code (when present) so callers can
 * branch on the reason without string-matching the human-readable message.
 */
export class ApiError extends Error {
	readonly status: number;
	readonly code: string | null;

	constructor(message: string, status: number, code: string | null = null) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
	}
}

/**
 * Backend error codes. Keep in sync with backend/internal/api/helpers/render.go.
 */
export const ApiErrorCode = {
	NotConnected: "not_connected",
} as const;
export type ApiErrorCodeValue = (typeof ApiErrorCode)[keyof typeof ApiErrorCode];

/**
 * Parses a non-OK Response into an ApiError, extracting `error` and `code`
 * from the JSON body when available. Does not throw — call sites decide.
 */
export async function apiErrorFromResponse(
	response: Response,
	fallbackMessage: string
): Promise<ApiError> {
	let message = fallbackMessage;
	let code: string | null = null;
	try {
		const body: { error?: string; code?: string } = await response.json();
		if (body.error) message = body.error;
		if (body.code) code = body.code;
	} catch {
		// Non-JSON body — keep fallback message.
	}
	return new ApiError(message, response.status, code);
}

/**
 * Check if an error is a network error (API unreachable).
 * This includes TypeError from fetch failures and other network-related errors.
 */
export function isNetworkError(error: unknown): boolean {
	if (error instanceof ApiUnreachableError) {
		return true;
	}
	if (error instanceof TypeError) {
		// TypeError with "Failed to fetch" indicates network failure
		return error.message.includes("Failed to fetch") || error.message.includes("NetworkError");
	}
	// Check for other network-related error messages
	if (error instanceof Error) {
		const msg = error.message.toLowerCase();
		return (
			msg.includes("network") ||
			msg.includes("connection") ||
			msg.includes("econnrefused") ||
			msg.includes("unreachable") ||
			msg.includes("proxy error") ||
			msg.includes("bad gateway") ||
			msg.includes("service unavailable")
		);
	}
	return false;
}

/**
 * Check if an error message indicates the API is unreachable.
 * This catches proxy errors that return 500/502/503 when the backend is down.
 */
export function isProxyConnectionError(errorMessage: string): boolean {
	const msg = errorMessage.toLowerCase();
	return (
		msg.includes("econnrefused") ||
		msg.includes("proxy error") ||
		msg.includes("bad gateway") ||
		msg.includes("service unavailable") ||
		msg.includes("connection refused") ||
		msg.includes("connect econnrefused") ||
		msg.includes("unable to connect") ||
		msg.includes("socket hang up") ||
		msg.includes("etimedout") ||
		msg.includes("ehostunreach")
	);
}

export function buildApiUrl(path: string): string {
	return `${getBackendUrl()}${path}`;
}

/**
 * A connection to a port nothing is listening on does not always fail fast — it
 * can hang until the OS gives up, which is minutes. The desktop app is served
 * from the backend's own origin, so it never sees this; the panel talks to
 * localhost over the network stack and does. Without a ceiling the panel sits in
 * a loading state indefinitely instead of saying Octobud isn't running.
 *
 * Generous enough that a slow local query is never cut short.
 */
const REQUEST_TIMEOUT_MS = 10_000;

/**
 * Simple fetch wrapper for API calls.
 */
export async function fetchAPI(
	url: string,
	options: RequestInit = {},
	fetchImpl?: typeof fetch
): Promise<Response> {
	const fetchFn = fetchImpl || fetch;
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

	// Respect a caller-supplied signal as well as the timeout.
	options.signal?.addEventListener("abort", () => controller.abort(), { once: true });

	try {
		return await fetchFn(buildApiUrl(url), { ...options, signal: controller.signal });
	} catch (error) {
		// Network errors (TypeError: Failed to fetch) indicate the API is unreachable
		if (isNetworkError(error) || (error instanceof DOMException && error.name === "AbortError")) {
			throw new ApiUnreachableError();
		}
		throw error;
	} finally {
		clearTimeout(timer);
	}
}

// Export with old name for backwards compatibility during transition
export const fetchWithAuth = fetchAPI;
