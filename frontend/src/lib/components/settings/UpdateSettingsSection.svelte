<!-- Copyright (C) 2025 Austin Beattie

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. -->

<script lang="ts">
	import { onMount } from "svelte";
	import { browser } from "$app/environment";
	import { getUpdateStore } from "$lib/stores/updateStore";
	import { toastStore } from "$lib/stores/toastStore";

	const updateStore = getUpdateStore();

	const { settings, isChecking, updateInfo, lastCheckedAt, error } = updateStore;

	let isLoading = true;

	// Form state
	let enabled = false;
	let checkFrequency: "on_startup" | "daily" | "weekly" | "never" = "daily";
	let includePrereleases = false;

	const frequencyOptions = [
		{ value: "on_startup" as const, label: "On startup only" },
		{ value: "daily" as const, label: "Daily" },
		{ value: "weekly" as const, label: "Weekly" },
		{ value: "never" as const, label: "Never" },
	];

	onMount(async () => {
		await updateStore.loadSettings();
		const currentSettings = $settings;
		if (currentSettings) {
			enabled = currentSettings.enabled;
			checkFrequency = currentSettings.checkFrequency;
			includePrereleases = currentSettings.includePrereleases;
		}
		isLoading = false;
	});

	async function handleEnabledChange(value: boolean) {
		enabled = value;
		try {
			await updateStore.saveSettings({ enabled: value });
			toastStore.success(value ? "Update checking enabled" : "Update checking disabled");
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : "Failed to update settings");
			// Revert on error
			enabled = !value;
		}
	}

	async function handleFrequencyChange(value: "on_startup" | "daily" | "weekly" | "never") {
		checkFrequency = value;
		try {
			await updateStore.saveSettings({ checkFrequency: value });
			toastStore.success("Update check frequency updated");
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : "Failed to update settings");
		}
	}

	async function handlePrereleasesChange(value: boolean) {
		includePrereleases = value;
		try {
			await updateStore.saveSettings({ includePrereleases: value });
			toastStore.success(value ? "Pre-releases included" : "Pre-releases excluded");
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : "Failed to update settings");
			includePrereleases = !value;
		}
	}

	async function handleManualCheck() {
		try {
			await updateStore.checkForUpdate(true); // Force check
			const info = $updateInfo;
			if (info?.updateAvailable) {
				toastStore.success(`Update available: ${info.latestVersion}`);
			} else {
				toastStore.success("You're running the latest version");
			}
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : "Failed to check for updates");
		}
	}

	function formatLastChecked(date: Date | null): string {
		if (!date) return "Never";
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		
		if (diffMins < 1) return "Just now";
		if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? "" : "s"} ago`;
		
		const diffHours = Math.floor(diffMins / 60);
		if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? "" : "s"} ago`;
		
		const diffDays = Math.floor(diffHours / 24);
		return `${diffDays} day${diffDays === 1 ? "" : "s"} ago`;
	}

	let isDownloading = false;

	async function handleDownload() {
		const info = $updateInfo;
		if (!info?.downloadUrl) {
			toastStore.error("Download URL not available");
			return;
		}

		isDownloading = true;
		try {
			// Create a temporary anchor element to trigger download
			if (browser) {
				const link = document.createElement("a");
				link.href = info.downloadUrl;
				link.download = info.assetName || "Octobud-update.pkg";
				link.target = "_blank";
				document.body.appendChild(link);
				link.click();
				document.body.removeChild(link);
				
				toastStore.success("Download started");
			}
		} catch (err) {
			console.error("Failed to download update:", err);
			toastStore.error("Failed to start download");
		} finally {
			isDownloading = false;
		}
	}
</script>

