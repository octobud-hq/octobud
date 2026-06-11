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

	// ============================================================================
	// IMPORTS
	// ============================================================================

	// Framework & Core
	import { onMount, onDestroy, tick, getContext } from "svelte";
	import { get } from "svelte/store";
	import { SvelteSet } from "svelte/reactivity";
	import { goto, afterNavigate } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page as pageStore } from "$app/stores";
	import type { PageData } from "./$types";

	// Components
	import PageHeader from "$lib/components/shared/PageHeader.svelte";
	import NotificationView from "$lib/components/notification_view/NotificationView.svelte";

	// State Controllers
	import { createTimelineController } from "$lib/state/timelineController";
	import { selectKey } from "$lib/state/notificationsPage";

	// Stores
	import { toastStore } from "$lib/stores/toastStore";

	// API & Types
	import {
		fetchNotificationDetail,
		refreshNotificationSubject,
		parseSubjectSummary,
	} from "$lib/api/notifications";
	import type { NotificationDetail } from "$lib/api/types";

	// Utilities
	import { registerListShortcuts } from "$lib/keyboard/listShortcuts";
	import { registerCommand, unregisterCommand } from "$lib/keyboard/commandRegistry";
	import type { CommandContext } from "$lib/keyboard/commandRegistry";

	// Composables
	import { debugLog } from "$lib/utils/debug";

	// ============================================================================
	// PROPS
	// ============================================================================

	export let data: PageData & {
		apiError?: string | null;
		apiErrorCode?: string | null;
	};

	// ============================================================================
	// GET CONTROLLER FROM CONTEXT
	// ============================================================================

	const pageController = getContext("notificationPageController") as ReturnType<
		typeof import("$lib/state/notificationPageController").createNotificationPageController
	>;

	// Get layout context during initialization
	const layoutContext = getContext("layoutFunctions") as any;
	const viewDialogActions = getContext("viewDialogActions") as any;

	// ============================================================================
	// COMPONENT REFS
	// ============================================================================

	let notificationViewComponent: any = null;

	// ============================================================================
	// EXTRACT STORES FROM PAGE CONTROLLER
	// ============================================================================

	const {
		views,
		selectedViewId,
		pageData,
		page,
		quickQuery,
		multiselectMode,
		selectAllMode,
		detailOpen,
		detailNotificationId,
		currentDetailNotification,
		currentDetail,
		detailLoading,
		detailShowingStaleData,
		detailIsRefreshing,
		detailHasPermissionError,
		keyboardFocusIndex,
		selectedIds,
		splitModeEnabled,
		listPaneWidth,
		sidebarCollapsed,
		savedListScrollPosition,
		tags,
	} = pageController.stores;

	const {
		builtInViewList,
		defaultViewSlug,
		defaultViewDisplayName,
		selectedView,
		selectedViewSlug,
		hasActiveFilters,
		totalPages,
		pageRangeStart,
		pageRangeEnd,
		selectionMap,
		individualSelectionDisabled,
		selectionEnabled,
	} = pageController.derived;

	// ============================================================================
	// LOCAL STATE
	// ============================================================================

	let lastData: typeof data | null = null;
	let lastProcessedNotificationId: string | null = null;

	// Track the last known sort date to detect when a notification is updated via polling.
	// When the date changes, we refresh the timeline to show new activity.
	let lastKnownSortDate: string | null = null;
	let lastKnownDetailGithubId: string | null = null;

	// Cancels the detail + refresh-subject fetches when the user switches
	// notifications or closes detail, so a slow GitHub response doesn't tie up
	// a browser connection slot and stall subsequent navigation requests.
	let detailFetchController: AbortController | null = null;

	// Debounce the refresh-subject (GitHub-backed) call so flying through
	// notifications doesn't spawn a refresh-from-GitHub per press. The cached
	// detail fetch is still immediate; only the expensive subject refresh
	// waits for the user to settle. Cleared when detail changes/closes/unmounts.
	let refreshSubjectTimer: ReturnType<typeof setTimeout> | null = null;
	const REFRESH_SUBJECT_DEBOUNCE_MS = 250;

	// Split-mode only: debounce the detail fetch so rapid j/k navigation through
	// a list (where each press already shows cache-first content) doesn't fire a
	// network call per press. Single mode and URL-loads bypass this and fetch
	// immediately because they have no cached content to show.
	let detailFetchTimer: ReturnType<typeof setTimeout> | null = null;
	const DETAIL_FETCH_DEBOUNCE_MS = 200;

	function isAbortError(err: unknown): boolean {
		return err instanceof DOMException && err.name === "AbortError";
	}

	// Helper to check if a notification type is CI activity
	function checkIsCIActivity(subjectType: string | undefined | null): boolean {
		if (!subjectType) return false;
		const type = subjectType.toLowerCase().replace(/[-_\s]/g, "");
		return type === "checkrun" || type === "checksuite" || type === "workflowrun";
	}

	// ============================================================================
	// TIMELINE CONTROLLER (PR timeline data)
	// ============================================================================

	const timelineController = createTimelineController();

	// Register a direct handler so handleSyncNewNotifications can trigger
	// timeline refreshes without relying on reactive effectiveSortDate detection.
	pageController.actions.registerTimelineRefreshHandler((githubId: string) => {
		timelineController.actions.refreshTimeline(githubId);
	});

	// Register a handler so scrollListToTop (e.g. desktop-notification click) can
	// reach into the visible list and scroll it. The pageController doesn't have
	// a ref to NotificationView itself.
	pageController.actions.registerListScrollHandler((scrollTop: number) => {
		notificationViewComponent?.setScrollPosition(scrollTop);
	});

	// Sync from parent data prop and reset state
	$: if (data !== lastData) {
		const viewChanged = lastData?.selectedViewSlug !== data.selectedViewSlug;
		const previousPageNumber = lastData?.initialPageNumber ?? lastData?.initialPage?.page ?? null;
		const nextPageNumber = data.initialPageNumber ?? data.initialPage?.page ?? null;
		const pageChanged =
			previousPageNumber !== null &&
			nextPageNumber !== null &&
			previousPageNumber !== nextPageNumber;
		lastData = data;
		// Sync page controller with fresh data
		pageController.actions.syncFromData(data);

		// Only clear selection when navigating to a different view, not on refresh
		if (viewChanged) {
			pageController.actions.clearSelection();
		}

		// Reset keyboard focus on view OR page change. Detail-only data updates
		// (e.g. rapid j/k cross-detail nav, polling refreshes, ?id-only gotos)
		// must NOT reset focus — doing so caused rapid keypresses to feel
		// "swallowed" because focus would snap back between presses. The
		// separate clearFocusIfNeeded reactive below clamps out-of-range
		// indices, so safety is preserved.
		if (viewChanged || pageChanged) {
			keyboardFocusIndex.set(null);
		}
		// Reset scroll position on view change OR page change so each new page
		// starts at the top instead of inheriting the prior page's scroll.
		// scrollListToTop() also resets savedListScrollPosition, so SingleMode
		// returns to the top when the detail is closed. Cross-page keyboard nav
		// (j/k at page edges) later calls focusAt(last/first), which scrolls
		// the focused row back into view on top of this.
		if (viewChanged || pageChanged) {
			pageController.actions.scrollListToTop();
		}

		// Execute any pending focus action from cross-page keyboard navigation
		void tick().then(() => {
			pageController.actions.executePendingFocusAction();
		});
	}

	// Clear focus when needed - depends on page store to trigger reactivity
	$: if ($pageStore) {
		pageController.actions.clearFocusIfNeeded();
	}

	// ============================================================================
	// URL → DETAIL SYNC
	// ============================================================================
	//
	// We need to react to URL state from THREE sources:
	//   1. Initial mount / full navigation (view change, pagination, goto from
	//      link / form / desktop notification): afterNavigate fires.
	//   2. Browser back/forward across SHALLOW history entries (pushState'd
	//      detail opens): afterNavigate does NOT fire here — SvelteKit's
	//      popstate handler returns early for shallow entries. But it DOES call
	//      update_url(), which notifies $pageStore. So a reactive on $pageStore
	//      catches these cases.
	//   3. Shallow pushState (in-app click / j/k detail open): neither fires.
	//      That's correct — handleOpenInlineDetail already updated the stores
	//      synchronously; no URL→store sync needed.
	//
	// Crucially, the reactive depends ONLY on $pageStore — NOT on
	// $detailNotificationId. If it depended on the latter, shallow pushState
	// (which updates detailNotificationId but leaves $page.url stale) would
	// re-fire it and reconcile against the stale URL, breaking detail open.
	//
	// The function is naturally idempotent (Svelte writables don't notify when
	// set to the same value, and openDetail/closeDetail compare to current
	// state), so we don't dedupe across signals — letting all three fire is
	// safe and self-correcting. An earlier dedupe attempt by `lastReconciledUrlId`
	// caused bugs because shallow pushState never updates the URL it tracked,
	// so its "last seen" state drifted from the actual URL/store state.

	function reconcileDetailFromUrl(urlIdParam: string | null): void {
		const currentDetailId = get(detailNotificationId);
		const itemsSnapshot = get(pageData).items;

		// URL has ?id= and store doesn't match — open the right detail
		if (urlIdParam && urlIdParam !== currentDetailId) {
			let notification;
			let notificationIndex = -1;

			if (urlIdParam === "__first__" && itemsSnapshot.length > 0) {
				notification = itemsSnapshot[0];
				notificationIndex = 0;
			} else if (urlIdParam === "__last__" && itemsSnapshot.length > 0) {
				notification = itemsSnapshot[itemsSnapshot.length - 1];
				notificationIndex = itemsSnapshot.length - 1;
			} else {
				const urlIdParamStr = String(urlIdParam);
				notificationIndex = itemsSnapshot.findIndex((n) => {
					const itemId = String(n.githubId ?? n.id);
					return itemId === urlIdParamStr;
				});
				if (notificationIndex !== -1) {
					notification = itemsSnapshot[notificationIndex];
				}
			}

			if (notification) {
				const actualId = notification.githubId ?? notification.id;

				// If we used a special marker, replace the URL with the actual ID
				if (urlIdParam === "__first__" || urlIdParam === "__last__") {
					const url = new URL(window.location.href);
					url.searchParams.set("id", actualId);
					void goto(resolve(url.pathname as any) + url.search, {
						replaceState: true,
						noScroll: true,
						keepFocus: true,
					});
				}

				pageController.actions.openDetail(actualId);

				if (notificationIndex !== -1) {
					pageController.actions.setFocusIndex(notificationIndex);
				}
			} else if (urlIdParam !== "__first__" && urlIdParam !== "__last__") {
				// Notification not on the current page — open by ID; the detail
				// fetch reactive will load it from the API.
				pageController.actions.openDetail(String(urlIdParam));
			}
			return;
		}

		// URL has no ?id= and detail is open — close it. This handles view
		// changes, pagination that strips ?id=, and browser back-to-list.
		if (!urlIdParam && get(detailOpen)) {
			pageController.actions.closeDetail();
		}
	}

	// Source 1: afterNavigate for real navigations (initial enter, goto,
	// popstate across real-nav boundaries, link, form). We read from
	// window.location rather than $pageStore because under SvelteKit 2 +
	// Svelte 5, the legacy $app/stores `page` only gets re-notified by
	// update_url() (shallow popstate); during a full navigation it's not
	// guaranteed to reflect the current URL by the time afterNavigate fires.
	// window.location is always up-to-date with the browser's URL bar.
	afterNavigate(() => {
		if (typeof window === "undefined") return;
		reconcileDetailFromUrl(new URL(window.location.href).searchParams.get("id"));
	});

	// Source 2: direct popstate listener — handles browser back/forward
	// through shallow (pushState'd) detail history. SvelteKit's popstate
	// handler returns early for shallow entries without firing afterNavigate,
	// so we need our own listener. We read window.location directly rather
	// than the $pageStore because SvelteKit's handler runs first (registered
	// in start()) so update_url has already completed by the time we read.
	//
	// We do NOT use a `$: reconcileDetailFromUrl($pageStore.url...)` reactive
	// as a third backup. Under Svelte 5 compat mode, that block re-runs more
	// eagerly than expected — e.g., when handleOpenInlineDetail updates the
	// detail store (which doesn't change $page.url because shallow pushState
	// leaves it stale). The reactive would then call reconcileDetailFromUrl
	// with a null urlIdParam against an open detail and incorrectly close it.
	onMount(() => {
		if (typeof window === "undefined") return;
		const handlePopstate = () => {
			reconcileDetailFromUrl(new URL(window.location.href).searchParams.get("id"));
		};
		window.addEventListener("popstate", handlePopstate);
		return () => window.removeEventListener("popstate", handlePopstate);
	});

	// ============================================================================
	// REACTIVE STATEMENTS - Detail Data Fetching
	// ============================================================================

	// Reactive statement to fetch detail when detailNotificationId changes
	// Only depends on detailNotificationId and detailOpen, not pageData
	$: if (
		$detailOpen &&
		$detailNotificationId &&
		lastProcessedNotificationId !== $detailNotificationId
	) {
		const notificationId = $detailNotificationId;
		lastProcessedNotificationId = notificationId;

		// Abort any in-flight fetches for the previously selected notification.
		if (detailFetchController) {
			detailFetchController.abort();
		}
		detailFetchController = new AbortController();
		const detailSignal = detailFetchController.signal;

		// Cancel any pending debounced refresh-subject call for the previously
		// selected notification — if the user navigated before it fired, we
		// don't want to fire it now against the wrong (or no longer current) id.
		if (refreshSubjectTimer !== null) {
			clearTimeout(refreshSubjectTimer);
			refreshSubjectTimer = null;
		}

		// Cancel any pending debounced detail-fetch from the previously selected
		// notification (cache-first path only). The in-flight fetch (if any) is
		// already canceled via the AbortController above; this clears the timer
		// window before the fetch has even started.
		if (detailFetchTimer !== null) {
			clearTimeout(detailFetchTimer);
			detailFetchTimer = null;
		}

		// Find the notification in the current page (read once, not reactive)
		const pageDataSnapshot = get(pageData);
		const notification = pageDataSnapshot.items.find(
			(n) => (n.githubId ?? n.id) === notificationId
		);

		const isSplitMode = get(splitModeEnabled);
		// Cache-first display: in split mode, when the notification is on the
		// current page with a parseable subjectRaw, render immediately from the
		// list cache and debounce the network fetch behind it. Single mode and
		// URL-loads (no cached entry) fall back to the loading skeleton + an
		// immediate fetch because they have nothing else to show.
		const canCacheFirst = isSplitMode && !!notification && !!notification.subjectRaw;

		if (canCacheFirst && notification) {
			currentDetail.set({
				notification,
				subject: parseSubjectSummary(notification.subjectRaw) ?? null,
			});
			detailLoading.set(false);
		} else {
			// Start loading state — no cached content to show yet
			detailLoading.set(true);
		}
		// Refresh-subject spinner shows until the GitHub-backed refresh returns
		// (or is skipped for CI activities/missing githubId).
		detailIsRefreshing.set(true);
		detailShowingStaleData.set(false);
		detailHasPermissionError.set(false);

		const currentQuery = get(quickQuery);

		if (canCacheFirst) {
			// Debounce the network fetch so rapid j/k navigation in split mode
			// doesn't spawn a fetch per press — the cached content covers the gap.
			detailFetchTimer = setTimeout(() => {
				detailFetchTimer = null;
				if (get(detailNotificationId) !== notificationId) {
					detailIsRefreshing.set(false);
					return;
				}
				// eslint-disable-next-line svelte/infinite-reactive-loop -- the timer assignments inside doFetchDetail() target refreshSubjectTimer/detailFetchTimer, neither of which is a reactive dep of this $: block (which gates on $detailOpen + $detailNotificationId + lastProcessedNotificationId).
				doFetchDetail();
			}, DETAIL_FETCH_DEBOUNCE_MS);
		} else {
			// Immediate fetch — single mode, or URL load with no cache to show.
			doFetchDetail();
		}

		function doFetchDetail() {
			// Step 1: Fetch detail (fast lookup)
			// This will populate the detail state with subject from the detail fetch
			// Pass the current query so the backend can calculate correct actionHints
			// If notification is in pageData, use it as fallback; otherwise fetch by ID only
			const detailPromise = fetchNotificationDetail(notificationId, {
				fallback: notification || undefined,
				query: currentQuery,
				signal: detailSignal,
			})
				.then((detail) => {
					// Only update if this is still the notification we're viewing
					if (get(detailNotificationId) === notificationId) {
						// Update the notification in the store with actionHints from backend
						// Backend has computed correct actionHints with query context
						pageController.actions.updateNotification(detail.notification);

						// Populate detail state with subject from detail fetch
						currentDetail.set(detail);
						detailLoading.set(false);
						detailShowingStaleData.set(false);

						// SINGLE MODE: Mark as read when detail loads
						if (!isSplitMode) {
							pageController.actions.softMarkRead(detail.notification);
						}
					}
					return detail;
				})
				.catch((error) => {
					// Aborted when user navigated away — don't show an error state.
					if (isAbortError(error)) return null;
					console.error("Failed to load notification detail:", error);
					// Show stale data indicator
					if (get(detailNotificationId) === notificationId) {
						detailLoading.set(false);
						detailShowingStaleData.set(true);
						// Fall back to cached data if we have it, otherwise show error state
						if (notification) {
							const cachedDetail: NotificationDetail = {
								notification,
								subject: parseSubjectSummary(notification.subjectRaw) ?? null,
							};
							currentDetail.set(cachedDetail);
						}
					}
					return null;
				});

			// Step 2: Kick off refresh subject call concurrently (if we have a githubId from the fetched detail)
			// Skip refresh for CI activities (workflowrun, checksuite, checkrun) as they don't have refreshable subject data
			detailPromise.then((detail) => {
				if (!detail) {
					// No detail fetched, just clear the refreshing indicator
					if (get(detailNotificationId) === notificationId) {
						detailIsRefreshing.set(false);
					}
					return;
				}
				const githubId = detail.notification.githubId;
				if (!githubId) {
					// No githubId, just clear the refreshing indicator
					if (get(detailNotificationId) === notificationId) {
						detailIsRefreshing.set(false);
					}
					return;
				}
				// Skip refresh for CI activities - they don't have refreshable subject data
				if (checkIsCIActivity(detail.notification.subjectType)) {
					// Clear refreshing indicator immediately for CI activities
					if (get(detailNotificationId) === notificationId) {
						detailIsRefreshing.set(false);
					}
					return;
				}

				// Debounce the GitHub-backed refresh so rapid detail nav (j/k or
				// clicks) doesn't fire one network call per press. Cleared at the
				// top of this reactive when detailNotificationId changes again, in
				// the close branch below, and in onDestroy.
				// eslint-disable-next-line svelte/infinite-reactive-loop -- refreshSubjectTimer is not a reactive dep of the enclosing $: block (which gates on $detailOpen + $detailNotificationId + lastProcessedNotificationId); the write also happens inside a deferred setTimeout callback.
				refreshSubjectTimer = setTimeout(() => {
					// eslint-disable-next-line svelte/infinite-reactive-loop -- same as above; clearing the handle inside the deferred callback does not affect any reactive dep.
					refreshSubjectTimer = null;
					// Guard against a final-moment store change between the timer
					// firing and the network call leaving (AbortController handles
					// in-flight cancellation past this point).
					if (get(detailNotificationId) !== notificationId) {
						detailIsRefreshing.set(false);
						return;
					}
					doRefreshSubject();
				}, REFRESH_SUBJECT_DEBOUNCE_MS);

				// The actual refresh logic lives in this nested helper so the
				// outer .then can return without losing readability.
				function doRefreshSubject() {
					// Pass the current query so the backend can calculate correct actionHints
					refreshNotificationSubject(githubId, { query: currentQuery, signal: detailSignal })
						.then((refreshedNotification) => {
							// Check if we're still viewing the same notification
							if (get(detailNotificationId) === notificationId) {
								// Update the notification - backend has computed correct actionHints with query context
								pageController.actions.updateNotification(refreshedNotification);

								// Step 3: When refresh returns, replace the subject in the detail
								// Parse the refreshed subject from the refreshed notification's subjectRaw
								const refreshedSubject = parseSubjectSummary(refreshedNotification.subjectRaw);

								// Update the current detail with the refreshed subject
								const currentDetailValue = get(currentDetail);
								if (currentDetailValue) {
									currentDetail.set({
										...currentDetailValue,
										subject: refreshedSubject,
										notification: refreshedNotification,
									});
								}
								detailShowingStaleData.set(false);
								detailHasPermissionError.set(false);
							}
						})
						.catch((refreshErr) => {
							// Aborted when user navigated away — don't surface as error.
							if (isAbortError(refreshErr)) return;
							// Check if we're still viewing the same notification
							if (get(detailNotificationId) === notificationId) {
								// Check if this is a 403 Forbidden error (permission issue)
								const errorWithStatus = refreshErr as Error & { status?: number };
								const statusCode = errorWithStatus?.status;
								const errorMessage = refreshErr?.message ?? "";
								const is403Error = statusCode === 403 || errorMessage.includes("403");

								if (is403Error) {
									detailHasPermissionError.set(true);
									detailShowingStaleData.set(false);
								} else {
									// Only show stale indicator if we have cached subject data
									if (notification?.subjectRaw) {
										detailShowingStaleData.set(true);
									}
									detailHasPermissionError.set(false);
								}
							}
						})
						.finally(() => {
							// Clear refreshing indicator
							if (get(detailNotificationId) === notificationId) {
								detailIsRefreshing.set(false);
							}
						});
				}
			});
		}
	} else if (!$detailOpen) {
		// Clear detail when closed
		// NOTE: currentDetailNotification is automatically cleared via derived store
		// NOTE: closeDetail() already clears these, but this reactive statement ensures
		// cleanup happens even if detail was closed via other means (e.g., URL change)
		// Only update if not already null to avoid unnecessary reactive updates
		if (get(currentDetail) !== null) {
			currentDetail.set(null);
		}
		if (get(detailLoading)) {
			detailLoading.set(false);
		}
		if (get(detailShowingStaleData)) {
			detailShowingStaleData.set(false);
		}
		if (get(detailIsRefreshing)) {
			detailIsRefreshing.set(false);
		}
		if (get(detailHasPermissionError)) {
			detailHasPermissionError.set(false);
		}
		if (lastProcessedNotificationId !== null) {
			lastProcessedNotificationId = null;
		}

		// Abort any in-flight detail/refresh fetches so a slow GitHub response
		// doesn't hold a browser connection slot after the detail is closed.
		if (detailFetchController) {
			detailFetchController.abort();
			detailFetchController = null;
		}

		// Cancel any pending debounced refresh-subject call.
		if (refreshSubjectTimer !== null) {
			clearTimeout(refreshSubjectTimer);
			refreshSubjectTimer = null;
		}

		// Cancel any pending debounced detail-fetch (cache-first path).
		if (detailFetchTimer !== null) {
			clearTimeout(detailFetchTimer);
			detailFetchTimer = null;
		}

		// Reset timeline controller to abort any ongoing loading
		// This prevents timeline loading from blocking detail closing
		timelineController.actions.reset();

		// Clear timeline refresh tracking
		lastKnownSortDate = null;
		lastKnownDetailGithubId = null;
	}

	// Detect when the currently-open notification is updated via polling
	// and trigger a silent timeline refresh to show new activity
	$: if ($detailOpen && $currentDetailNotification) {
		const githubId = $currentDetailNotification.githubId;
		const sortDate =
			$currentDetailNotification.effectiveSortDate ?? $currentDetailNotification.updatedAt;

		if (githubId !== lastKnownDetailGithubId) {
			// Different notification — reset tracking (initial load will handle timeline)
			lastKnownDetailGithubId = githubId ?? null;
			lastKnownSortDate = sortDate ?? null;
		} else if (lastKnownSortDate && sortDate && sortDate !== lastKnownSortDate) {
			// Same notification but sort date changed — poll detected new activity
			lastKnownSortDate = sortDate;
			if (githubId) {
				timelineController.actions.refreshTimeline(githubId);
			}
		}
	}

	$: systemViewsBySlug = new Map($views.filter((v) => v.systemView).map((v) => [v.slug, v]));

	$: inboxView = systemViewsBySlug.get("inbox");

	// ============================================================================
	// HANDLER FUNCTIONS
	// ============================================================================

	async function toggleSplitMode() {
		await pageController.actions.toggleSplitMode();
	}

	function handlePaneResize(event: CustomEvent<{ deltaX: number }>) {
		const { deltaX } = event.detail;
		const containerWidth = typeof window !== "undefined" ? window.innerWidth : 1920;
		// Calculate percentage change
		const percentChange = (deltaX / containerWidth) * 100;
		let newWidth = $listPaneWidth + percentChange;

		// Enforce min/max constraints
		newWidth = Math.max(30, Math.min(70, newWidth));
		listPaneWidth.set(newWidth);
	}

	// Handle updating current view with modified query - opens edit dialog
	function handleUpdateView() {
		const currentView = get(selectedView);
		if (!currentView) {
			toastStore.error("Cannot update view: no view selected");
			return;
		}

		const currentQuery = get(quickQuery);
		// Use view dialog controller from context (retrieved during initialization)
		if (viewDialogActions?.startEditingWithQuery) {
			viewDialogActions.startEditingWithQuery(currentView, currentQuery);
		}
	}

	// Handle saving modified query as new view
	async function handleSaveAsNewView() {
		const currentQuery = get(quickQuery);
		// Use view dialog controller from context (retrieved during initialization)
		if (viewDialogActions?.openNewDialogWithQuery) {
			viewDialogActions.openNewDialogWithQuery(currentQuery);
		}
	}

	// Helper to find scroll parent element
	function findScrollParent(element: HTMLElement): HTMLElement | null {
		let current = element.parentElement;
		while (current) {
			const overflowY = window.getComputedStyle(current).overflowY;
			if (overflowY === "auto" || overflowY === "scroll") {
				return current;
			}
			current = current.parentElement;
		}
		return null;
	}

	function focusViewSearchInput(): boolean {
		return notificationViewComponent?.focusSearchInput() ?? false;
	}

	function toggleFilterDropdown(): boolean {
		notificationViewComponent?.toggleFilterDropdown();
		return true;
	}

	function isFilterDropdownOpen(): boolean {
		return notificationViewComponent?.isFilterDropdownOpen() ?? false;
	}

	function isAnyDialogOpen(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		return layoutContext?.isAnyDialogOpen() || commandPalette?.isPaletteOpen() === true;
	}

	function toggleShortcutsModal(): boolean {
		return layoutContext?.toggleShortcutsModal() ?? false;
	}

	// Command palette handlers
	function openCommandPaletteForView(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.openWithCommand("view");
		return true;
	}

	function openCommandPaletteForSearch(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.openWithCommand("search");
		return true;
	}

	function openCommandPalettePrompt(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.openWithPrompt();
		return true;
	}

	function openCommandPaletteBulk(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.openWithCommand("bulk");
		return true;
	}

	function openCommandPaletteEmpty(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.focusInput();
		return true;
	}

	function openFocusedNotificationInGithub(): boolean {
		// If detail is open, open the detail notification in GitHub
		if ($detailOpen && $currentDetailNotification) {
			const url =
				$currentDetailNotification.htmlUrl ||
				$currentDetailNotification.subjectUrl ||
				$currentDetailNotification.githubUrl;
			if (url) {
				window.open(url, "_blank", "noopener,noreferrer");
				return true;
			}
			return false;
		}

		// Otherwise, open the focused notification in GitHub
		const notification = pageController.actions.getFocusedNotification();
		if (!notification) {
			return false;
		}
		const url = notification.htmlUrl || notification.subjectUrl || notification.githubUrl;
		if (url) {
			window.open(url, "_blank", "noopener,noreferrer");
			return true;
		}
		return false;
	}

	// Multiselect keyboard shortcut handlers
	function handleToggleFocusedSelection(): boolean {
		const notification = pageController.actions.getFocusedNotification();
		if (!notification) {
			return false;
		}
		const key = selectKey(notification);
		if (!key) {
			return false;
		}

		const newSelectedIds = new SvelteSet($selectedIds);
		if (newSelectedIds.has(key)) {
			newSelectedIds.delete(key);
		} else {
			newSelectedIds.add(key);
		}
		pageController.actions.setSelectedIds(newSelectedIds);
		return true;
	}

	function handleCycleSelectAll(): boolean {
		const currentMode = get(selectAllMode);
		if (currentMode === "none") {
			// Select all on page
			pageController.actions.selectAllPage();
		} else if (currentMode === "page") {
			// Select all across all pages
			pageController.actions.selectAllAcrossPages();
		} else {
			// Clear selection (back to none)
			pageController.actions.clearSelection();
		}
		return true;
	}

	// ============================================================================
	// LIFECYCLE
	// ============================================================================

	onMount(() => {
		// Register all commands with the command registry
		registerCommand("resetFocus", () => pageController.actions.resetKeyboardFocus());
		registerCommand("openPaletteView", () => openCommandPaletteForView());
		registerCommand("openPaletteSearch", () => openCommandPaletteForSearch());
		registerCommand("openPalettePrompt", () => openCommandPalettePrompt());
		registerCommand("openPaletteEmpty", () => openCommandPaletteEmpty());
		registerCommand("openPaletteBulk", () => openCommandPaletteBulk());
		registerCommand("focusViewSearch", () => focusViewSearchInput());
		registerCommand("toggleFilterDropdown", () => toggleFilterDropdown());
		registerCommand("toggleShortcutsModal", () => toggleShortcutsModal());
		registerCommand("focusNext", () => {
			void pageController.actions.navigateToNextNotification();
			return true;
		});
		registerCommand("focusPrevious", () => {
			void pageController.actions.navigateToPreviousNotification();
			return true;
		});
		registerCommand("focusFirst", () => {
			void pageController.actions.focusFirstNotification();
			return true;
		});
		registerCommand("focusLast", () => {
			void pageController.actions.focusLastNotification();
			return true;
		});
		registerCommand("goToNextPage", () => {
			void pageController.actions.handleGoToNextPage();
			return true;
		});
		registerCommand("goToPreviousPage", () => {
			void pageController.actions.handleGoToPreviousPage();
			return true;
		});
		registerCommand("markFocusedArchive", () =>
			pageController.actions.archiveFocusedNotification()
		);
		registerCommand("markFocusedRead", () => pageController.actions.markFocusedNotificationRead());
		registerCommand("markFocusedMute", () => pageController.actions.muteFocusedNotification());
		registerCommand("markFocusedUnfilter", () =>
			pageController.actions.unfilterFocusedNotification()
		);
		registerCommand("openFocusedInGithub", () => openFocusedNotificationInGithub());
		registerCommand("toggleFocused", () => {
			if ($detailOpen) {
				void pageController.actions.handleCloseInlineDetail();
				return true;
			}

			const notification = pageController.actions.getFocusedNotification();
			if (notification) {
				// Save scroll position before opening detail (SingleMode only)
				// Find scroll container from the focused element
				if (!$splitModeEnabled) {
					const focusedElement = document.activeElement as HTMLElement | null;
					if (focusedElement) {
						const scrollParent = findScrollParent(focusedElement);
						if (scrollParent) {
							savedListScrollPosition.set(scrollParent.scrollTop);
						}
					}
				}
				void pageController.actions.handleOpenInlineDetail(notification);
				return true;
			}
			return false;
		});
		registerCommand("navigateNextView", () => pageController.actions.navigateToNextView());
		registerCommand("navigatePreviousView", () => pageController.actions.navigateToPreviousView());
		registerCommand("toggleMultiselectMode", () => {
			pageController.actions.toggleMultiselectMode();
			return true;
		});
		registerCommand("toggleFocusedSelection", () => handleToggleFocusedSelection());
		registerCommand("cycleSelectAll", () => handleCycleSelectAll());
		registerCommand("bulkMarkRead", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkMarkRead();
			return true;
		});
		registerCommand("bulkMarkUnread", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkMarkUnread();
			return true;
		});
		registerCommand("bulkArchive", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkArchive();
			return true;
		});
		registerCommand("bulkUnarchive", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkUnarchive();
			return true;
		});
		registerCommand("bulkMute", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkMute();
			return true;
		});
		registerCommand("bulkUnmute", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkUnmute();
			return true;
		});
		registerCommand("clearSelection", () => {
			pageController.actions.clearSelection();
			return true;
		});
		registerCommand("openFocusedSnoozeDropdown", () =>
			pageController.actions.snoozeFocusedNotification()
		);
		registerCommand("closeSnoozeDropdown", () => {
			pageController.actions.closeSnoozeDropdown();
			return true;
		});
		registerCommand("openFocusedTagDropdown", () =>
			pageController.actions.tagFocusedNotification()
		);
		registerCommand("closeTagDropdown", () => {
			pageController.actions.closeTagDropdown();
			return true;
		});
		registerCommand("openBulkSnoozeDropdown", () => {
			notificationViewComponent?.openBulkSnoozeDropdown();
			return true;
		});
		registerCommand("closeBulkSnoozeDropdown", () => {
			notificationViewComponent?.closeBulkSnoozeDropdown();
			return true;
		});
		registerCommand("openBulkTagDropdown", () => {
			notificationViewComponent?.openBulkTagDropdown();
			return true;
		});
		registerCommand("closeBulkTagDropdown", () => {
			notificationViewComponent?.closeBulkTagDropdown();
			return true;
		});
		registerCommand("markFocusedStar", () => pageController.actions.starFocusedNotification());
		registerCommand("bulkStar", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkStar();
			return true;
		});
		registerCommand("bulkUnstar", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkUnstar();
			return true;
		});
		registerCommand("bulkUnfilter", () => {
			if ($selectedIds.size === 0 && get(selectAllMode) === "none") {
				return false;
			}
			void pageController.actions.bulkUnfilter();
			return true;
		});
		registerCommand("toggleSplitMode", () => {
			toggleSplitMode();
			return true;
		});
		// Custom smooth scroll: native scrollBy({behavior:"smooth"}) has a fixed ~400ms
		// duration and queues on rapid re-invocation. This animator runs ~200ms with
		// easing and cancels any in-flight animation so auto-repeat never stalls or
		// lingers after key release.
		let scrollRaf: number | null = null;
		const scrollDetailBy = (direction: 1 | -1): boolean => {
			const el = document.querySelector<HTMLElement>("[data-detail-scroll-container]");
			if (!el) return false;
			if (scrollRaf !== null) {
				cancelAnimationFrame(scrollRaf);
				scrollRaf = null;
			}
			const delta = direction * Math.round(el.clientHeight * 0.85);
			const duration = 200;
			const startTop = el.scrollTop;
			const startTime = performance.now();
			const step = (now: number) => {
				const t = Math.min(1, (now - startTime) / duration);
				const eased = 1 - (1 - t) * (1 - t); // easeOutQuad
				el.scrollTop = startTop + delta * eased;
				scrollRaf = t < 1 ? requestAnimationFrame(step) : null;
			};
			scrollRaf = requestAnimationFrame(step);
			return true;
		};
		registerCommand("scrollDetailDown", () => scrollDetailBy(1));
		registerCommand("scrollDetailUp", () => scrollDetailBy(-1));
		registerCommand("toggleSidebar", () => {
			pageController.actions.toggleSidebar();
			return true;
		});
		registerCommand("toggleHistoryDropdown", () => {
			return layoutContext?.toggleHistoryDropdown() ?? false;
		});
		registerCommand("closeHistoryDropdown", () => {
			return layoutContext?.closeHistoryDropdown() ?? false;
		});
		registerCommand("historyNavigateDown", () => {
			return layoutContext?.historyNavigateDown() ?? false;
		});
		registerCommand("historyNavigateUp", () => {
			return layoutContext?.historyNavigateUp() ?? false;
		});
		registerCommand("historyUndoFocused", () => {
			return layoutContext?.historyUndoFocused() ?? false;
		});
		registerCommand("historyOpenFocusedNotification", () => {
			return layoutContext?.historyOpenFocusedNotification() ?? false;
		});

		// Create command context with state getters
		const commandContext: CommandContext = {
			getDetailOpen: () => $detailOpen,
			getSplitViewMode: () => $splitModeEnabled,
			isAnyDialogOpen,
			isFilterDropdownOpen,
			getMultiselectActive: () => get(multiselectMode),
			getHasSelection: () => $selectedIds.size > 0 || get(selectAllMode) !== "none",
			getSnoozeDropdownOpen: () => get(pageController.stores.snoozeDropdownState) !== null,
			getTagDropdownOpen: () => get(pageController.stores.tagDropdownState) !== null,
			getBulkSnoozeDropdownOpen: () =>
				notificationViewComponent?.getBulkSnoozeDropdownOpen() ?? false,
			getBulkTagDropdownOpen: () => notificationViewComponent?.getBulkTagDropdownOpen() ?? false,
			getHistoryDropdownOpen: () => layoutContext?.isHistoryDropdownOpen() ?? false,
		};

		const unregisterShortcuts = registerListShortcuts(commandContext);

		// List of all command names to unregister on cleanup
		const commandNames = [
			"resetFocus",
			"openPaletteView",
			"openPaletteSearch",
			"openPalettePrompt",
			"openPaletteEmpty",
			"openPaletteBulk",
			"focusViewSearch",
			"toggleFilterDropdown",
			"toggleShortcutsModal",
			"toggleHistoryDropdown",
			"closeHistoryDropdown",
			"historyNavigateDown",
			"historyNavigateUp",
			"historyUndoFocused",
			"historyOpenFocusedNotification",
			"focusNext",
			"focusPrevious",
			"focusFirst",
			"focusLast",
			"goToNextPage",
			"goToPreviousPage",
			"markFocusedArchive",
			"markFocusedRead",
			"markFocusedMute",
			"markFocusedUnfilter",
			"openFocusedInGithub",
			"toggleFocused",
			"navigateNextView",
			"navigatePreviousView",
			"toggleMultiselectMode",
			"toggleFocusedSelection",
			"cycleSelectAll",
			"bulkMarkRead",
			"bulkMarkUnread",
			"bulkArchive",
			"bulkUnarchive",
			"bulkMute",
			"bulkUnmute",
			"clearSelection",
			"openFocusedSnoozeDropdown",
			"closeSnoozeDropdown",
			"openFocusedTagDropdown",
			"closeTagDropdown",
			"openBulkSnoozeDropdown",
			"closeBulkSnoozeDropdown",
			"openBulkTagDropdown",
			"closeBulkTagDropdown",
			"markFocusedStar",
			"bulkStar",
			"bulkUnstar",
			"bulkUnfilter",
			"toggleSplitMode",
			"toggleSidebar",
			"scrollDetailDown",
			"scrollDetailUp",
		];

		return () => {
			unregisterShortcuts();
			// Cancel any in-flight detail scroll animation so it doesn't keep
			// running against a detached element after unmount.
			if (scrollRaf !== null) {
				cancelAnimationFrame(scrollRaf);
				scrollRaf = null;
			}
			// Unregister all commands on cleanup
			for (const name of commandNames) {
				unregisterCommand(name);
			}
		};
	});

	onDestroy(() => {
		// Abort any in-flight detail/refresh fetches when navigating away to
		// settings or another route, so they don't hold connection slots.
		if (detailFetchController) {
			detailFetchController.abort();
			detailFetchController = null;
		}
		// Cancel any pending debounced refresh-subject call.
		if (refreshSubjectTimer !== null) {
			clearTimeout(refreshSubjectTimer);
			refreshSubjectTimer = null;
		}
		// Cancel any pending debounced detail-fetch (cache-first path).
		if (detailFetchTimer !== null) {
			clearTimeout(detailFetchTimer);
			detailFetchTimer = null;
		}
		// Reset timeline controller when component is destroyed (e.g., navigating away)
		timelineController.actions.reset();
	});
