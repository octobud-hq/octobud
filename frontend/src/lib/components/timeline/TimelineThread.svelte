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

	import type { TimelineController } from "$lib/state/timelineController";
	import TimelineItem from "./TimelineItem.svelte";
	import { writable } from "svelte/store";
	import { tick, onMount, onDestroy } from "svelte";

	export let githubId: string;
	export let timelineController: TimelineController;

	const { items, isLoading, error, pagination, hasAttemptedAutoLoad } = timelineController.stores;
	const { autoLoadTimeline, loadTimeline, loadMoreTimeline } = timelineController.actions;

	let hasLoaded = false;
	const isLoadingMore = writable(false);
	let autoLoadDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	// Auto-load timeline on mount with debounce
	onMount(() => {
		// Debounce auto-load to avoid rapid-fire requests during navigation
		autoLoadDebounceTimer = setTimeout(() => {
			if (!$hasAttemptedAutoLoad && githubId) {
				autoLoadTimeline(githubId, 10);
			}
		}, 300);
	});

	onDestroy(() => {
		if (autoLoadDebounceTimer) {
			clearTimeout(autoLoadDebounceTimer);
		}
	});

	async function handleLoadTimeline() {
		hasLoaded = true;
		isLoadingMore.set(true);
		try {
			await loadTimeline(githubId, 10);
			await tick();
		} finally {
			isLoadingMore.set(false);
		}
	}

	async function handleLoadMore() {
		isLoadingMore.set(true);
		try {
			await loadMoreTimeline(githubId, 10);
			await tick();
		} finally {
			isLoadingMore.set(false);
		}
	}

	function handleRetry() {
		if (hasLoaded) {
			loadMoreTimeline(githubId, 10);
		} else {
			handleLoadTimeline();
		}
	}

	$: hasItems = $items.length > 0;
	$: hasMoreToLoad = $pagination.hasMore;
	$: lastItemIndex = $items.length - 1;

	// Generate unique keys for timeline items (some events may have null IDs)
	$: itemsWithKeys = $items.map((item, index) => ({
		item,
		key: item.id != null ? String(item.id) : `${item.type}-${item.timestamp || ""}-${index}`,
	}));
	// Show button if there's more to load OR if we're currently loading
	// Use derived logic to prevent button from disappearing during load
	$: showLoadMoreButton = $isLoadingMore || hasMoreToLoad;
	// Only show initial button if auto-load failed and we haven't loaded manually
	$: showInitialButton = $hasAttemptedAutoLoad && $error && !hasItems && !$isLoading;
	// Show initial loading state when we're loading and have no items yet
	$: showInitialLoading = $isLoading && !hasItems;
</script>

