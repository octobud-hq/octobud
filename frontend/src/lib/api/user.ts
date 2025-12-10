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

export interface MuteStatus {
	mutedUntil: string | null;
	isMuted: boolean;
}

export interface SetMuteRequest {
	duration: "30m" | "1h" | "rest_of_day";
}

/**
 * User API functions.
 */

import { fetchAPI } from "./fetch";

export interface SyncSettings {
	initialSyncDays?: number | null;
	initialSyncMaxCount?: number | null;
	initialSyncUnreadOnly: boolean;
	setupCompleted: boolean;
}

export interface UserResponse {
	githubUserId?: string;
	githubUsername?: string;
	syncSettings?: SyncSettings | null;
}

export async function getCurrentUser(fetchImpl?: typeof fetch): Promise<UserResponse> {
	const response = await fetchAPI(
		"/api/user/me",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get user" }));
		throw new Error(error.error || "Failed to get user");
	}

	return response.json();
}

export async function getSyncSettings(fetchImpl?: typeof fetch): Promise<SyncSettings> {
	const response = await fetchAPI(
		"/api/user/sync-settings",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get sync settings" }));
		throw new Error(error.error || "Failed to get sync settings");
	}

	return response.json();
}

export interface UpdateSyncSettingsRequest {
	initialSyncDays?: number | null;
	initialSyncMaxCount?: number | null;
	initialSyncUnreadOnly: boolean;
	setupCompleted: boolean;
}

export async function updateSyncSettings(
	settings: UpdateSyncSettingsRequest,
	fetchImpl?: typeof fetch
): Promise<SyncSettings> {
	const response = await fetchAPI(
		"/api/user/sync-settings",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(settings),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to update sync settings" }));
		throw new Error(error.error || "Failed to update sync settings");
	}

	return response.json();
}

export interface SyncState {
	oldestNotificationSyncedAt?: string | null;
	initialSyncCompletedAt?: string | null;
}

export async function getSyncState(fetchImpl?: typeof fetch): Promise<SyncState> {
	const response = await fetchAPI(
		"/api/user/sync-state",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get sync state" }));
		throw new Error(error.error || "Failed to get sync state");
	}

	return response.json();
}

export interface SyncOlderRequest {
	days: number;
	maxCount?: number | null;
	unreadOnly?: boolean;
	beforeDate?: string | null; // RFC3339 format date to override the "before" cutoff
}

export async function syncOlderNotifications(
	params: SyncOlderRequest,
	fetchImpl?: typeof fetch
): Promise<void> {
	const response = await fetchAPI(
		"/api/user/sync-older",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(params),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to start sync of older notifications" }));
		throw new Error(error.error || "Failed to start sync of older notifications");
	}

	// Response is 202 Accepted with no body
}

// GitHub Token Management

export interface GitHubStatus {
	connected: boolean;
	source: "none" | "stored" | "oauth";
	maskedToken?: string;
	githubUsername?: string;
}

export async function getGitHubStatus(fetchImpl?: typeof fetch): Promise<GitHubStatus> {
	const response = await fetchAPI(
		"/api/user/github-status",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get GitHub status" }));
		throw new Error(error.error || "Failed to get GitHub status");
	}

	return response.json();
}

export interface SetGitHubTokenResponse {
	githubUsername: string;
}

export async function setGitHubToken(
	token: string,
	fetchImpl?: typeof fetch
): Promise<SetGitHubTokenResponse> {
	const response = await fetchAPI(
		"/api/user/github-token",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ token }),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to set GitHub token" }));
		throw new Error(error.error || "Failed to set GitHub token");
	}

	return response.json();
}

