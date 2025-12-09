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

	import NotificationRow from "./NotificationRow.svelte";
	import type { Notification } from "$lib/api/types";

	import { onMount, onDestroy } from "svelte";

	export let hasActiveFilters: boolean;
	export let totalCount: number;
	export let pageRangeStart: number;
	export let pageRangeEnd: number;

	export let items: Notification[];
	export let isLoading: boolean;
	export let selectionEnabled: boolean;
	export let individualSelectionDisabled: boolean = false;
	export let detailNotificationId: string | null = null; // ID of notification currently open in detail view
	export let isSplitView: boolean = false; // Whether we're in split view mode
	export let selectionMap: Map<string, boolean>;
	export let initialScrollPosition: number = 0; // Scroll position to restore on mount
	export let apiError: string | null = null; // Inline error message (for query validation errors)

	// Scroll position management
	let scrollContainer: HTMLDivElement | null = null;

	// Restore scroll position when component mounts
	onMount(() => {
		if (initialScrollPosition > 0 && scrollContainer) {
			scrollContainer.scrollTop = initialScrollPosition;
		}
	});

	// Export methods for parent to control scroll position
	export function getScrollPosition(): number {
		return scrollContainer ? scrollContainer.scrollTop : 0;
	}

	export function setScrollPosition(scrollTop: number) {
		if (scrollContainer) {
			scrollContainer.scrollTop = scrollTop;
		}
	}

	export function saveScrollPosition() {
		// This is now just an alias for getScrollPosition for backward compatibility
		return getScrollPosition();
	}

	export function restoreScrollPosition() {
		// Deprecated - parent should use setScrollPosition instead
	}

	export function resetScrollPosition() {
		setScrollPosition(0);
	}

	let delayedLoading = false;
	let loadingDelayTimer: number | null = null;

	// Delay showing loading state by 200ms to avoid jank on quick loads
	const LOADING_DELAY_MS = 200;

	$: {
		if (isLoading) {
			// Start a timer to show loading state after delay
			if (loadingDelayTimer === null) {
				loadingDelayTimer = window.setTimeout(() => {
					delayedLoading = true;
					loadingDelayTimer = null;
				}, LOADING_DELAY_MS);
			}
		} else {
			// Clear the timer if loading completes before delay
			if (loadingDelayTimer !== null) {
				window.clearTimeout(loadingDelayTimer);
				loadingDelayTimer = null;
			}
			delayedLoading = false;
		}
	}

	onDestroy(() => {
		if (loadingDelayTimer !== null) {
			window.clearTimeout(loadingDelayTimer);
		}
	});
</script>

<section
	class="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden pl-4 pr-2 bg-transparent dark:bg-transparent {isSplitView
		? 'border-t border-gray-200 dark:border-gray-800 pt-4'
		: ''}"
>
	<div
		bind:this={scrollContainer}
		class="flex-1 overflow-y-auto pr-3"
		style="scrollbar-gutter: stable; overflow-anchor: none;"
	>
		{#if apiError}
			<!-- Inline error message (for query validation errors) -->
			<div class="mb-2 pl-2">
				<div
					class="rounded-lg border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/20 p-4"
				>
					<div class="flex items-start gap-3">
						<svg
							class="h-5 w-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
						<div class="flex-1">
							<p class="text-sm font-semibold text-amber-800 dark:text-amber-200">Invalid Query</p>
							<p class="mt-1 text-sm text-amber-700 dark:text-amber-300">{apiError}</p>
						</div>
					</div>
				</div>
			</div>
		{/if}

		{#if items.length === 0 && !delayedLoading && !apiError}
			<!-- Empty state with vertical centering -->
			<div class="flex flex-col h-full">
				<!-- Page range indicator -->
				<div class="mb-2 pl-2 flex items-center justify-between flex-shrink-0">
					<span class="text-xs text-gray-600 dark:text-gray-500">
						{totalCount}
						{totalCount === 1 ? "notification" : "notifications"}
						{#if hasActiveFilters}
							(filtered)
						{/if}
					</span>
					<span class="text-xs text-gray-600 dark:text-gray-500"> 0 of 0 </span>
				</div>
				<!-- Centered empty state message -->
				<div class="flex flex-1 items-center justify-center">
					<div class="flex flex-col items-center justify-center gap-2 text-center">
						<p class="text-lg font-semibold text-gray-700 dark:text-gray-200">Nothing here yet</p>
						<p class="max-w-sm text-sm text-gray-600 dark:text-gray-400">
							Try adjusting the filters or pick a different view to see more notifications.
						</p>
					</div>
				</div>
			</div>
		{:else if !apiError}
			<!-- Page range indicator (scrolls with content) -->
			<div class="mb-2 pl-2 flex items-center justify-between">
				<span class="text-xs text-gray-500">
					{totalCount}
					{totalCount === 1 ? "notification" : "notifications"}
					{#if hasActiveFilters}
						(filtered)
					{/if}
				</span>
				<span class="text-xs text-gray-500">
					{#if totalCount === 0}
						0 of 0
					{:else if pageRangeStart === 0}
						0 of {totalCount}
					{:else}
						{pageRangeStart}-{pageRangeEnd} of {totalCount}
					{/if}
				</span>
			</div>

			<div class="space-y-2.5 pb-4">
				{#if delayedLoading}
					<div class="space-y-2.5">
						{#each Array(3) as _, i (i)}
							<div
								class="h-24 animate-pulse rounded-2xl bg-gray-200 dark:bg-gray-800/60"
								aria-hidden="true"
							></div>
						{/each}
					</div>
				{:else}
					{#each items as notification, index (notification.id)}
						{@const notificationKey = notification.githubId || notification.id}
						{@const selected =
							selectionMap.has(notificationKey) && selectionMap.get(notificationKey) === true}
						{@const isDetailOpen =
							detailNotificationId !== null &&
							detailNotificationId === (notification.githubId ?? notification.id)}
						<NotificationRow
							{notification}
							{selectionEnabled}
							selectionDisabled={individualSelectionDisabled}
							{selected}
							{isDetailOpen}
						/>
					{/each}
				{/if}
			</div>
		{/if}
	</div>
</section>
