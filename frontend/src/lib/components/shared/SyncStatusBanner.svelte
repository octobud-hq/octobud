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

	import { slide } from "svelte/transition";

	export let show = false;
	export let timedOut = false;
	export let onDismiss: (() => void) | null = null;
</script>

{#if show && !timedOut}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center gap-3 border-b border-indigo-200 bg-indigo-50 px-4 py-3 dark:border-indigo-800 dark:bg-indigo-950/50"
		role="status"
		aria-live="polite"
	>
		<!-- Loading spinner -->
		<svg
			class="h-5 w-5 animate-spin text-indigo-600 dark:text-indigo-400"
			xmlns="http://www.w3.org/2000/svg"
			fill="none"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
			></circle>
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
			></path>
		</svg>
		<span class="text-sm font-medium text-indigo-900 dark:text-indigo-100">
			Syncing notifications from GitHub...
		</span>
	</div>
{/if}

{#if timedOut}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-800 dark:bg-amber-950/50"
		role="status"
		aria-live="polite"
	>
		<span class="text-sm font-medium text-amber-900 dark:text-amber-100">
			Initial sync completed, but no notifications were found in your GitHub account.
		</span>
		{#if onDismiss}
			<button
				type="button"
				on:click={onDismiss}
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
		{/if}
	</div>
{/if}
