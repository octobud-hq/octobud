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
	export let onDismiss: (() => void) | null = null;
	export let onRefresh: (() => void) | null = null;
</script>

{#if show}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-4 py-3 dark:border-amber-800 dark:bg-amber-950/50"
		role="alert"
		aria-live="polite"
	>
		<div class="flex items-center gap-3">
			<!-- Warning icon -->
			<svg
				class="h-5 w-5 flex-shrink-0 text-amber-600 dark:text-amber-400"
				xmlns="http://www.w3.org/2000/svg"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<path
					d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
				/>
				<line x1="12" y1="9" x2="12" y2="13" />
				<line x1="12" y1="17" x2="12.01" y2="17" />
			</svg>
			<span class="text-sm font-medium text-amber-900 dark:text-amber-100">
				Background sync appears to have stopped. Refresh the page to update notifications.
			</span>
		</div>
		<div class="flex items-center gap-2">
			{#if onRefresh}
				<button
					type="button"
					on:click={onRefresh}
					class="flex-shrink-0 rounded-md bg-amber-600 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-amber-700 focus:outline-none focus:ring-2 focus:ring-amber-600 focus:ring-offset-2 dark:bg-amber-500 dark:hover:bg-amber-600 dark:focus:ring-offset-gray-950"
				>
					Refresh
				</button>
			{/if}
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
	</div>
{/if}
