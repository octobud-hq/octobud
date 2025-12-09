<script lang="ts">
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

	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { hasSyncSettingsConfigured, fetchUserInfo } from "$lib/stores/authStore";
	import {
		updateSyncSettings,
		getGitHubStatus,
		setGitHubToken,
		clearGitHubToken,
		type GitHubStatus,
	} from "$lib/api/user";
	import { runOAuthFlow } from "$lib/api/oauth";
	import { resetSWPollState } from "$lib/utils/serviceWorkerRegistration";

	type DaySelection = number | "custom" | "all";
	let selectedOption: DaySelection = 30; // Default to 30 days
	let customDays: number | null = null;
	let maxNotifications: number | null = null;
	let syncUnreadOnly = false;
	let error = "";
	let isLoading = true;
	let isSubmitting = false;

	// GitHub connection state
	let githubStatus: GitHubStatus | null = null;
	let isConnectingOAuth = false;
	let oauthUserCode = "";
	let isClearingToken = false;
	let connectionError = "";

	// PAT fallback state (collapsed by default)
	let showPatInput = false;
	let tokenInput = "";
	let isValidatingToken = false;
	let tokenError = "";

	// Predefined day options
	const dayOptions = [30, 60, 90];

	// Validation limits (must match backend)
	const MAX_SYNC_DAYS = 3650; // 10 years
	const MAX_NOTIFICATION_COUNT = 100000;

	// Show warning for large sync periods
	$: showLargeSyncWarning =
		selectedOption === "all" ||
		(selectedOption === "custom" && customDays !== null && customDays > 90);

	// Check if we can proceed (GitHub must be connected)
	$: canProceed = githubStatus?.connected === true;

	// Get effective days for submission
	function getEffectiveDays(): number | null {
		if (selectedOption === "all") return null;
		if (selectedOption === "custom") return customDays;
		return selectedOption;
	}

	async function loadGitHubStatus() {
		try {
			githubStatus = await getGitHubStatus();
		} catch (err) {
			console.error("Failed to fetch GitHub status:", err);
			// Set a default state
			githubStatus = { connected: false, source: "none" };
		}
	}

	onMount(async () => {
		// Fetch user info and GitHub status
		try {
			await Promise.all([fetchUserInfo(), loadGitHubStatus()]);
			const hasConfigured = hasSyncSettingsConfigured();
			if (hasConfigured) {
				// Already configured, redirect to main app
				await goto(resolve("/views/inbox"));
			}
		} catch (err) {
			console.error("Failed to fetch initial data:", err);
			// Continue to setup page even if fetch fails
		} finally {
			isLoading = false;
		}
	});

	async function handleConnectWithGitHub() {
		connectionError = "";
		isConnectingOAuth = true;
		oauthUserCode = "";

		try {
			await runOAuthFlow((status, message) => {
				if (status === "waiting" && message) {
					// Extract user code from message like "Enter code: XXXX-XXXX"
					const match = message.match(/Enter code: (.+)/);
					if (match) {
						oauthUserCode = match[1];
					}
				}
			});
			// Refresh status after OAuth completes
			await loadGitHubStatus();
		} catch (err) {
			connectionError = err instanceof Error ? err.message : "Failed to connect with GitHub";
		} finally {
			isConnectingOAuth = false;
			oauthUserCode = "";
		}
	}

	function handleCancelOAuth() {
		isConnectingOAuth = false;
		oauthUserCode = "";
	}

	async function handleSetToken() {
		if (!tokenInput.trim()) {
			tokenError = "Please enter a token";
			return;
		}

		tokenError = "";
		isValidatingToken = true;

		try {
			await setGitHubToken(tokenInput.trim());
			// Refresh status after setting token
			await loadGitHubStatus();
			tokenInput = "";
			showPatInput = false;
		} catch (err) {
			tokenError = err instanceof Error ? err.message : "Failed to validate token";
		} finally {
			isValidatingToken = false;
		}
	}

	async function handleClearToken() {
		isClearingToken = true;
		connectionError = "";
		tokenError = "";

		try {
			await clearGitHubToken();
			await loadGitHubStatus();
		} catch (err) {
			connectionError = err instanceof Error ? err.message : "Failed to disconnect";
		} finally {
			isClearingToken = false;
		}
	}

	function handleDaysChange(option: DaySelection) {
		selectedOption = option;
		if (option === "custom") {
			customDays = customDays || 30;
		} else if (option !== "all") {
			customDays = null;
		}
	}

	async function handleSubmit() {
		error = "";
		isSubmitting = true;

		try {
			const initialSyncDays = getEffectiveDays();

			// Validate - allow null (all time) but reject invalid custom values
			if (selectedOption === "custom") {
				if (!customDays || customDays < 1) {
					error = "Please enter a valid number of days";
					isSubmitting = false;
					return;
				}
				if (customDays > MAX_SYNC_DAYS) {
					error = `Days cannot exceed ${MAX_SYNC_DAYS} (10 years)`;
					isSubmitting = false;
					return;
				}
			}

			// Validate max notifications if provided
			if (maxNotifications !== null && maxNotifications > MAX_NOTIFICATION_COUNT) {
				error = `Maximum notifications cannot exceed ${MAX_NOTIFICATION_COUNT.toLocaleString()}`;
				isSubmitting = false;
				return;
			}

			await updateSyncSettings({
				initialSyncDays,
				initialSyncMaxCount: maxNotifications || null,
				initialSyncUnreadOnly: syncUnreadOnly,
				setupCompleted: true,
			});

			// Refresh user info to update store
			await fetchUserInfo();

			// Reset SW poll state so all notifications are treated as new
			await resetSWPollState();

			// Set sync start timestamp so the banner shows until notifications arrive
			if (typeof window !== "undefined") {
				localStorage.setItem("octobud:sync-started-at", Date.now().toString());
			}

			// Redirect to main app
			await goto(resolve("/views/inbox"));
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to save sync settings";
			isSubmitting = false;
		}
	}

	// Use default recommendation: 30 days
	function useRecommended() {
		selectedOption = 30;
		customDays = null;
		maxNotifications = null;
		syncUnreadOnly = false;
	}