export async function clearGitHubToken(fetchImpl?: typeof fetch): Promise<GitHubStatus> {
	const response = await fetchAPI(
		"/api/user/github-token",
		{
			method: "DELETE",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to clear GitHub token" }));
		throw new Error(error.error || "Failed to clear GitHub token");
	}

	return response.json();
}

// Storage Management

export interface StorageStats {
	totalCount: number;
	archivedCount: number;
	starredCount: number;
	snoozedCount: number;
	unreadCount: number;
	taggedCount: number;
	eligibleForCleanup: number;
}

export interface RetentionSettings {
	enabled: boolean;
	retentionDays: number;
	protectStarred: boolean;
	protectTagged: boolean;
	lastCleanupAt?: string;
}

export interface CleanupResult {
	notificationsDeleted: number;
	pullRequestsDeleted: number;
}

export async function getStorageStats(fetchImpl?: typeof fetch): Promise<StorageStats> {
	const response = await fetchAPI(
		"/api/user/storage-stats",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get storage stats" }));
		throw new Error(error.error || "Failed to get storage stats");
	}

	return response.json();
}

export async function getRetentionSettings(fetchImpl?: typeof fetch): Promise<RetentionSettings> {
	const response = await fetchAPI(
		"/api/user/retention-settings",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to get retention settings" }));
		throw new Error(error.error || "Failed to get retention settings");
	}

	return response.json();
}

export interface UpdateRetentionSettingsRequest {
	enabled: boolean;
	retentionDays: number;
	protectStarred: boolean;
	protectTagged: boolean;
}

export async function updateRetentionSettings(
	settings: UpdateRetentionSettingsRequest,
	fetchImpl?: typeof fetch
): Promise<RetentionSettings> {
	const response = await fetchAPI(
		"/api/user/retention-settings",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(settings),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to update retention settings" }));
		throw new Error(error.error || "Failed to update retention settings");
	}

	return response.json();
}

export interface RunCleanupRequest {
	retentionDays: number;
	protectStarred: boolean;
	protectTagged: boolean;
}

export async function runManualCleanup(
	params: RunCleanupRequest,
	fetchImpl?: typeof fetch
): Promise<CleanupResult> {
	const response = await fetchAPI(
		"/api/user/cleanup",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(params),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to run cleanup" }));
		throw new Error(error.error || "Failed to run cleanup");
	}

	return response.json();
}

// Mute Management

export async function getMuteStatus(fetchImpl?: typeof fetch): Promise<MuteStatus> {
	const response = await fetchAPI(
		"/api/user/mute-status",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get mute status" }));
		throw new Error(error.error || "Failed to get mute status");
	}

	return response.json();
}

export async function setMute(
	duration: "30m" | "1h" | "rest_of_day",
	fetchImpl?: typeof fetch
): Promise<MuteStatus> {
	const response = await fetchAPI(
		"/api/user/mute",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ duration }),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to set mute" }));
		throw new Error(error.error || "Failed to set mute");
	}

	return response.json();
}

export async function clearMute(fetchImpl?: typeof fetch): Promise<MuteStatus> {
	const response = await fetchAPI(
		"/api/user/mute",
		{
			method: "DELETE",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to clear mute" }));
		throw new Error(error.error || "Failed to clear mute");
	}

	return response.json();
}

// Delete all GitHub data
export async function deleteAllGitHubData(fetchImpl?: typeof fetch): Promise<void> {
	const response = await fetchAPI(
		"/api/user/github-data",
		{
			method: "DELETE",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to delete GitHub data" }));
		throw new Error(error.error || "Failed to delete GitHub data");
	}
}

// Update Management

export interface UpdateSettings {
	enabled: boolean;
	checkFrequency: "on_startup" | "daily" | "weekly" | "never";
	includePrereleases: boolean;
	lastCheckedAt?: string;
	dismissedUntil?: string;
}

export interface UpdateSettingsRequest {
	enabled?: boolean;
	checkFrequency?: "on_startup" | "daily" | "weekly" | "never";
	includePrereleases?: boolean;
	dismissedUntil?: string;
}

export interface UpdateCheckResponse {
	updateAvailable: boolean;
	currentVersion?: string;
	latestVersion?: string;
	releaseName?: string;
	releaseNotes?: string;
	publishedAt?: string;
	downloadUrl?: string;
	assetName?: string;
	assetSize?: number;
	isPrerelease?: boolean;
}

export async function getUpdateSettings(fetchImpl?: typeof fetch): Promise<UpdateSettings> {
	const response = await fetchAPI(
		"/api/user/update-settings",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get update settings" }));
		throw new Error(error.error || "Failed to get update settings");
	}

	return response.json();
}

export async function updateUpdateSettings(
	settings: UpdateSettingsRequest,
	fetchImpl?: typeof fetch
): Promise<UpdateSettings> {
	const response = await fetchAPI(
		"/api/user/update-settings",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(settings),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to update update settings" }));
		throw new Error(error.error || "Failed to update update settings");
	}

	return response.json();
}

export async function checkForUpdates(
	force: boolean = false,
	fetchImpl?: typeof fetch
): Promise<UpdateCheckResponse> {
	const url = force ? "/api/user/update-check?force=true" : "/api/user/update-check";
	const response = await fetchAPI(
		url,
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to check for updates" }));
		throw new Error(error.error || "Failed to check for updates");
	}

	return response.json();
}

export interface VersionResponse {
	version: string;
}

export async function getVersion(fetchImpl?: typeof fetch): Promise<VersionResponse> {
	const response = await fetchAPI(
		"/api/user/version",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get version" }));
		throw new Error(error.error || "Failed to get version");
	}

	return response.json();
}

export async function getInstalledVersion(fetchImpl?: typeof fetch): Promise<VersionResponse> {
	const response = await fetchAPI(
		"/api/user/installed-version",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get installed version" }));
		throw new Error(error.error || "Failed to get installed version");
	}

	return response.json();
}

export interface RestartResponse {
	status: string;
	message?: string;
}

export async function restartApp(fetchImpl?: typeof fetch): Promise<void> {
	const response = await fetchAPI(
		"/api/user/restart",
		{
			method: "POST",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to restart app" }));
		throw new Error(error.error || "Failed to restart app");
	}

	// Response is not needed, but we can return it if needed
	await response.json();
}
