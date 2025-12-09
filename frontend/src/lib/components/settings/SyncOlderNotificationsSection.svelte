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
	import { getSyncState, syncOlderNotifications, type SyncState } from "$lib/api/user";
	import { toastStore } from "$lib/stores/toastStore";
	import { toLocalDatetime } from "$lib/utils/time";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";

	type DaySelection = number | "custom";

	let syncState: SyncState | null = null;
	let isLoading = true;
	let isSubmitting = false;
	let error = "";
	let showConfirmDialog = false;

	// Form state
	let selectedDays: DaySelection = 30;
	let customDays: number | null = null;
	let maxNotifications: number | null = null;
	let syncUnreadOnly = false;

	// Before date override state
	let useBeforeDateOverride = false;
	let beforeDateOverride: string = "";

	// Predefined day options
	const dayOptions = [30, 60, 90];

	// Validation limits (must match backend)
	const MAX_SYNC_DAYS = 3650; // 10 years
	const MAX_NOTIFICATION_COUNT = 100000;

	// Show warning for large sync periods
	$: showLargeSyncWarning =
		(selectedDays === "custom" && customDays !== null && customDays > 90) ||
		(typeof selectedDays === "number" && selectedDays > 90);

	// Validation errors
	$: customDaysError =
		selectedDays === "custom" && customDays !== null
			? customDays < 1
				? "Must be at least 1 day"
				: customDays > MAX_SYNC_DAYS
					? `Cannot exceed ${MAX_SYNC_DAYS.toLocaleString()} days`
					: null
			: null;

	$: maxNotificationsError =
		maxNotifications !== null
			? maxNotifications < 1
				? "Must be at least 1"
				: maxNotifications > MAX_NOTIFICATION_COUNT
					? `Cannot exceed ${MAX_NOTIFICATION_COUNT.toLocaleString()}`
					: null
			: null;

	// Check if form has validation errors
	$: hasValidationErrors = customDaysError !== null || maxNotificationsError !== null;

	// Get effective days for submission
	function getEffectiveDays(): number {
		if (selectedDays === "custom") return customDays || 30;
		return selectedDays;
	}

	// Format date for display (with time)
	function formatDate(dateStr: string): string {
		const timestamp = Date.parse(dateStr);
		const date = new Date(timestamp);
		return date.toLocaleDateString(undefined, {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	}

	// Format date for display (short, no time)
	function formatShortDate(dateStr: string): string {
		const timestamp = Date.parse(dateStr);
		const date = new Date(timestamp);
		return date.toLocaleDateString(undefined, {
			year: "numeric",
			month: "short",
			day: "numeric",
		});
	}

	// Calculate start date by subtracting days from end date
	function calculateStartDate(endDateStr: string, days: number): string {
		const timestamp = Date.parse(endDateStr);
		// Subtract days in milliseconds to avoid Date mutation
		const startTimestamp = timestamp - days * 24 * 60 * 60 * 1000;
		return new Date(startTimestamp).toISOString();
	}

	// Convert local datetime-local input value to RFC3339
	function toRFC3339(localDatetime: string): string {
		if (!localDatetime) return "";
		const timestamp = Date.parse(localDatetime);
		const date = new Date(timestamp);
		return date.toISOString();
	}

	onMount(async () => {
		// Initialize beforeDateOverride with the current date
		beforeDateOverride = toLocalDatetime(new Date(Date.now()).toISOString());

		try {
			syncState = await getSyncState();
		} catch (err) {
			console.error("Failed to fetch sync state:", err);
			error = "Failed to load sync state";
		} finally {
			isLoading = false;
		}
	});

	function handleDaysChange(option: DaySelection) {
		selectedDays = option;
		if (option === "custom") {
			customDays = customDays || 30;
		}
	}

	function handleSubmit() {
		error = "";

		const days = getEffectiveDays();

		// Validate before showing dialog
		if (days < 1) {
			error = "Please enter a valid number of days";
			return;
		}
		if (days > MAX_SYNC_DAYS) {
			error = `Days cannot exceed ${MAX_SYNC_DAYS}`;
			return;
		}

		if (maxNotifications !== null) {
			if (maxNotifications < 1) {
				error = "Maximum notifications must be at least 1";
				return;
			}
			if (maxNotifications > MAX_NOTIFICATION_COUNT) {
				error = `Maximum notifications cannot exceed ${MAX_NOTIFICATION_COUNT}`;
				return;
			}
		}

		// Validate before date override if enabled
		if (useBeforeDateOverride && !beforeDateOverride) {
			error = "Please select a starting date or disable the override";
			return;
		}

		// Show confirmation dialog
		showConfirmDialog = true;
	}

	async function handleConfirmedSync() {
		showConfirmDialog = false;
		isSubmitting = true;

		try {
			const days = getEffectiveDays();

			await syncOlderNotifications({
				days,
				maxCount: maxNotifications,
				unreadOnly: syncUnreadOnly,
				beforeDate: useBeforeDateOverride ? toRFC3339(beforeDateOverride) : null,
			});

			toastStore.success(`Syncing ${days} more days of notifications...`);

			// Refresh sync state to show updated oldest timestamp
			syncState = await getSyncState();
		} catch (err) {
			console.error("Failed to sync older notifications:", err);
			error = err instanceof Error ? err.message : "Failed to start sync";
			toastStore.error(error);
		} finally {
			isSubmitting = false;
		}
	}

	function handleCancelSync() {
		showConfirmDialog = false;
	}

	// Allow sync if we have an oldest notification OR if using a before date override
	$: canSync = syncState?.oldestNotificationSyncedAt != null || useBeforeDateOverride;

	// Compute effective days reactively (must reference variables directly for Svelte reactivity)
	$: effectiveDays = selectedDays === "custom" ? customDays || 30 : selectedDays;

	// Build confirm dialog body
	$: confirmDialogBody = (() => {
		const endDate = useBeforeDateOverride
			? formatShortDate(toRFC3339(beforeDateOverride))
			: syncState?.oldestNotificationSyncedAt
				? formatShortDate(syncState.oldestNotificationSyncedAt)
				: "your oldest synced notification";

		const startDate = useBeforeDateOverride
			? formatShortDate(calculateStartDate(toRFC3339(beforeDateOverride), effectiveDays))
			: syncState?.oldestNotificationSyncedAt
				? formatShortDate(calculateStartDate(syncState.oldestNotificationSyncedAt, effectiveDays))
				: `${effectiveDays} days before`;

		const notificationType = syncUnreadOnly ? "unread" : "all";
		const limitClause = maxNotifications ? ` (up to ${maxNotifications.toLocaleString()})` : "";

		return `This will sync ${notificationType} notifications from ${startDate} to ${endDate}${limitClause}. This may take a while depending on how many notifications exist.`;
	})();
</script>

<div class="space-y-4">
	<div>
		<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">Sync</h3>
		<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">Manage sync with GitHub</p>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-8 text-sm text-gray-500 dark:text-gray-400">
			Loading...
		</div>
	{:else if !canSync && !useBeforeDateOverride}
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-gray-700 dark:bg-gray-900/50 dark:text-gray-400"
		>
			Complete initial setup first. Once notifications have been synced, you can fetch older ones
			here.
		</div>
	{:else}
		<!-- Form card -->
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
		>
			{#if error}
				<div
					class="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
					role="alert"
				>
					{error}
				</div>
			{/if}

			<form on:submit|preventDefault={handleSubmit} class="space-y-5">
				<!-- Days selector -->
				<div>
					<div class="mb-3">
						<p class="text-sm font-medium text-gray-700 dark:text-gray-300">
							Sync additional history
						</p>
						<p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
							{#if syncState?.oldestNotificationSyncedAt}
								Add more notifications before your oldest synced (<span
									class="font-medium text-gray-600 dark:text-gray-300"
									>{formatShortDate(syncState.oldestNotificationSyncedAt)}</span
								>).
							{:else}
								Fetch older notifications from GitHub.
							{/if}
						</p>
					</div>
					<div class="grid grid-cols-4 gap-2">
						{#each dayOptions as days (days)}
							<button
								type="button"
								on:click={() => handleDaysChange(days)}
								disabled={isSubmitting}
								class="rounded-lg border px-3 py-2 text-sm font-medium transition-colors cursor-pointer {selectedDays ===
								days
									? 'border-indigo-500 bg-indigo-50 text-indigo-700 dark:border-indigo-400 dark:bg-indigo-500/10 dark:text-indigo-400'
									: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700'}"
							>
								+{days} days
							</button>
						{/each}
						<button
							type="button"
							on:click={() => handleDaysChange("custom")}
							disabled={isSubmitting}
							class="rounded-lg border px-3 py-2 text-sm font-medium transition-colors cursor-pointer {selectedDays ===
							'custom'
								? 'border-indigo-500 bg-indigo-50 text-indigo-700 dark:border-indigo-400 dark:bg-indigo-500/10 dark:text-indigo-400'
								: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700'}"
						>
							Custom
						</button>
					</div>

					{#if selectedDays === "custom"}
						<input
							type="number"
							min="1"
							max={MAX_SYNC_DAYS}
							placeholder="Enter number of days"
							bind:value={customDays}
							disabled={isSubmitting}
							class="mt-2 w-full rounded-lg border px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-1 dark:text-white dark:placeholder-gray-500 {customDaysError
								? 'border-red-300 bg-red-50 focus:border-red-500 focus:ring-red-500 dark:border-red-700 dark:bg-red-950/30 dark:focus:border-red-400 dark:focus:ring-red-400'
								: 'border-gray-200 bg-white focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:focus:border-indigo-400 dark:focus:ring-indigo-400'}"
						/>
						{#if customDaysError}
							<p class="mt-1 text-xs text-red-600 dark:text-red-400">{customDaysError}</p>
						{/if}
					{/if}

					{#if showLargeSyncWarning}
						<p class="mt-2 text-xs text-amber-600 dark:text-amber-500">
							Large sync periods may take a while and include many notifications.
						</p>
					{/if}
				</div>

				<!-- Advanced options -->
				<details class="group">
					<summary
						class="flex cursor-pointer list-none items-center gap-1.5 text-sm font-medium text-gray-600 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
					>
						<svg
							class="h-3.5 w-3.5 transition-transform group-open:rotate-90"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M9 5l7 7-7 7"
							/>
						</svg>
						Advanced options
					</summary>

					<div class="mt-3 rounded-xl bg-white/60 dark:bg-gray-900/80 p-4 space-y-4">
						<!-- Max notifications -->
						<div>
							<label
								for="maxNotifications"
								class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Maximum notifications
							</label>
							<p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 mb-1.5">
								Optional. Enter a maximum number of notifications to sync.
							</p>
							<input
								id="maxNotifications"
								type="number"
								min="1"
								max={MAX_NOTIFICATION_COUNT}
								placeholder="No limit"
								bind:value={maxNotifications}
								disabled={isSubmitting}
								class="w-full rounded-lg border px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-1 dark:text-white dark:placeholder-gray-500 {maxNotificationsError
									? 'border-red-300 bg-red-50 focus:border-red-500 focus:ring-red-500 dark:border-red-700 dark:bg-red-950/30 dark:focus:border-red-400 dark:focus:ring-red-400'
									: 'border-gray-300 bg-white focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-950 dark:focus:border-indigo-400 dark:focus:ring-indigo-400'}"
							/>
							{#if maxNotificationsError}
								<p class="mt-1 text-xs text-red-600 dark:text-red-400">{maxNotificationsError}</p>
							{/if}
						</div>

						<!-- Unread only toggle -->
						<div class="flex items-center justify-between gap-4">
							<div>
								<span
									id="unread-only-label"
									class="text-sm font-medium text-gray-700 dark:text-gray-300"
									>Only unread notifications</span
								>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
									Only sync messages marked unread in GitHub. <span class="font-medium"
										>Not recommended for most users.</span
									>
								</p>
							</div>
							<button
								type="button"
								role="switch"
								aria-checked={syncUnreadOnly}
								aria-labelledby="unread-only-label"
								disabled={isSubmitting}
								on:click={() => (syncUnreadOnly = !syncUnreadOnly)}
								class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 {syncUnreadOnly
									? 'bg-indigo-600'
									: 'bg-gray-200 dark:bg-gray-600'} {isSubmitting
									? 'opacity-50 cursor-not-allowed'
									: ''}"
							>
								<span
									aria-hidden="true"
									class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {syncUnreadOnly
										? 'translate-x-5'
										: 'translate-x-0'}"
								></span>
							</button>
						</div>

						<!-- Custom starting date -->
						<div>
							<div class="flex items-center justify-between mb-2">
								<div>
									<span
										id="override-date-label"
										class="text-sm font-medium text-gray-700 dark:text-gray-300"
										>Custom starting date</span
									>
									<p class="text-xs text-gray-500 dark:text-gray-400">
										Sync from a specific date instead of oldest notification.
									</p>
								</div>
								<button
									type="button"
									role="switch"
									aria-checked={useBeforeDateOverride}
									aria-labelledby="override-date-label"
									disabled={isSubmitting}
									on:click={() => (useBeforeDateOverride = !useBeforeDateOverride)}
									class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 {useBeforeDateOverride
										? 'bg-indigo-600'
										: 'bg-gray-200 dark:bg-gray-600'} {isSubmitting
										? 'opacity-50 cursor-not-allowed'
										: ''}"
								>
									<span
										aria-hidden="true"
										class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {useBeforeDateOverride
											? 'translate-x-5'
											: 'translate-x-0'}"
									></span>
								</button>
							</div>
							{#if useBeforeDateOverride}
								<input
									id="beforeDateOverride"
									type="datetime-local"
									bind:value={beforeDateOverride}
									disabled={isSubmitting}
									class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-950 dark:text-white dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
								/>
							{/if}
						</div>
					</div>
				</details>

				<!-- Submit row with summary and button -->
				<div
					class="flex items-center justify-between gap-4 rounded-xl bg-indigo-50 dark:bg-indigo-950/30 border border-indigo-100 dark:border-indigo-900/50 p-3 mt-1"
				>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						{#if useBeforeDateOverride && beforeDateOverride}
							Sync <span class="font-medium text-indigo-600 dark:text-indigo-400"
								>{syncUnreadOnly ? "unread" : "all"}</span
							>
							notifications from
							<span class="font-medium text-gray-700 dark:text-gray-200"
								>{formatShortDate(
									calculateStartDate(toRFC3339(beforeDateOverride), effectiveDays)
								)}</span
							>
							to
							<span class="font-medium text-gray-700 dark:text-gray-200"
								>{formatShortDate(toRFC3339(beforeDateOverride))}</span
							>{#if maxNotifications}
								<span class="text-gray-500 dark:text-gray-400">
									&nbsp;(up to {maxNotifications.toLocaleString()})</span
								>
							{/if}
						{:else if syncState?.oldestNotificationSyncedAt}
							Sync <span class="font-medium text-gray-700 dark:text-gray-200"
								>{syncUnreadOnly ? "unread" : "all"}</span
							>
							notifications from
							<span class="font-medium text-gray-700 dark:text-gray-200"
								>{formatShortDate(
									calculateStartDate(syncState.oldestNotificationSyncedAt, effectiveDays)
								)}</span
							>
							to
							<span class="font-medium text-gray-700 dark:text-gray-200"
								>{formatShortDate(syncState.oldestNotificationSyncedAt)}</span
							>{#if maxNotifications}
								<span class="text-gray-500 dark:text-gray-400">
									&nbsp;(up to {maxNotifications.toLocaleString()})</span
								>
							{/if}
						{:else}
							<span class="text-gray-500 dark:text-gray-400"
								>Enable a custom starting date to sync</span
							>
						{/if}
					</p>
					<button
						type="submit"
						disabled={isSubmitting || !canSync || hasValidationErrors}
						class="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer flex-shrink-0"
					>
						{#if isSubmitting}
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
							Syncing...
						{:else}
							Start sync
						{/if}
					</button>
				</div>
			</form>
		</div>
	{/if}
</div>

<ConfirmDialog
	open={showConfirmDialog}
	title="Sync older notifications?"
	body={confirmDialogBody}
	confirmLabel="Start sync"
	cancelLabel="Cancel"
	confirming={isSubmitting}
	onConfirm={handleConfirmedSync}
	onCancel={handleCancelSync}
/>
