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

import { writable, derived, get } from "svelte/store";
import type { Readable } from "svelte/store";
import { getGitHubStatus, type GitHubStatus } from "$lib/api/user";

const POLL_INTERVAL_MS = 5 * 60 * 1000;
const DISMISS_STORAGE_KEY = "github-token-expiring-dismissed-for";

/**
 * GitHub Status Store
 * Polls /api/user/github-status so banners can react to token health changes
 * (approaching expiration, 401 responses) observed by the backend RoundTripper.
 */
export function createGithubStatusStore() {
	const status = writable<GitHubStatus | null>(null);
	const dismissedFor = writable<string | null>(null);

	let pollIntervalId: ReturnType<typeof setInterval> | null = null;
	let focusHandler: (() => void) | null = null;

	async function refresh(): Promise<void> {
		try {
			const next = await getGitHubStatus();
			status.set(next);
		} catch (err) {
			console.error("Failed to fetch GitHub status:", err);
		}
	}

	function loadDismissed(): void {
		if (typeof window === "undefined") return;
		try {
			dismissedFor.set(window.localStorage.getItem(DISMISS_STORAGE_KEY));
		} catch {
			dismissedFor.set(null);
		}
	}

	function dismissExpiring(): void {
		const current = get(status);
		if (!current?.tokenExpiresAt) return;
		try {
			window.localStorage.setItem(DISMISS_STORAGE_KEY, current.tokenExpiresAt);
			dismissedFor.set(current.tokenExpiresAt);
		} catch {
			// localStorage may be unavailable (private mode, etc.); swallow.
		}
	}

	/**
	 * Banner the UI should render, or null if no banner applies. An "expiring"
	 * banner that has been dismissed for the current expiresAt resolves to null
	 * until the expiration timestamp changes (e.g. user rotates their PAT).
	 */
	const bannerState = derived(
		[status, dismissedFor],
		([$status, $dismissedFor]): "invalid" | "expiring" | null => {
			if (!$status || !$status.connected) return null;
			if ($status.tokenHealth === "invalid") return "invalid";
			if ($status.tokenHealth === "expiring") {
				if ($status.tokenExpiresAt && $dismissedFor === $status.tokenExpiresAt) {
					return null;
				}
				return "expiring";
			}
			return null;
		}
	);

	async function initialize(): Promise<void> {
		// Idempotent — avoid leaking a second interval/focus handler if called
		// more than once (the store is a singleton across route remounts).
		if (pollIntervalId !== null) return;

		loadDismissed();
		await refresh();

		pollIntervalId = setInterval(() => {
			void refresh();
		}, POLL_INTERVAL_MS);

		if (typeof window !== "undefined") {
			focusHandler = () => {
				void refresh();
			};
			window.addEventListener("focus", focusHandler);
		}
	}

	function destroy(): void {
		if (pollIntervalId !== null) {
			clearInterval(pollIntervalId);
			pollIntervalId = null;
		}
		if (focusHandler && typeof window !== "undefined") {
			window.removeEventListener("focus", focusHandler);
			focusHandler = null;
		}
	}

	return {
		status: status as Readable<GitHubStatus | null>,
		bannerState: bannerState as Readable<"invalid" | "expiring" | null>,
		refresh,
		dismissExpiring,
		initialize,
		destroy,
	};
}

// Singleton instance
let githubStatusStoreInstance: ReturnType<typeof createGithubStatusStore> | null = null;

/**
 * Get the singleton GitHub status store instance
 */
export function getGithubStatusStore(): ReturnType<typeof createGithubStatusStore> {
	if (!githubStatusStoreInstance) {
		githubStatusStoreInstance = createGithubStatusStore();
	}
	return githubStatusStoreInstance;
}