</script>

<NotificationView
	bind:this={notificationViewComponent}
	apiError={data.apiError}
	apiErrorCode={data.apiErrorCode}
	apiErrorIsInline={data.apiErrorIsInline ?? false}
	combinedQuery={$quickQuery}
	canUpdateView={!!$selectedView}
	onChangeQuery={(newQuery) => quickQuery.set(newQuery)}
	onSubmitQuery={pageController.actions.handleQuickQueryChange}
	onClear={pageController.actions.clearFilters}
	onUpdateView={handleUpdateView}
	onSaveAsNewView={handleSaveAsNewView}
	page={$page}
	totalPages={$totalPages}
	onPrevious={() => pageController.actions.handleGoToPage($page - 1)}
	onNext={() => pageController.actions.handleGoToPage($page + 1)}
	multiselectMode={$multiselectMode}
	onToggleMultiselect={pageController.actions.toggleMultiselectMode}
	splitModeEnabled={$splitModeEnabled}
	onToggleSplitMode={toggleSplitMode}
	hasActiveFilters={$hasActiveFilters}
	pageRangeStart={$pageRangeStart}
	pageRangeEnd={$pageRangeEnd}
	items={$pageData.items}
	selectionEnabled={$selectionEnabled}
	individualSelectionDisabled={$individualSelectionDisabled}
	detailNotificationId={$detailNotificationId}
	selectionMap={$selectionMap}
	detailOpen={$detailOpen}
	currentDetailNotification={$currentDetailNotification}
	currentDetail={$currentDetail}
	detailLoading={$detailLoading}
	detailShowingStaleData={$detailShowingStaleData}
	detailIsRefreshing={$detailIsRefreshing}
	hasPermissionError={$detailHasPermissionError}
	{timelineController}
	totalCount={$pageData.total}
	listPaneWidth={$listPaneWidth}
	onPaneResize={handlePaneResize}
	tags={$tags}
	initialScrollPosition={$savedListScrollPosition}
/>