</script>

<div class="relative min-h-screen overflow-y-auto bg-indigo-50/30 pt-16 dark:bg-gray-950">
	<div class="mx-auto w-full max-w-xl px-4 py-10 sm:px-6 sm:py-12">
		<!-- Header -->
		<div class="flex items-center gap-4">
			<div
				class="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-600"
			>
				<img src="/baby_octo.svg" alt="Octobud" class="h-9 w-9" />
			</div>
			<div>
				<h1 class="text-xl font-semibold text-gray-900 dark:text-white">Welcome to Octobud!</h1>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					Connect your GitHub account and configure sync settings to get started
				</p>
			</div>
		</div>

		{#if isLoading}
			<div class="mt-12 flex items-center justify-center py-12">
				<svg class="h-8 w-8 animate-spin text-indigo-500" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
			</div>
		{:else}
			<!-- Step 1: GitHub Connection -->
			<div class="mt-8">
				<div class="flex items-center gap-3 mb-4">
					<div
						class="flex h-7 w-7 items-center justify-center rounded-full {githubStatus?.connected
							? 'bg-green-100 dark:bg-green-900/30'
							: 'bg-indigo-100 dark:bg-indigo-900/30'}"
					>
						{#if githubStatus?.connected}
							<svg
								class="h-4 w-4 text-green-600 dark:text-green-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M5 13l4 4L19 7"
								/>
							</svg>
						{:else}
							<span class="text-sm font-semibold text-indigo-600 dark:text-indigo-400">1</span>
						{/if}
					</div>
					<h2 class="text-base font-semibold text-gray-900 dark:text-white">Connect GitHub</h2>
				</div>

				<div
					class="rounded-2xl bg-white p-6 shadow-[0_4px_24px_-4px_rgba(0,0,0,0.08)] dark:border dark:border-gray-800 dark:bg-gray-900 dark:shadow-none"
				>
					{#if githubStatus?.connected}
						<!-- Connected state -->
						<div class="flex items-center justify-between gap-3">
							<div class="flex items-center gap-3">
								<div
									class="flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30"
								>
									<svg
										class="h-5 w-5 text-green-600 dark:text-green-400"
										fill="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"
										/>
									</svg>
								</div>
								<div>
									<p class="text-sm font-medium text-gray-900 dark:text-white">
										Connected as <span class="text-green-600 dark:text-green-400"
											>{githubStatus.githubUsername}</span
										>
									</p>
									<p class="text-xs text-gray-500 dark:text-gray-400">
										{#if githubStatus.source === "oauth"}
											Connected via GitHub
										{:else if githubStatus.source === "stored"}
											Connected via token
										{/if}
									</p>
								</div>
							</div>
							<button
								type="button"
								on:click={handleClearToken}
								disabled={isClearingToken}
								class="text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300 disabled:opacity-50 cursor-pointer"
							>
								{isClearingToken ? "Resetting..." : "Reset"}
							</button>
						</div>
					{:else if isConnectingOAuth}
						<!-- Connecting via OAuth state -->
						<div class="text-center py-6">
							<div class="flex justify-center mb-4">
								<svg class="h-6 w-6 animate-spin text-indigo-500" fill="none" viewBox="0 0 24 24">
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
							</div>
							<p class="text-sm font-medium text-gray-900 dark:text-white mb-5">
								Waiting for GitHub authorization...
							</p>
							{#if oauthUserCode}
								<div class="mb-5">
									<p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
										Enter this code on GitHub:
									</p>
									<div
										class="inline-flex items-center justify-center rounded-xl bg-gray-100 dark:bg-gray-800 px-6 py-4 border-2 border-gray-200 dark:border-gray-700"
									>
										<code
											class="font-mono text-2xl font-bold tracking-widest text-gray-900 dark:text-white"
											>{oauthUserCode}</code
										>
									</div>
									<p class="text-xs text-gray-400 dark:text-gray-500 mt-3">
										or go to <a
											href="https://github.com/login/device"
											target="_blank"
											rel="noopener noreferrer"
											class="text-indigo-600 dark:text-indigo-400 hover:underline"
											>github.com/login/device</a
										>
									</p>
								</div>
							{/if}
							<button
								type="button"
								on:click={handleCancelOAuth}
								class="text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300 cursor-pointer"
							>
								Cancel
							</button>
						</div>
					{:else}
						<!-- Not connected state - OAuth primary -->
						<div class="space-y-6">
							{#if connectionError}
								<div
									class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
								>
									{connectionError}
								</div>
							{/if}

							<div class="text-center py-4">
								<p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
									Connect your GitHub account to sync notifications
								</p>
								<button
									type="button"
									on:click={handleConnectWithGitHub}
									class="inline-flex items-center gap-3 rounded-xl bg-gray-900 dark:bg-white px-7 py-3.5 text-sm font-semibold text-white dark:text-gray-900 transition-all hover:bg-gray-800 dark:hover:bg-gray-100 cursor-pointer shadow-sm hover:shadow-md"
								>
									<svg class="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
										<path
											d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"
										/>
									</svg>
									Connect with GitHub
								</button>
							</div>

							<!-- PAT fallback (collapsed) -->
							<div class="pt-2">
								<button
									type="button"
									on:click={() => (showPatInput = !showPatInput)}
									class="flex items-center gap-1.5 text-xs text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-400 cursor-pointer"
								>
									<svg
										class="h-3 w-3 transition-transform {showPatInput ? 'rotate-90' : ''}"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2.5"
									>
										<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
									</svg>
									Use Personal Access Token instead
								</button>

								{#if showPatInput}
									<div class="mt-4 space-y-3">
										<p class="text-xs text-gray-500 dark:text-gray-400">
											Token needs <code
												class="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded"
												>notifications</code
											>
											and
											<code class="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded"
												>repo</code
											>
											scopes.
											<a
												href="https://github.com/settings/tokens/new?scopes=notifications,repo&description=Octobud"
												target="_blank"
												rel="noopener noreferrer"
												class="inline-flex items-center gap-1 text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
											>
												Create one
												<svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
													/>
												</svg>
											</a>
										</p>

										{#if tokenError}
											<div
												class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
											>
												{tokenError}
											</div>
										{/if}

										<div class="flex gap-2">
											<input
												type="password"
												placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
												bind:value={tokenInput}
												disabled={isValidatingToken}
												class="flex-1 rounded-xl border-2 border-gray-200 bg-white px-4 py-2.5 text-sm text-gray-900 placeholder-gray-400 transition-colors focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
												on:keydown={(e) => e.key === "Enter" && handleSetToken()}
											/>
											<button
												type="button"
												on:click={handleSetToken}
												disabled={isValidatingToken || !tokenInput.trim()}
												class="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white transition-all hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
											>
												{#if isValidatingToken}
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
													Validating...
												{:else}
													Connect
												{/if}
											</button>
										</div>
									</div>
								{/if}
							</div>
						</div>
					{/if}
				</div>
			</div>

			<!-- Step 2: Sync Settings -->
			<div class="mt-8">
				<div class="flex items-center gap-3 mb-4">
					<div
						class="flex h-7 w-7 items-center justify-center rounded-full {canProceed
							? 'bg-indigo-100 dark:bg-indigo-900/30'
							: 'bg-gray-100 dark:bg-gray-800'}"
					>
						<span
							class="text-sm font-semibold {canProceed
								? 'text-indigo-600 dark:text-indigo-400'
								: 'text-gray-400 dark:text-gray-500'}">2</span
						>
					</div>
					<h2
						class="text-base font-semibold {canProceed
							? 'text-gray-900 dark:text-white'
							: 'text-gray-400 dark:text-gray-500'}"
					>
						Configure Sync
					</h2>
				</div>

				<!-- Main form card -->
				<form class="relative" on:submit|preventDefault={handleSubmit}>
					{#if !canProceed}
						<div
							class="absolute inset-0 bg-white/50 dark:bg-gray-950/50 z-10 rounded-2xl"
							aria-hidden="true"
						></div>
					{/if}

					{#if error}
						<div
							class="mb-6 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
							role="alert"
						>
							{error}
						</div>
					{/if}

					<div
						class="rounded-2xl bg-white p-6 shadow-[0_4px_24px_-4px_rgba(0,0,0,0.08)] dark:border dark:border-gray-800 dark:bg-gray-900 dark:shadow-none sm:p-8 {!canProceed
							? 'opacity-60'
							: ''}"
					>
						<!-- Time period selection -->
						<div>
							<h3 class="text-sm font-semibold text-gray-900 dark:text-white">
								Sync notifications from
							</h3>
							<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
								Choose a time period or sync everything
							</p>
							<div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
								{#each dayOptions as days (days)}
									<button
										type="button"
										on:click={() => handleDaysChange(days)}
										class="relative rounded-xl border-2 px-4 py-3 text-center transition-all cursor-pointer {selectedOption ===
										days
											? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500 dark:border-indigo-400 dark:bg-indigo-500/10 dark:ring-indigo-400'
											: 'border-gray-200 bg-gray-50 hover:border-gray-300 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600 dark:hover:bg-gray-700/50'}"
										disabled={isSubmitting}
									>
										<span
											class="block text-lg font-semibold {selectedOption === days
												? 'text-indigo-600 dark:text-indigo-400'
												: 'text-gray-900 dark:text-white'}"
										>
											{days}
										</span>
										<span
											class="block text-xs {selectedOption === days
												? 'text-indigo-500 dark:text-indigo-400'
												: 'text-gray-500 dark:text-gray-400'}"
										>
											days
										</span>
									</button>
								{/each}
								<button
									type="button"
									on:click={() => handleDaysChange("all")}
									class="relative rounded-xl border-2 px-4 py-3 text-center transition-all cursor-pointer {selectedOption ===
									'all'
										? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500 dark:border-indigo-400 dark:bg-indigo-500/10 dark:ring-indigo-400'
										: 'border-gray-200 bg-gray-50 hover:border-gray-300 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600 dark:hover:bg-gray-700/50'}"
									disabled={isSubmitting}
								>
									<span
										class="block text-lg font-semibold {selectedOption === 'all'
											? 'text-indigo-600 dark:text-indigo-400'
											: 'text-gray-900 dark:text-white'}"
									>
										All
									</span>
									<span
										class="block text-xs {selectedOption === 'all'
											? 'text-indigo-500 dark:text-indigo-400'
											: 'text-gray-500 dark:text-gray-400'}"
									>
										time
									</span>
								</button>
							</div>

							<!-- Custom input -->
							<button
								type="button"
								on:click={() => handleDaysChange("custom")}
								class="mt-3 w-full rounded-xl border-2 px-4 py-3 text-left transition-all cursor-pointer {selectedOption ===
								'custom'
									? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500 dark:border-indigo-400 dark:bg-indigo-500/10 dark:ring-indigo-400'
									: 'border-gray-200 bg-gray-50 hover:border-gray-300 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600 dark:hover:bg-gray-700/50'}"
								disabled={isSubmitting}
							>
								<span
									class="text-sm font-medium {selectedOption === 'custom'
										? 'text-indigo-600 dark:text-indigo-400'
										: 'text-gray-700 dark:text-gray-300'}"
								>
									Custom number of days
								</span>
							</button>

							{#if selectedOption === "custom"}
								<div class="mt-3">
									<input
										type="number"
										min="1"
										max={MAX_SYNC_DAYS}
										placeholder="Enter number of days"
										bind:value={customDays}
										disabled={isSubmitting}
										class="w-full rounded-xl border-2 border-gray-200 bg-white px-4 py-3 text-gray-900 placeholder-gray-400 transition-colors focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
									/>
								</div>
							{/if}

							{#if showLargeSyncWarning}
								<div
									class="mt-4 flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-700/50 dark:bg-amber-900/20"
								>
									<svg
										class="mt-0.5 h-5 w-5 flex-shrink-0 text-amber-500 dark:text-amber-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
										/>
									</svg>
									<div class="text-sm text-amber-800 dark:text-amber-300">
										{#if selectedOption === "all"}
											<span class="font-medium">This may take a while.</span> Syncing all notifications
											could include thousands of items.
										{:else}
											<span class="font-medium">Large sync.</span> Consider starting smaller-you can always
											sync more later.
										{/if}
									</div>
								</div>
							{/if}
						</div>

						<!-- Advanced options -->
						<details class="group mt-8 border-t border-gray-200 pt-6 dark:border-gray-700">
							<summary
								class="flex cursor-pointer list-none items-center justify-between text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								<span>Advanced options</span>
								<svg
									class="h-5 w-5 text-gray-400 transition-transform group-open:rotate-180"
									fill="none"
									viewBox="0 0 24 24"
									stroke="currentColor"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 9l-7 7-7-7"
									/>
								</svg>
							</summary>

							<div class="mt-4 space-y-4">
								<!-- Max notifications -->
								<div>
									<label
										for="maxNotifications"
										class="block text-sm font-medium text-gray-700 dark:text-gray-300"
									>
										Maximum notifications
									</label>
									<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
										Stop syncing after reaching this limit
									</p>
									<input
										id="maxNotifications"
										type="number"
										min="1"
										max={MAX_NOTIFICATION_COUNT}
										placeholder="No limit"
										bind:value={maxNotifications}
										disabled={isSubmitting}
										class="mt-2 w-full rounded-xl border-2 border-gray-200 bg-white px-4 py-2.5 text-gray-900 placeholder-gray-400 transition-colors focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
									/>
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
							</div>
						</details>
					</div>

					<!-- Actions -->
					<div class="mt-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
						<button
							type="button"
							on:click={useRecommended}
							disabled={isSubmitting || !canProceed}
							class="order-2 text-sm text-gray-500 transition-colors hover:text-gray-700 disabled:opacity-50 dark:text-gray-400 dark:hover:text-gray-300 sm:order-1 cursor-pointer"
						>
							Reset to recommended
						</button>
						<button
							type="submit"
							disabled={isSubmitting || !canProceed}
							class="order-1 inline-flex items-center justify-center gap-2 rounded-xl bg-indigo-600 px-6 py-2.5 text-sm font-semibold text-white transition-all hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-gray-950 sm:order-2 cursor-pointer"
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
								Starting sync...
							{:else}
								Start syncing
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M13 7l5 5m0 0l-5 5m5-5H6"
									/>
								</svg>
							{/if}
						</button>
					</div>
				</form>
			</div>

			<!-- Footer note -->
			<p class="mt-8 text-center text-xs text-gray-400 dark:text-gray-500">
				You can sync additional history and manage your GitHub connection in the settings.
			</p>
		{/if}
	</div>
</div>
