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
	import {
		getGitHubStatus,
		setGitHubToken,
		clearGitHubToken,
		type GitHubStatus,
	} from "$lib/api/user";
	import { runOAuthFlow } from "$lib/api/oauth";
	import { toastStore } from "$lib/stores/toastStore";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";

	let status: GitHubStatus | null = null;
	let isLoading = true;
	let isConnectingOAuth = false;
	let oauthUserCode = "";
	let isClearing = false;
	let showConfirmDialog = false;
	let error = "";

	// PAT fallback state
	let showPatInput = false;
	let tokenInput = "";
	let isUpdating = false;
	let showTokenUpdate = false;

	async function loadStatus() {
		try {
			status = await getGitHubStatus();
		} catch (err) {
			console.error("Failed to load GitHub status:", err);
			status = { connected: false, source: "none" };
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadStatus();
	});

	async function handleConnectWithGitHub() {
		error = "";
		isConnectingOAuth = true;
		oauthUserCode = "";

		try {
			const username = await runOAuthFlow((flowStatus, message) => {
				if (flowStatus === "waiting" && message) {
					const match = message.match(/Enter code: (.+)/);
					if (match) {
						oauthUserCode = match[1];
					}
				}
			});
			await loadStatus();
			showTokenUpdate = false;
			toastStore.success(`Connected to GitHub as ${username}`);
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to connect with GitHub";
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
			error = "Please enter a token";
			return;
		}

		error = "";
		isUpdating = true;

		try {
			const result = await setGitHubToken(tokenInput.trim());
			await loadStatus();
			tokenInput = "";
			showPatInput = false;
			showTokenUpdate = false;
			toastStore.success(`Connected to GitHub as ${result.githubUsername}`);
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to validate token";
		} finally {
			isUpdating = false;
		}
	}

	function handleDisconnectClick() {
		if (!status?.connected) return;

		// Show confirmation dialog
		showConfirmDialog = true;
	}

	async function performClearToken() {
		showConfirmDialog = false;
		isClearing = true;

		try {
			const newStatus = await clearGitHubToken();
			status = newStatus;
			toastStore.success("Disconnected from GitHub");
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : "Failed to clear token");
		} finally {
			isClearing = false;
		}
	}

	function handleCancelDisconnect() {
		showConfirmDialog = false;
	}

	function handleCancelUpdate() {
		showPatInput = false;
		showTokenUpdate = false;
		tokenInput = "";
		error = "";
	}

	$: sourceText =
		status?.source === "oauth"
			? "Connected via GitHub"
			: status?.source === "stored"
				? "Connected via token"
				: "";
</script>

<div>
	<div>
		<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">GitHub Connection</h3>
		<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
			Manage Octobud's connection with GitHub
		</p>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-6 mt-4">
			<svg class="h-5 w-5 animate-spin text-gray-500" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
				></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				></path>
			</svg>
		</div>
	{:else if status?.connected && !showTokenUpdate}
		<!-- Connected state -->
		<div
			class="mt-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200 dark:border-gray-800"
		>
			<!-- Nested status card -->
			<div
				class="bg-white dark:bg-gray-800/50 rounded-lg p-3 border border-gray-100 dark:border-gray-700/50"
			>
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
								>{status.githubUsername}</span
							>
						</p>
						<p class="text-xs text-gray-500 dark:text-gray-400">
							{sourceText}
						</p>
					</div>
				</div>
			</div>

			<div class="mt-4 flex justify-end items-center gap-4">
				{#if status.source === "stored" || status.source === "oauth"}
					<button
						type="button"
						on:click={handleDisconnectClick}
						disabled={isUpdating || isClearing || isConnectingOAuth}
						class="text-xs font-medium text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 disabled:text-gray-400 dark:disabled:text-gray-600 disabled:cursor-not-allowed transition-colors cursor-pointer"
					>
						{#if isClearing}
							Disconnecting...
						{:else}
							Disconnect
						{/if}
					</button>
				{/if}
				<button
					type="button"
					on:click={() => (showTokenUpdate = true)}
					disabled={isUpdating || isClearing || isConnectingOAuth}
					class="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
				>
					Reconnect
				</button>
			</div>
		</div>
	{:else if isConnectingOAuth}
		<!-- Connecting via OAuth state -->
		<div
			class="mt-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200 dark:border-gray-800"
		>
			<div class="text-center py-4">
				<div class="flex justify-center mb-4">
					<svg class="h-6 w-6 animate-spin text-indigo-500" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
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
						<p class="text-xs text-gray-500 dark:text-gray-400 mb-3">Enter this code on GitHub:</p>
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
		</div>
	{:else}
		<!-- Not connected or reconnecting -->
		<div
			class="mt-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200 dark:border-gray-800"
		>
			<div class="space-y-4">
				{#if error}
					<div
						class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
					>
						{error}
					</div>
				{/if}

				<!-- OAuth primary button -->
				<div class="text-center py-4">
					{#if !status?.connected}
						<p class="text-sm text-gray-600 dark:text-gray-400 mb-5">
							Connect your GitHub account to sync notifications
						</p>
					{/if}
					<button
						type="button"
						on:click={handleConnectWithGitHub}
						disabled={isUpdating || isConnectingOAuth}
						class="inline-flex items-center gap-3 rounded-xl bg-gray-900 dark:bg-white px-6 py-3 text-sm font-semibold text-white dark:text-gray-900 transition-all hover:bg-gray-800 dark:hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer shadow-sm hover:shadow-md"
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
				<div class="pt-4 border-t border-gray-200 dark:border-gray-700">
					<button
						type="button"
						on:click={() => (showPatInput = !showPatInput)}
						class="flex items-center gap-2 text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300 cursor-pointer w-full"
					>
						<svg
							class="h-4 w-4 transition-transform {showPatInput ? 'rotate-90' : ''}"
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
						Use Personal Access Token instead
					</button>

					{#if showPatInput}
						<div class="mt-4 space-y-3">
							<p class="text-xs text-gray-500 dark:text-gray-400">
								Token needs <code class="text-xs bg-gray-200 dark:bg-gray-700 px-1 py-0.5 rounded"
									>notifications</code
								>
								and
								<code class="text-xs bg-gray-200 dark:bg-gray-700 px-1 py-0.5 rounded">repo</code>
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

							<div class="flex gap-2">
								<input
									id="github-token"
									type="password"
									placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
									bind:value={tokenInput}
									disabled={isUpdating}
									class="flex-1 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-gray-500 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
									on:keydown={(e) => e.key === "Enter" && handleSetToken()}
								/>
								<button
									type="button"
									on:click={handleSetToken}
									disabled={isUpdating || !tokenInput.trim()}
									class="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
								>
									{#if isUpdating}
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

				{#if showTokenUpdate}
					<div class="text-center">
						<button
							type="button"
							on:click={handleCancelUpdate}
							disabled={isUpdating || isConnectingOAuth}
							class="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 cursor-pointer"
						>
							Cancel
						</button>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<ConfirmDialog
	open={showConfirmDialog}
	title="Disconnect from GitHub?"
	body="Are you sure you want to disconnect from GitHub? You will need to enter a new token to sync notifications."
	confirmLabel="Disconnect"
	cancelLabel="Cancel"
	confirming={isClearing}
	onConfirm={performClearToken}
	onCancel={handleCancelDisconnect}
/>
