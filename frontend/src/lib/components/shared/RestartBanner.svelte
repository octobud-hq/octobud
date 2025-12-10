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
	import { slide } from "svelte/transition";
	import { getUpdateStore } from "$lib/stores/updateStore";
	import { restartApp } from "$lib/api/user";

	const updateStore = getUpdateStore();

	// Access restart-related stores
	const { restartNeeded, installedVersion } = updateStore;

	$: shouldShow = $restartNeeded && $installedVersion !== null;
	$: currentInstalledVersion = $installedVersion;

	let isRestarting = false;

	async function handleRestart() {
		isRestarting = true;
		try {
			await restartApp();
			// The app will restart, so the page will become unresponsive
			// This is expected behavior
		} catch (err) {
			console.error("Failed to restart:", err);
			alert("Failed to restart. Please quit Octobud manually from the menu bar and relaunch it.");
			isRestarting = false;
		}
	}

	async function handleDismiss() {
		// Dismiss for the current session only (don't persist)
		// The check will run again in 5 minutes anyway
		updateStore.dismissRestartBanner();
	}
</script>

{#if shouldShow && currentInstalledVersion}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-orange-200 bg-orange-50 px-4 py-3 dark:border-orange-800 dark:bg-orange-950/50"
		role="alert"
		aria-live="polite"
	>
		<div class="flex items-center gap-3 flex-1">
			<!-- Warning icon -->
			<svg
				class="h-5 w-5 flex-shrink-0 text-orange-600 dark:text-orange-400"
				fill="currentColor"
				viewBox="0 0 20 20"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					fill-rule="evenodd"
					d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
					clip-rule="evenodd"
				/>
			</svg>

			<div class="flex-1">
				<p class="text-sm font-medium text-orange-900 dark:text-orange-100">
					New version installed ({currentInstalledVersion})
				</p>
				<p class="text-xs text-orange-700 dark:text-orange-300 mt-0.5">
					Restart Octobud to use the new version
				</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				type="button"
				on:click={handleRestart}
				disabled={isRestarting}
				class="inline-flex items-center gap-2 flex-shrink-0 rounded-full bg-orange-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-orange-600 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-orange-500 dark:hover:bg-orange-600 dark:focus:ring-offset-gray-950"
			>
				{#if isRestarting}
					<svg
						class="h-3 w-3 animate-spin"
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
					>
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
				{:else}
					<svg
						class="h-3 w-3"
						fill="currentColor"
						viewBox="0 0 20 20"
						xmlns="http://www.w3.org/2000/svg"
					>
						<path
							fill-rule="evenodd"
							d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
							clip-rule="evenodd"
						/>
					</svg>
				{/if}
				{isRestarting ? "Restarting..." : "Restart Now"}
			</button>
			<button
				type="button"
				on:click={handleDismiss}
				class="flex-shrink-0 rounded-md p-1 text-orange-700 transition hover:bg-orange-100 focus:outline-none focus:ring-2 focus:ring-orange-600 focus:ring-offset-2 dark:text-orange-300 dark:hover:bg-orange-900/50 dark:focus:ring-offset-gray-950"
				aria-label="Dismiss"
			>
				<svg
					class="h-5 w-5"
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<line x1="18" y1="6" x2="6" y2="18"></line>
					<line x1="6" y1="6" x2="18" y2="18"></line>
				</svg>
			</button>
		</div>
	</div>
{/if}
