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
 */

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

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
	if (!API_BASE_URL) {
		return path;
	}
	const base = API_BASE_URL.replace(/\/+$/, "");
	return `${base}${path}`;
}

/**
 * Simple fetch wrapper for API calls.
 */
export async function fetchAPI(
	url: string,
	options: RequestInit = {},
	fetchImpl?: typeof fetch
): Promise<Response> {
	const fetchFn = fetchImpl || fetch;

	try {
		return await fetchFn(buildApiUrl(url), options);
	} catch (error) {
		// Network errors (TypeError: Failed to fetch) indicate the API is unreachable
		if (isNetworkError(error)) {
			throw new ApiUnreachableError();
		}
		throw error;
	}
}

// Export with old name for backwards compatibility during transition
export const fetchWithAuth = fetchAPI;