{#if showInitialLoading}
	<!-- Initial loading state (shown while auto-loading) -->
	<div class="relative">
		<div class="flex gap-3 pt-4">
			<!-- Empty spacer for avatar column (no thread line) -->
			<div class="flex-shrink-0 relative z-10" style="width: 40px;"></div>

			<div class="flex-1">
				<button
					type="button"
					class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-100/50 dark:bg-gray-800/50 px-4 py-3 text-sm font-medium text-gray-900 dark:text-gray-200 transition disabled:opacity-50 disabled:cursor-not-allowed"
					disabled={true}
				>
					<span class="flex items-center justify-center gap-2">
						<svg
							class="h-4 w-4 animate-spin text-gray-600 dark:text-gray-400"
							viewBox="0 0 24 24"
							fill="none"
							xmlns="http://www.w3.org/2000/svg"
						>
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="3"
							/>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							/>
						</svg>
						<span>Loading activity...</span>
					</span>
				</button>
			</div>
		</div>
	</div>
{:else if showInitialButton}
	<!-- Initial load button (only shown if auto-load failed) -->
	<div class="relative">
		<div class="flex gap-3 pt-4">
			<!-- Empty spacer for avatar column (no thread line) -->
			<div class="flex-shrink-0 relative z-10" style="width: 40px;"></div>

			<div class="flex-1">
				<button
					type="button"
					class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-800/50 px-4 py-3 text-sm font-medium text-gray-900 dark:text-gray-200 transition hover:bg-gray-200 dark:hover:bg-gray-800 hover:border-gray-400 dark:hover:border-gray-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 disabled:opacity-50 disabled:cursor-not-allowed"
					on:click={handleLoadTimeline}
					disabled={$isLoadingMore}
				>
					{#if $isLoadingMore}
						<span class="flex items-center justify-center gap-2">
							<svg
								class="h-4 w-4 animate-spin text-gray-600 dark:text-gray-400"
								viewBox="0 0 24 24"
								fill="none"
								xmlns="http://www.w3.org/2000/svg"
							>
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="3"
								/>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								/>
							</svg>
							<span>Loading activity...</span>
						</span>
					{:else}
						Load activity
					{/if}
				</button>
			</div>
		</div>
	</div>
{:else if $error && !hasItems}
	<!-- Error state -->
	<div class="relative">
		<div class="flex gap-3 pt-4">
			<!-- Empty spacer for avatar column (no thread line) -->
			<div class="flex-shrink-0 relative z-10" style="width: 40px;"></div>

			<div class="flex-1 rounded-lg border border-red-800 bg-red-900/20 px-4 py-3">
				<div class="flex items-start gap-3">
					<svg
						class="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5"
						viewBox="0 0 24 24"
						fill="none"
						xmlns="http://www.w3.org/2000/svg"
					>
						<path
							d="M12 9V13M12 17H12.01M10.29 3.86L1.82 18C1.64537 18.3024 1.55299 18.6453 1.55201 18.9945C1.55103 19.3437 1.64149 19.6871 1.81442 19.9905C1.98735 20.2939 2.23672 20.5467 2.53771 20.7239C2.83869 20.901 3.18072 20.9962 3.53 21H20.47C20.8193 20.9962 21.1613 20.901 21.4623 20.7239C21.7633 20.5467 22.0127 20.2939 22.1856 19.9905C22.3585 19.6871 22.449 19.3437 22.448 18.9945C22.447 18.6453 22.3546 18.3024 22.18 18L13.71 3.86C13.5317 3.56611 13.2807 3.32312 12.9812 3.15448C12.6817 2.98585 12.3437 2.89725 12 2.89725C11.6563 2.89725 11.3183 2.98585 11.0188 3.15448C10.7193 3.32312 10.4683 3.56611 10.29 3.86Z"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
					<div class="flex-1">
						<p class="text-sm font-medium text-red-200">Failed to load activity</p>
						<p class="mt-1 text-sm text-red-300">{$error}</p>
						<button
							type="button"
							class="mt-2 text-sm font-medium text-red-300 hover:text-red-200 underline"
							on:click={handleRetry}
						>
							Try again
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
{:else if hasItems}
	<!-- Timeline thread with continuous line -->
	<div class="relative">
		<!-- Load more button (at top, for older activity) -->
		{#if showLoadMoreButton}
			<div class="flex gap-3 pt-4 mb-4">
				<!-- Avatar column with thread line -->
				<div class="flex-shrink-0 relative z-10" style="width: 40px;">
					<!-- Thread line through load more button -->
					<div
						class="absolute left-1/2 -translate-x-1/2 top-0 bottom-0 w-0.5 bg-gray-300 dark:bg-gray-800 -z-10"
					></div>
				</div>

				<div class="flex-1">
					<button
						type="button"
						class="w-full rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-100/50 dark:bg-gray-800/30 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-800/50 hover:border-gray-400 dark:hover:border-gray-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 disabled:opacity-50 disabled:cursor-not-allowed"
						on:click={handleLoadMore}
						disabled={$isLoadingMore}
					>
						<span class="flex items-center justify-center gap-2">
							{#if $isLoadingMore}
								<svg
									class="h-4 w-4 animate-spin text-gray-600 dark:text-gray-400"
									viewBox="0 0 24 24"
									fill="none"
									xmlns="http://www.w3.org/2000/svg"
								>
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="3"
									/>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									/>
								</svg>
								<span>Loading activity...</span>
							{:else}
								<span>Load previous activity</span>
							{/if}
						</span>
					</button>
				</div>
			</div>
		{/if}

		<!-- Timeline items list -->
		{#each itemsWithKeys as { item, key }, index (key)}
			<TimelineItem {item} showThread={true} isLastItem={index === lastItemIndex} />
		{/each}
	</div>
{:else if !$isLoading && $hasAttemptedAutoLoad}
	<!-- No activity state -->
	<div class="relative">
		<div class="flex gap-3 pt-4">
			<!-- Empty spacer for avatar column (no thread line) -->
			<div class="flex-shrink-0 relative z-10" style="width: 40px;"></div>

			<div
				class="flex-1 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-800/30 px-4 py-3"
			>
				<p class="text-sm text-gray-600 dark:text-gray-400 text-center">No activity yet.</p>
			</div>
		</div>
	</div>
{/if}
