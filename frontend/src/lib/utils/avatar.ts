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
 * Resolves a GitHub avatar redirect URL to the direct avatar URL.
 * GitHub redirects from https://github.com/username.png to the actual avatar URL.
 * This function follows the redirect and returns the final URL.
 *
 * @param redirectUrl - The GitHub redirect URL (e.g., https://github.com/username.png)
 * @returns Promise resolving to the direct avatar URL, or null if resolution fails
 */
export async function resolveAvatarRedirect(redirectUrl: string): Promise<string | null> {
	try {
		const response = await fetch(redirectUrl, {
			method: "HEAD",
			redirect: "follow",
			// Don't include credentials to avoid CORS issues
			credentials: "omit",
		});

		if (response.ok) {
			// Return the final URL after redirects
			return response.url;
		}

		return null;
	} catch (error) {
		// If fetch fails (CORS, network error, etc.), return null
		console.warn("Failed to resolve avatar redirect:", error);
		return null;
	}
}

/**
 * Gets an avatar URL, preferring direct URLs when available.
 * Falls back to resolving redirect URLs if needed.
 *
 * @param directUrl - Direct avatar URL (e.g., from GitHub API)
 * @param username - GitHub username (for fallback redirect URL)
 * @returns Promise resolving to the avatar URL, or null if unavailable
 */
export async function getAvatarUrl(
	directUrl: string | null | undefined,
	username: string | null | undefined
): Promise<string | null> {
	// Prefer direct URL if available
	if (directUrl) {
		return directUrl;
	}

	// Fall back to resolving redirect URL
	if (username && username !== "Unknown") {
		const redirectUrl = `https://github.com/${encodeURIComponent(username)}.png`;
		return await resolveAvatarRedirect(redirectUrl);
	}

	return null;
}

/**
 * Helper to compute the fallback redirect URL for a username.
 * @param username - GitHub username
 * @returns Redirect URL or null if username is invalid
 */
export function getRedirectAvatarUrl(username: string | null | undefined): string | null {
	if (!username || username === "Unknown") {
		return null;
	}
	return `https://github.com/${encodeURIComponent(username)}.png`;
}

/**
 * Computes the base avatar URL (direct URL if available, otherwise redirect URL).
 * This is a helper for the common pattern in components.
 *
 * @param directUrl - Direct avatar URL from API
 * @param username - GitHub username for fallback
 * @returns Avatar URL (direct or redirect) or null
 */
export function computeAvatarUrl(
	directUrl: string | null | undefined,
	username: string | null | undefined
): string | null {
	return directUrl || getRedirectAvatarUrl(username);
}

/**
 * Checks if a URL is a redirect URL (github.com/username.png pattern).
 * @param url - URL to check
 * @returns True if it's a redirect URL
 */
export function isRedirectAvatarUrl(url: string | null | undefined): boolean {
	return !!(url && url.includes("github.com/") && url.endsWith(".png"));
}

/**
 * Creates an error handler function for avatar images.
 * This handles retrying redirect resolution when an image fails to load.
 *
 * @param options - Configuration options
 * @returns Error handler function
 */
export function createAvatarErrorHandler(options: {
	redirectUrl: string | null;
	directUrl: string | null | undefined;
	setResolvedUrl: (url: string | null) => void;
	setLoadFailed: (failed: boolean) => void;
	getResolvedUrl: () => string | null;
	getLoadFailed: () => boolean;
}) {
	const { redirectUrl, directUrl, setResolvedUrl, setLoadFailed, getResolvedUrl, getLoadFailed } =
		options;

	return () => {
		// If we haven't resolved the redirect yet, try one more time
		if (redirectUrl && !directUrl && !getResolvedUrl() && !getLoadFailed()) {
			resolveAvatarRedirect(redirectUrl).then((resolved) => {
				if (resolved) {
					setResolvedUrl(resolved);
					setLoadFailed(false);
				} else {
					setLoadFailed(true);
				}
			});
		} else {
			setLoadFailed(true);
		}
	};
}
