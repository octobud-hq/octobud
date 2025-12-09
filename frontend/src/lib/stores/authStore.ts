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
 * Auth store for GitHub-based identity.
 */

import { writable, derived, get } from "svelte/store";
import { getCurrentUser, type SyncSettings } from "$lib/api/user";

// User info from GitHub
interface GitHubIdentity {
	userId: string;
	username: string;
}

// Internal stores
const githubIdentityStore = writable<GitHubIdentity | null>(null);
const syncSettingsStore = writable<SyncSettings | null>(null);

// Derived stores
export const isGitHubConnected = derived(
	githubIdentityStore,
	($identity) => $identity !== null && $identity.userId !== ""
);
export const githubUsername = derived(
	githubIdentityStore,
	($identity) => $identity?.username ?? null
);
export const githubUserId = derived(githubIdentityStore, ($identity) => $identity?.userId ?? null);
export const syncSettings = derived(syncSettingsStore, ($settings) => $settings);

// For backwards compatibility with code that checks isAuthenticated
// Now means "has GitHub connected"
export const isAuthenticated = isGitHubConnected;
export const currentUser = githubUsername;

// Helper to check if sync settings are configured
export function hasSyncSettingsConfigured(): boolean {
	const settings = get(syncSettingsStore);
	return settings !== null && settings.setupCompleted;
}

// Helper to check if GitHub is connected
export function hasGitHubConnected(): boolean {
	const identity = get(githubIdentityStore);
	return identity !== null && identity.userId !== "";
}

// Fetch user info including GitHub identity and sync settings
export async function fetchUserInfo(): Promise<void> {
	const user = await getCurrentUser();

	// Update GitHub identity
	if (user.githubUserId || user.githubUsername) {
		githubIdentityStore.set({
			userId: user.githubUserId || "",
			username: user.githubUsername || "",
		});
	} else {
		githubIdentityStore.set(null);
	}

	// Update sync settings
	if (user.syncSettings) {
		syncSettingsStore.set(user.syncSettings);
	} else {
		syncSettingsStore.set(null);
	}
}

// Clear local state (e.g., when GitHub connection is cleared)
export function clearAuthState(): void {
	githubIdentityStore.set(null);
	syncSettingsStore.set(null);
}
