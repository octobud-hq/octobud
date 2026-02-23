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
	import { writable, get } from "svelte/store";
	import { tick, onMount, onDestroy } from "svelte";
	import { fly, fade } from "svelte/transition";
	import { getNotificationSettingsStore } from "$lib/stores/notificationSettings";

	const { timelineAutoScroll } = getNotificationSettingsStore();

	export let githubId: string;
	export let timelineController: TimelineController;
	export let hasPermissionError: boolean = false;
	export let timelineLastSeenAt: string | undefined = undefined;
	export let onUpdateLastSeen: ((timestamp: string) => void) | undefined = undefined;
	export let onNewActivityViewed: (() => void) | undefined = undefined;

	const { items, isLoading, error, pagination, hasAttemptedAutoLoad } = timelineController.stores;
	const { autoLoadTimeline, loadTimeline, loadMoreTimeline } = timelineController.actions;

	let hasLoaded = false;
	const isLoadingMore = writable(false);
	let autoLoadDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	// Single observer for detecting when user scrolls to the last timeline item.
	// Replaces per-item observer — we only need to know if the user reached the end.
	// State stored on a const object to avoid Svelte reactive tracking in $: blocks
	// (Svelte only tracks top-level let reassignment, not property mutations on const objects).
	const observerState = {
		observer: null as IntersectionObserver | null,
		currentElement: null as HTMLElement | null,
		setupTimer: null as ReturnType<typeof setTimeout> | null,
	};
	let observerReady = false;
	const OBSERVER_GRACE_MS = 1500;

	function setupLastItemObserver() {
		if (typeof IntersectionObserver === "undefined") return;

		observerState.observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (!entry.isIntersecting) continue;
					if (document.visibilityState !== "visible") continue;
					if (firstUnseenIndex < 0) continue;
					if (hasScrolledToUnseen) continue;

					dismissNewActivity();
				}
			},
			{ threshold: 0 }
		);
	}

	// Dismiss new activity indicator: update last-seen, mark read, hide button
	function dismissNewActivity() {
		if (hasScrolledToUnseen) return;

		hasScrolledToUnseen = true;

		// Update last-seen to the newest item's timestamp
		const currentItems = get(items);
		if (currentItems.length > 0 && onUpdateLastSeen) {
			const newestTimestamp = getItemTimestamp(currentItems[currentItems.length - 1]);
			if (newestTimestamp) {
				onUpdateLastSeen(newestTimestamp);
			}
		}

		// Mark read if notification was marked unread by new activity
		onNewActivityViewed?.();
	}

	// Auto-load timeline on mount with debounce
	onMount(() => {
		setupLastItemObserver();
		observerReady = true;

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
		if (observerState.setupTimer) {
			clearTimeout(observerState.setupTimer);
		}
		if (observerState.observer) {
			observerState.observer.disconnect();
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
	// Hide button if there's a permission error (403)
	$: showInitialButton =
		!hasPermissionError && $hasAttemptedAutoLoad && $error && !hasItems && !$isLoading;
	// Show initial loading state when we're loading and have no items yet
	$: showInitialLoading = $isLoading && !hasItems;

	// Helper to get the effective timestamp of a timeline item
	function getItemTimestamp(item: typeof $items[0]): string | undefined {
		return item.timestamp || item.createdAt || item.submittedAt || item.updatedAt;
	}

	// Calculate which items are unseen based on timelineLastSeenAt
	// Items are sorted oldest to newest in the timeline, so we find the first item
	// that is newer than the lastSeenAt timestamp
	$: firstUnseenIndex = (() => {
		if (!timelineLastSeenAt || $items.length === 0) return -1;
		const lastSeenDate = new Date(timelineLastSeenAt);
		for (let i = 0; i < $items.length; i++) {
			const itemTimestamp = getItemTimestamp($items[i]);
			if (itemTimestamp) {
				const itemDate = new Date(itemTimestamp);
				if (itemDate > lastSeenDate) {
					return i;
				}
			}
		}
		return -1; // All items have been seen
	})();

	// Sticky divider position — persists even after items are marked as seen,
	// so the divider line stays visible for the session. Only updates when
	// new unseen items arrive at a different position.
	let stickyDividerIndex = -1;

	$: if (firstUnseenIndex >= 0) {
		stickyDividerIndex = firstUnseenIndex;
	}

	// Track if user has scrolled to unseen items
	let hasScrolledToUnseen = false;
	let previousFirstUnseenIndex = -1;

	// Reset state when new unseen items arrive (e.g. live refresh)
	$: {
		if (firstUnseenIndex >= 0 && previousFirstUnseenIndex === -1) {
			hasScrolledToUnseen = false;
		}
		previousFirstUnseenIndex = firstUnseenIndex;
	}

	// Update which element the observer is watching. Extracted into a named function
	// so that Svelte's compiler cannot see internal variable references
	// (itemElementsByIndex, etc.) as dependencies of the $: block that calls it.
	// Delays .observe() by the grace period so the observer doesn't fire on mount
	// or immediately when new items arrive — it only fires after the user has had
	// time to notice the "New activity" indicator.
	function updateObserverTarget() {
		const obs = observerState.observer;
		if (!obs) return;

		// Cancel any pending delayed observe
		if (observerState.setupTimer) {
			clearTimeout(observerState.setupTimer);
			observerState.setupTimer = null;
		}

		// Unobserve previous element
		if (observerState.currentElement) {
			obs.unobserve(observerState.currentElement);
			observerState.currentElement = null;
		}

		// Delay observation by the grace period
		observerState.setupTimer = setTimeout(() => {
			observerState.setupTimer = null;
			if (hasScrolledToUnseen) return;

			const currentItems = get(items);
			const lastEl = itemElementsByIndex.get(currentItems.length - 1);
			if (lastEl) {
				obs.observe(lastEl);
				observerState.currentElement = lastEl;
			}
		}, OBSERVER_GRACE_MS);
	}

	// Reactively observe the last timeline item when there are unseen items.
	// Uses a grace period + visibility gate to avoid firing on mount or when tab is hidden.
	$: if ($items.length > 0 && firstUnseenIndex >= 0 && !hasScrolledToUnseen && observerReady) {
		tick().then(updateObserverTarget);
	}

	// Map to store element references by index
	let itemElementsByIndex: Map<number, HTMLElement> = new Map();

	// Scroll to first unseen item within the nearest scrollable ancestor.
	// Does NOT dismiss — dismissal is handled by the IntersectionObserver
	// (when the user scrolls to the last item) or by handleNewActivityClick.
	function scrollToFirstUnseen() {
		const element = firstUnseenIndex >= 0 ? itemElementsByIndex.get(firstUnseenIndex) : null;
		if (!element) return;

		const scrollContainer = findScrollContainer(element);
		if (scrollContainer) {
			const containerRect = scrollContainer.getBoundingClientRect();
			const elementRect = element.getBoundingClientRect();
			const offsetTop = elementRect.top - containerRect.top + scrollContainer.scrollTop;
			scrollContainer.scrollTo({ top: offsetTop, behavior: "smooth" });
		}
	}

	// Button click: scroll to new activity AND dismiss the indicator
	function handleNewActivityClick() {
		scrollToFirstUnseen();
		dismissNewActivity();
	}

	// Walk up the DOM to find the nearest scrollable ancestor
	function findScrollContainer(el: HTMLElement): HTMLElement | null {
		let current = el.parentElement;
		while (current) {
			const style = getComputedStyle(current);
			if (
				(style.overflowY === "auto" || style.overflowY === "scroll") &&
				current.scrollHeight > current.clientHeight
			) {
				return current;
			}
			current = current.parentElement;
		}
		return null;
	}

	// Store element reference by index
	function storeElementRef(index: number, element: HTMLElement) {
		itemElementsByIndex.set(index, element);
	}

	// Auto-scroll to first unseen on initial load if enabled.
	// scrollToFirstUnseen is a named function reference so Svelte's compiler
	// doesn't track its internal variable accesses as $: dependencies.
	$: if (
		!hasScrolledToUnseen &&
		firstUnseenIndex >= 0 &&
		$timelineAutoScroll
	) {
		tick().then(scrollToFirstUnseen);
	}

	// Show floating button when there are unseen items and user hasn't scrolled to them
	$: showNewActivityButton = firstUnseenIndex >= 0 && !hasScrolledToUnseen;

	// Svelte action for timeline items — stores element ref and handles index updates
	function observeTimelineItem(node: HTMLElement, index: number) {
		let currentIndex = index;
		storeElementRef(currentIndex, node);
		return {
			update(newIndex: number) {
				if (newIndex !== currentIndex) {
					itemElementsByIndex.delete(currentIndex);
					currentIndex = newIndex;
					storeElementRef(currentIndex, node);
				}
			},
			destroy() {
				itemElementsByIndex.delete(currentIndex);
				if (node === observerState.currentElement && observerState.observer) {
					observerState.observer.unobserve(node);
					observerState.currentElement = null;
				}
			},
		};
	}
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
			<!-- New activity divider -->
			{#if index === stickyDividerIndex && stickyDividerIndex > 0}
				<div class="new-activity-divider flex items-center gap-3 my-4">
					<div class="flex flex-col items-center flex-shrink-0 relative z-10" style="width: 40px;">
						<!-- Thread line through divider -->
						<div
							class="absolute left-1/2 -translate-x-1/2 top-0 bottom-0 w-0.5 bg-gray-300 dark:bg-gray-800 -z-10"
						></div>
						<!-- Plus circle icon -->
						<div class="h-8 w-8 rounded-full bg-blue-600 dark:bg-blue-500 ring-2 ring-white dark:ring-gray-950 flex items-center justify-center">
							<svg class="h-4 w-4 text-white" viewBox="0 0 16 16" fill="currentColor">
								<path d="M8 2a.75.75 0 0 1 .75.75v4.5h4.5a.75.75 0 0 1 0 1.5h-4.5v4.5a.75.75 0 0 1-1.5 0v-4.5h-4.5a.75.75 0 0 1 0-1.5h4.5v-4.5A.75.75 0 0 1 8 2Z"/>
							</svg>
						</div>
					</div>
					<div class="flex-1 flex items-center gap-2">
						<div class="h-px flex-1 bg-blue-500 dark:bg-blue-400"></div>
						<span
							class="text-xs font-medium text-blue-600 dark:text-blue-400 whitespace-nowrap px-2"
						>
							New activity
						</span>
						<div class="h-px flex-1 bg-blue-500 dark:bg-blue-400"></div>
					</div>
				</div>
			{/if}
			<div
				data-timestamp={getItemTimestamp(item)}
				use:observeTimelineItem={index}
			>
				<TimelineItem {item} showThread={true} isLastItem={index === lastItemIndex} />
			</div>
		{/each}
	</div>

	<!-- Floating "New Activity" button — zero-height sticky wrapper prevents layout shift on dismiss -->
	<div class="sticky bottom-8 z-30 pointer-events-none" style="height: 0;">
		{#if showNewActivityButton}
			<div
				class="absolute bottom-0 left-0 right-0 flex justify-center"
				in:fly={{ y: 12, duration: 250 }}
				out:fade={{ duration: 150 }}
			>
				<button
					type="button"
					on:click={handleNewActivityClick}
					class="pointer-events-auto px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-full shadow-lg transition-all hover:shadow-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900"
				>
					New activity
				</button>
			</div>
		{/if}
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
