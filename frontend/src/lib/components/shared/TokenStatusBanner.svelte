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
	import { readable, type Readable } from "svelte/store";
	import { slide } from "svelte/transition";
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import {
		getGithubStatusStore,
		type createGithubStatusStore,
	} from "$lib/stores/githubStatusStore";
	import { currentTime } from "$lib/stores/timeStore";
	import type { GitHubStatus } from "$lib/api/user";

	// Avoid instantiating the singleton during SSR — the layout mirrors this guard.
	const store: ReturnType<typeof createGithubStatusStore> | null = browser
		? getGithubStatusStore()
		: null;
	const status: Readable<GitHubStatus | null> = store?.status ?? readable(null);
	const bannerState: Readable<"invalid" | "expiring" | null> = store?.bannerState ?? readable(null);

	$: expiresAt = $status?.tokenExpiresAt ? new Date($status.tokenExpiresAt) : null;

	// "in 3 days", "in 12 hours", "in 45 minutes", or "soon" as a floor.
	// Depends on $currentTime so the countdown ticks as time passes.
	$: timeUntilLabel = (() => {
		if (!expiresAt) return null;
		const ms = expiresAt.getTime() - $currentTime.getTime();
		if (ms <= 0) return null;
		const minutes = Math.floor(ms / (60 * 1000));
		const hours = Math.floor(minutes / 60);
		const days = Math.floor(hours / 24);
		if (days >= 1) return `in ${days} day${days === 1 ? "" : "s"}`;
		if (hours >= 1) return `in ${hours} hour${hours === 1 ? "" : "s"}`;
		if (minutes >= 1) return `in ${minutes} minute${minutes === 1 ? "" : "s"}`;
		return "soon";
	})();

	function handleOpenSettings() {
		void goto("/settings");
	}

	function handleDismiss() {
		store?.dismissExpiring();
	}
</script>

{#if $bannerState === "invalid"}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-950/50"
		role="alert"
		aria-live="assertive"
	>
		<div class="flex items-center gap-3 flex-1">
			<svg
				class="h-5 w-5 flex-shrink-0 text-red-900 dark:text-red-100"
				fill="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
				aria-hidden="true"
			>
				<path
					fill-rule="evenodd"
					clip-rule="evenodd"
					d="M12 2.25C6.615 2.25 2.25 6.615 2.25 12S6.615 21.75 12 21.75 21.75 17.385 21.75 12 17.385 2.25 12 2.25ZM12 7.25a.75.75 0 0 1 .75.75v4.5a.75.75 0 0 1-1.5 0V8a.75.75 0 0 1 .75-.75Zm0 9.25a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
				/>
			</svg>

			<div class="flex-1">
				<p class="text-sm font-medium text-red-900 dark:text-red-100">
					GitHub token is no longer valid
				</p>
				<p class="text-xs text-red-700 dark:text-red-300 mt-0.5">
					Reconnect in Settings to keep syncing notifications.
				</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				type="button"
				on:click={handleOpenSettings}
				class="inline-flex items-center gap-2 flex-shrink-0 rounded-full bg-red-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-600 focus:ring-offset-2 dark:bg-red-500 dark:hover:bg-red-600 dark:focus:ring-offset-gray-950"
			>
				Open Settings
			</button>
		</div>
	</div>
{:else if $bannerState === "expiring"}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-800 dark:bg-amber-950/50"
		role="status"
		aria-live="polite"
	>
		<div class="flex items-center gap-3 flex-1">
			<svg
				class="h-5 w-5 flex-shrink-0 text-amber-900 dark:text-amber-100"
				fill="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
				aria-hidden="true"
			>
				<path
					fill-rule="evenodd"
					clip-rule="evenodd"
					d="M10.908 2.81a1.25 1.25 0 0 1 2.184 0l9.25 16.5A1.25 1.25 0 0 1 21.25 21.25H2.75a1.25 1.25 0 0 1-1.092-1.94l9.25-16.5ZM12 8a.75.75 0 0 1 .75.75v5a.75.75 0 0 1-1.5 0v-5A.75.75 0 0 1 12 8Zm0 9.5a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
				/>
			</svg>

			<div class="flex-1">
				<p class="text-sm font-medium text-amber-900 dark:text-amber-100">
					GitHub token expires {timeUntilLabel ?? "soon"}
				</p>
				<p class="text-xs text-amber-700 dark:text-amber-300 mt-0.5">
					Update it in Settings before it expires to avoid sync interruptions.
				</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				type="button"
				on:click={handleOpenSettings}
				class="inline-flex items-center gap-2 flex-shrink-0 rounded-full bg-amber-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-amber-700 focus:outline-none focus:ring-2 focus:ring-amber-600 focus:ring-offset-2 dark:bg-amber-500 dark:hover:bg-amber-600 dark:focus:ring-offset-gray-950"
			>
				Open Settings
			</button>
			<button
				type="button"
				on:click={handleDismiss}
				class="flex-shrink-0 rounded-md p-1 text-amber-700 transition hover:bg-amber-100 focus:outline-none focus:ring-2 focus:ring-amber-600 focus:ring-offset-2 dark:text-amber-300 dark:hover:bg-amber-900/50 dark:focus:ring-offset-gray-950"
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