{#if isLoading}
	<div class="flex items-center justify-center py-8">
		<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-indigo-600"></div>
	</div>
{:else}
	<div class="space-y-6">
		<!-- Enable/Disable Auto-Updates -->
		<div class="flex items-center justify-between">
			<div class="flex-1">
				<label
					for="update-enabled"
					class="block text-sm font-medium text-gray-900 dark:text-gray-100"
				>
					Enable automatic update checks
				</label>
				<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
					Automatically check for new versions of Octobud
				</p>
			</div>
			<button
				type="button"
				id="update-enabled"
				role="switch"
				aria-checked={enabled}
				aria-label="Enable automatic update checks"
				on:click={() => handleEnabledChange(!enabled)}
				class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 dark:focus:ring-offset-gray-950 {enabled
					? 'bg-indigo-600'
					: 'bg-gray-200 dark:bg-gray-700'}"
			>
				<span
					class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {enabled
						? 'translate-x-5'
						: 'translate-x-0'}"
					aria-hidden="true"
				></span>
			</button>
		</div>

		{#if enabled}
			<!-- Check Frequency -->
			<div>
				<label
					for="check-frequency"
					class="block text-sm font-medium text-gray-900 dark:text-gray-100 mb-2"
				>
					Check frequency
				</label>
				<select
					id="check-frequency"
					bind:value={checkFrequency}
					on:change={(e) => handleFrequencyChange(e.currentTarget.value as typeof checkFrequency)}
					class="block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 sm:text-sm"
				>
					{#each frequencyOptions as option}
						<option value={option.value}>{option.label}</option>
					{/each}
				</select>
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					{#if checkFrequency === "on_startup"}
						Checks for updates only when the app starts
					{:else if checkFrequency === "daily"}
						Checks for updates once per day
					{:else if checkFrequency === "weekly"}
						Checks for updates once per week
					{:else}
						Automatic checks are disabled
					{/if}
				</p>
			</div>

			<!-- Include Pre-releases -->
			<div class="flex items-center justify-between">
				<div class="flex-1">
					<label
						for="include-prereleases"
						class="block text-sm font-medium text-gray-900 dark:text-gray-100"
					>
						Include pre-releases
					</label>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
						Check for beta and release candidate versions
					</p>
				</div>
				<button
					type="button"
					id="include-prereleases"
					role="switch"
					aria-checked={includePrereleases}
					aria-label="Include pre-releases"
					on:click={() => handlePrereleasesChange(!includePrereleases)}
					class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 dark:focus:ring-offset-gray-950 {includePrereleases
						? 'bg-indigo-600'
						: 'bg-gray-200 dark:bg-gray-700'}"
				>
					<span
						class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {includePrereleases
							? 'translate-x-5'
							: 'translate-x-0'}"
						aria-hidden="true"
					></span>
				</button>
			</div>

			<!-- Last Checked -->
			{#if $lastCheckedAt}
				<div class="text-sm text-gray-600 dark:text-gray-400">
					Last checked: {formatLastChecked($lastCheckedAt)}
				</div>
			{/if}

			<!-- Manual Check Button -->
			<div>
				<button
					type="button"
					on:click={handleManualCheck}
					disabled={$isChecking}
					class="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-indigo-500 dark:hover:bg-indigo-600 dark:focus:ring-offset-gray-950"
				>
					{#if $isChecking}
						<svg
							class="h-4 w-4 animate-spin"
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
						>
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
					{/if}
					{$isChecking ? "Checking..." : "Check for updates now"}
				</button>
			</div>

			<!-- Update Available Info -->
			{#if $updateInfo?.updateAvailable}
				<div class="rounded-md bg-blue-50 p-4 dark:bg-blue-950/50">
					<div class="flex items-start gap-3 mb-3">
						<div class="flex-shrink-0">
							<svg
								class="h-5 w-5 text-blue-400"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fill-rule="evenodd"
									d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
									clip-rule="evenodd"
								/>
							</svg>
						</div>
						<h3 class="text-sm font-medium text-blue-800 dark:text-blue-200">
							Update Available
						</h3>
					</div>
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1 min-w-0">
							<div class="text-sm text-blue-700 dark:text-blue-300">
								<p>
									Version {$updateInfo.latestVersion} is available
									{#if $updateInfo.currentVersion}
										(current: {$updateInfo.currentVersion})
									{/if}
								</p>
								{#if $updateInfo.releaseName}
									<p class="mt-1 font-medium">{$updateInfo.releaseName}</p>
								{/if}
							</div>
						</div>
						{#if $updateInfo.downloadUrl}
							<div class="flex-shrink-0">
								<button
									type="button"
									on:click={handleDownload}
									disabled={isDownloading}
									class="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-indigo-500 dark:hover:bg-indigo-600 dark:focus:ring-offset-gray-950"
								>
									{#if isDownloading}
										<svg
											class="h-3 w-3 animate-spin"
											xmlns="http://www.w3.org/2000/svg"
											fill="none"
											viewBox="0 0 24 24"
										>
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
									{:else}
										<img
											src="/download.svg"
											alt="Download"
											class="h-3 w-3 brightness-0 invert"
										/>
									{/if}
									{isDownloading ? "Downloading..." : "Download Update"}
								</button>
							</div>
						{/if}
					</div>
				</div>
			{/if}
		{/if}

		{#if $error}
			<div class="rounded-md bg-red-50 p-4 dark:bg-red-950/50">
				<p class="text-sm text-red-800 dark:text-red-200">{$error}</p>
			</div>
		{/if}
	</div>
{/if}

