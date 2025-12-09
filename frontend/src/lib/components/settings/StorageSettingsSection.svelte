<!-- Copyright (C) 2025 Austin Beattie

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. -->

<script lang="ts">
	import { onMount } from "svelte";
	import {
		getStorageStats,
		getRetentionSettings,
		updateRetentionSettings,
		runManualCleanup,
		deleteAllGitHubData,
		type StorageStats,
		type RetentionSettings,
	} from "$lib/api/user";
	import { toastStore } from "$lib/stores/toastStore";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";
	import TextConfirmDialog from "$lib/components/dialogs/TextConfirmDialog.svelte";

	let stats: StorageStats | null = null;
	let settings: RetentionSettings | null = null;
	let isLoading = true;
	let isSaving = false;
	let isCleaningUp = false;
	let isDeletingGitHubData = false;
	let error = "";
	let showCleanupDialog = false;
	let showDeleteGitHubDataDialog = false;

	// Editable form state
	let enabled = false;
	let retentionDays = 90;
	let protectStarred = true;
	let protectTagged = true;

	// Retention day options
	const retentionOptions = [
		{ value: 1, label: "1 day" },
		{ value: 30, label: "30 days" },
		{ value: 60, label: "60 days" },
		{ value: 90, label: "90 days" },
		{ value: 180, label: "180 days" },
		{ value: 365, label: "1 year" },
	];

	// Calculate eligible count based on current settings
	$: eligibleCount = stats?.eligibleForCleanup ?? 0;

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		isLoading = true;
		error = "";

		try {
			const [statsResult, settingsResult] = await Promise.all([
				getStorageStats(),
				getRetentionSettings(),
			]);

			stats = statsResult;
			settings = settingsResult;

			// Initialize form state from settings
			enabled = settings.enabled;
			retentionDays = settings.retentionDays || 90;
			protectStarred = settings.protectStarred;
			protectTagged = settings.protectTagged;
		} catch (err) {
			console.error("Failed to load storage data:", err);
			error = err instanceof Error ? err.message : "Failed to load storage data";
		} finally {
			isLoading = false;
		}
	}

	// Generic save function - all settings auto-save on change
	async function saveSettings(
		updates: Partial<{
			enabled: boolean;
			retentionDays: number;
			protectStarred: boolean;
			protectTagged: boolean;
		}>,
		successMessage: string
	) {
		if (isSaving) return;

		isSaving = true;
		error = "";

		// Store previous values for rollback
		const prev = { enabled, retentionDays, protectStarred, protectTagged };

		try {
			const updated = await updateRetentionSettings({
				enabled: updates.enabled ?? enabled,
				retentionDays: updates.retentionDays ?? retentionDays,
				protectStarred: updates.protectStarred ?? protectStarred,
				protectTagged: updates.protectTagged ?? protectTagged,
			});

			settings = updated;
			toastStore.success(successMessage);

			// Reload stats to update eligible count
			stats = await getStorageStats();
		} catch (err) {
			console.error("Failed to save settings:", err);
			// Revert on error
			enabled = prev.enabled;
			retentionDays = prev.retentionDays;
			protectStarred = prev.protectStarred;
			protectTagged = prev.protectTagged;
			error = err instanceof Error ? err.message : "Failed to save settings";
			toastStore.error(error);
		} finally {
			isSaving = false;
		}
	}

	function handleToggleEnabled() {
		enabled = !enabled;
		saveSettings({ enabled }, enabled ? "Auto-cleanup enabled" : "Auto-cleanup disabled");
	}

	function handleRetentionDaysChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		retentionDays = parseInt(target.value, 10);
		const label =
			retentionOptions.find((o) => o.value === retentionDays)?.label ?? `${retentionDays} days`;
		saveSettings({ retentionDays }, `Retention set to ${label}`);
	}

	function handleToggleProtectStarred() {
		protectStarred = !protectStarred;
		saveSettings(
			{ protectStarred },
			protectStarred ? "Starred notifications protected" : "Starred notifications not protected"
		);
	}

	function handleToggleProtectTagged() {
		protectTagged = !protectTagged;
		saveSettings(
			{ protectTagged },
			protectTagged ? "Tagged notifications protected" : "Tagged notifications not protected"
		);
	}

	function handleCleanupClick() {
		showCleanupDialog = true;
	}

	async function handleConfirmCleanup() {
		showCleanupDialog = false;
		isCleaningUp = true;
		error = "";

		try {
			const result = await runManualCleanup({
				retentionDays,
				protectStarred,
				protectTagged,
			});

			const totalDeleted = result.notificationsDeleted + result.pullRequestsDeleted;
			if (totalDeleted > 0) {
				toastStore.success(
					`Cleaned up ${result.notificationsDeleted.toLocaleString()} notifications` +
						(result.pullRequestsDeleted > 0
							? ` and ${result.pullRequestsDeleted.toLocaleString()} orphaned records`
							: "")
				);
			} else {
				toastStore.success("No notifications were eligible for cleanup");
			}

			// Reload stats
			stats = await getStorageStats();
		} catch (err) {
			console.error("Failed to run cleanup:", err);
			error = err instanceof Error ? err.message : "Failed to run cleanup";
			toastStore.error(error);
		} finally {
			isCleaningUp = false;
		}
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return "Never";
		const date = new Date(dateStr);
		return date.toLocaleDateString(undefined, {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	}

	function formatNumber(n: number): string {
		return n.toLocaleString();
	}

	function handleDeleteGitHubDataClick() {
		showDeleteGitHubDataDialog = true;
	}

	async function handleConfirmDeleteGitHubData() {
		showDeleteGitHubDataDialog = false;
		isDeletingGitHubData = true;
		error = "";

		try {
			await deleteAllGitHubData();
			toastStore.success("All GitHub data deleted successfully");
			// Reload stats
			stats = await getStorageStats();
		} catch (err) {
			console.error("Failed to delete GitHub data:", err);
			error = err instanceof Error ? err.message : "Failed to delete GitHub data";
			toastStore.error(error);
		} finally {
			isDeletingGitHubData = false;
		}
	}
</script>

<div class="space-y-4">
	<div>
		<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">Storage</h3>
		<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
			Manage notification storage and automatic cleanup
		</p>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-8 text-sm text-gray-500 dark:text-gray-400">
			Loading...
		</div>
	{:else if error && !stats}
		<div
			class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
			role="alert"
		>
			{error}
		</div>
	{:else}
		<!-- Stats Card -->
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
		>
			<h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
				Notification Statistics
			</h4>

			<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
				<div
					class="bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-100 dark:border-gray-700"
				>
					<div class="text-2xl font-semibold text-gray-900 dark:text-gray-100">
						{formatNumber(stats?.totalCount ?? 0)}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">Total</div>
				</div>

				<div
					class="bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-100 dark:border-gray-700"
				>
					<div class="text-2xl font-semibold text-gray-900 dark:text-gray-100">
						{formatNumber(stats?.archivedCount ?? 0)}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">Archived</div>
				</div>

				<div
					class="bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-100 dark:border-gray-700"
				>
					<div class="text-2xl font-semibold text-indigo-500 dark:text-indigo-300">
						{formatNumber(stats?.unreadCount ?? 0)}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">Unread</div>
				</div>

				<div
					class="bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-100 dark:border-gray-700"
				>
					<div class="text-2xl font-semibold text-indigo-600 dark:text-indigo-400">
						{formatNumber(stats?.starredCount ?? 0)}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">Starred</div>
				</div>

				<div
					class="bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-100 dark:border-gray-700"
				>
					<div class="text-2xl font-semibold text-indigo-600 dark:text-indigo-400">
						{formatNumber(stats?.taggedCount ?? 0)}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">Tagged</div>
				</div>

				<div
					class="bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-100 dark:border-gray-700"
				>
					<div class="text-2xl font-semibold text-orange-600 dark:text-orange-400">
						{formatNumber(stats?.snoozedCount ?? 0)}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">Snoozed</div>
				</div>
			</div>
		</div>

		<!-- Auto-Cleanup Settings Card -->
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
		>
			<div class="flex items-center justify-between mb-4">
				<div>
					<h4 class="text-sm font-medium text-gray-700 dark:text-gray-300">Automatic Cleanup</h4>
					<p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
						Automatically delete old archived notifications
					</p>
				</div>
				<button
					type="button"
					role="switch"
					aria-checked={enabled}
					aria-label="Toggle automatic cleanup"
					on:click={handleToggleEnabled}
					class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 {enabled
						? 'bg-indigo-600'
						: 'bg-gray-200 dark:bg-gray-600'}"
				>
					<span
						aria-hidden="true"
						class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {enabled
							? 'translate-x-5'
							: 'translate-x-0'}"
					></span>
				</button>
			</div>

			<div class="space-y-4">
				<!-- Retention Period -->
				<div>
					<label
						for="retentionDays"
						class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5"
					>
						Delete notifications older than
					</label>
					<select
						id="retentionDays"
						value={retentionDays}
						on:change={handleRetentionDaysChange}
						class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
					>
						{#each retentionOptions as option (option.value)}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</div>

				<!-- Protection Options -->
				<div class="space-y-3">
					<label class="flex items-center gap-3 cursor-pointer">
						<input
							type="checkbox"
							checked={protectStarred}
							on:change={handleToggleProtectStarred}
							class="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 cursor-pointer"
						/>
						<span class="text-sm text-gray-700 dark:text-gray-300">
							Always keep starred notifications
						</span>
					</label>

					<label class="flex items-center gap-3 cursor-pointer">
						<input
							type="checkbox"
							checked={protectTagged}
							on:change={handleToggleProtectTagged}
							class="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 cursor-pointer"
						/>
						<span class="text-sm text-gray-700 dark:text-gray-300">
							Always keep tagged notifications
						</span>
					</label>
				</div>

				<!-- Info Box -->
				<div
					class="rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-100 dark:border-blue-900/50 p-3"
				>
					<p class="text-xs text-blue-700 dark:text-blue-300">
						Only <span class="font-medium">archived</span> and
						<span class="font-medium">muted</span> notifications are deleted. Inbox and snoozed notifications
						are always kept.
					</p>
				</div>

				<!-- Last cleanup info -->
				{#if settings?.lastCleanupAt}
					<p class="text-xs text-gray-500 dark:text-gray-400">
						Last automatic cleanup: {formatDate(settings.lastCleanupAt)}
					</p>
				{/if}
			</div>
		</div>

		<!-- Manual Cleanup Card -->
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
		>
			<h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Manual Cleanup</h4>
			<p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
				Clean up old notifications now using your current settings.
			</p>

			<div
				class="flex items-center justify-between gap-4 rounded-xl bg-amber-50 dark:bg-amber-950/30 border border-amber-100 dark:border-amber-900/50 p-3"
			>
				<div>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						<span class="font-medium text-amber-700 dark:text-amber-300">
							{formatNumber(eligibleCount)}
						</span>
						{eligibleCount === 1 ? "notification" : "notifications"} eligible for cleanup
					</p>
				</div>
				<button
					type="button"
					on:click={handleCleanupClick}
					disabled={isCleaningUp || eligibleCount === 0}
					class="inline-flex items-center gap-2 rounded-full bg-amber-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-amber-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer flex-shrink-0"
				>
					{#if isCleaningUp}
						<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						Cleaning up...
					{:else}
						Clean up now
					{/if}
				</button>
			</div>
		</div>

		<!-- Divider -->
		<div class="border-t border-gray-200 dark:border-gray-800 my-8"></div>

		<!-- Danger Zone Section -->
		<div>
			<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">Danger Zone</h3>
			<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
				Irreversible and destructive actions
			</p>
		</div>

		<!-- Delete All GitHub Data Card -->
		<div
			class="rounded-lg border border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-950/30 p-4"
		>
			<h4 class="text-sm font-medium text-red-700 dark:text-red-300 mb-2">
				Delete All GitHub Data
			</h4>
			<p class="text-xs text-red-600 dark:text-red-400 mb-4">
				Permanently delete all notifications, pull requests, repositories, and sync state. Your
				views, tags, and rules will be preserved. Notification triage actions (read/unread,
				archived, starred, etc.) will not be saved. You'll need to set up sync again to continue
				using the app.
			</p>
			<button
				type="button"
				on:click={handleDeleteGitHubDataClick}
				disabled={isDeletingGitHubData}
				class="inline-flex items-center gap-2 rounded-full bg-red-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
			>
				{#if isDeletingGitHubData}
					<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
					Deleting...
				{:else}
					Delete all GitHub data
				{/if}
			</button>
		</div>
	{/if}
</div>

<ConfirmDialog
	open={showCleanupDialog}
	title="Clean up notifications?"
	body="This will permanently delete {formatNumber(
		eligibleCount
	)} archived notification{eligibleCount === 1
		? ''
		: 's'} older than {retentionDays} days. This action cannot be undone."
	confirmLabel="Delete permanently"
	cancelLabel="Cancel"
	confirming={isCleaningUp}
	confirmTone="danger"
	onConfirm={handleConfirmCleanup}
	onCancel={() => (showCleanupDialog = false)}
/>

<TextConfirmDialog
	open={showDeleteGitHubDataDialog}
	title="Delete all GitHub data?"
	body="This will permanently delete all notifications, pull requests, repositories, and sync state. Your views, tags, and rules will be preserved. Notification triage actions (read/unread, archived, starred, etc.) will not be saved. You'll need to set up sync again to continue using the app. This action cannot be undone."
	confirmLabel="Delete all data"
	cancelLabel="Cancel"
	confirming={isDeletingGitHubData}
	confirmTone="danger"
	requiredText="Delete data"
	inputLabel="Type to confirm:"
	onConfirm={handleConfirmDeleteGitHubData}
	onCancel={() => (showDeleteGitHubDataDialog = false)}
/>
