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
	import { goto } from "$app/navigation";
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
		isLoading,
		multiselectMode,
		selectAllMode,
		detailOpen,
		detailNotificationId,
		currentDetailNotification,
		currentDetail,
		detailLoading,
		detailShowingStaleData,
		detailIsRefreshing,
		keyboardFocusIndex,
		selectedIds,
		splitModeEnabled,
		listPaneWidth,
		sidebarCollapsed,
		savedListScrollPosition,
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

	// ============================================================================
	// TIMELINE CONTROLLER (PR timeline data)
	// ============================================================================

	const timelineController = createTimelineController();

	// Sync from parent data prop and reset state
	$: if (data !== lastData) {
		const viewChanged = lastData?.selectedViewSlug !== data.selectedViewSlug;
		lastData = data;
		// Sync page controller with fresh data
		pageController.actions.syncFromData(data);

		// Only clear selection when navigating to a different view, not on refresh
		if (viewChanged) {
			pageController.actions.clearSelection();
		}

		// Reset keyboard focus when navigating to a new view
		keyboardFocusIndex.set(null);
		// Reset saved scroll position only when changing views, not when opening/closing detail
		if (viewChanged) {
			pageController.stores.savedListScrollPosition.set(0);
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
	// REACTIVE STATEMENTS - URL & Navigation
	// ============================================================================

	// Reactive statement to open detail view based on URL ?id= param
	$: {
		const urlIdParam = $pageStore.url.searchParams.get("id");
		const currentDetailId = $detailNotificationId;

		// If URL has ?id= but detail is not open or different notification
		if (urlIdParam && urlIdParam !== currentDetailId) {
			let notification;
			let notificationIndex = -1;

			// Handle special markers for cross-page navigation
			if (urlIdParam === "__first__" && $pageData.items.length > 0) {
				notification = $pageData.items[0];
				notificationIndex = 0;
			} else if (urlIdParam === "__last__" && $pageData.items.length > 0) {
				notification = $pageData.items[$pageData.items.length - 1];
				notificationIndex = $pageData.items.length - 1;
			} else {
				// Find notification in current page
				// Convert both sides to strings for comparison
				const urlIdParamStr = String(urlIdParam);
				notificationIndex = $pageData.items.findIndex((n) => {
					const itemId = String(n.githubId ?? n.id);
					return itemId === urlIdParamStr;
				});
				if (notificationIndex !== -1) {
					notification = $pageData.items[notificationIndex];
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

				// Open detail without modifying URL (URL already correct or being fixed)
				pageController.actions.openDetail(actualId);

				// Sync keyboard focus index to match the selected notification
				// Use setFocusIndex (not focusNotificationAt) to avoid triggering scroll
				// The DOM focus and scroll are handled separately by click handlers or keyboard nav
				if (notificationIndex !== -1) {
					pageController.actions.setFocusIndex(notificationIndex);
				}
			} else if (urlIdParam && urlIdParam !== "__first__" && urlIdParam !== "__last__") {
				// Notification not found in current pageData - still open it by ID
				// The detail fetching logic will load it from the API
				const urlIdParamStr = String(urlIdParam);
				pageController.actions.openDetail(urlIdParamStr);
			}
		}

		// If URL has no ?id= but detail is open, close it
		if (!urlIdParam && $detailOpen) {
			pageController.actions.closeDetail();
		}
	}

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

		// Find the notification in the current page (read once, not reactive)
		const pageDataSnapshot = get(pageData);
		const notification = pageDataSnapshot.items.find(
			(n) => (n.githubId ?? n.id) === notificationId
		);

		// Start refreshing in background (no loading skeleton, just refresh spinner)
		detailIsRefreshing.set(true);
		detailShowingStaleData.set(false);

		const currentQuery = get(quickQuery);

		// Step 1: Fetch detail immediately (fast lookup)
		// This will populate the detail state with subject from the detail fetch
		// Pass the current query so the backend can calculate correct actionHints
		// If notification is in pageData, use it as fallback; otherwise fetch by ID only
		const detailPromise = fetchNotificationDetail(notificationId, {
			fallback: notification || undefined,
			query: currentQuery,
		})
			.then((detail) => {
				// Only update if this is still the notification we're viewing
				if (get(detailNotificationId) === notificationId) {
					// Update the notification in the store with actionHints from backend
					// Backend has computed correct actionHints with query context
					pageController.actions.updateNotification(detail.notification);

					// Populate detail state with subject from detail fetch
					currentDetail.set(detail);
					detailShowingStaleData.set(false);

					// SINGLE MODE: Mark as read when detail loads (fire-and-forget)
					// This happens after detail is loaded, so we can optimistically update
					const isSplitMode = get(splitModeEnabled);
					if (!isSplitMode && !detail.notification.isRead) {
						// Optimistically update the UI immediately
						const optimisticUpdate = {
							...detail.notification,
							isRead: true,
						};
						pageController.actions.updateNotification(optimisticUpdate);

						// Fire-and-forget API call
						pageController.actions.markRead(detail.notification).catch((err) => {
							console.error("Failed to mark notification as read", err);
							// Revert optimistic update on error
							pageController.actions.updateNotification(detail.notification);
						});
					}
				}
				return detail;
			})
			.catch((error) => {
				console.error("Failed to load notification detail:", error);
				// Show stale data indicator
				if (get(detailNotificationId) === notificationId) {
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
			// Pass the current query so the backend can calculate correct actionHints
			refreshNotificationSubject(githubId, { query: currentQuery })
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
					}
				})
				.catch((refreshErr) => {
					// Only show stale indicator if we have cached subject data
					if (get(detailNotificationId) === notificationId && notification?.subjectRaw) {
						detailShowingStaleData.set(true);
					}
				})
				.finally(() => {
					// Clear refreshing indicator
					if (get(detailNotificationId) === notificationId) {
						detailIsRefreshing.set(false);
					}
				});
		});
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
		if (lastProcessedNotificationId !== null) {
			lastProcessedNotificationId = null;
		}

		// Reset timeline controller to abort any ongoing loading
		// This prevents timeline loading from blocking detail closing
		timelineController.actions.reset();
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
		registerCommand("toggleSidebar", () => {
			pageController.actions.toggleSidebar();
			return true;
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
		];

		return () => {
			unregisterShortcuts();
			// Unregister all commands on cleanup
			for (const name of commandNames) {
				unregisterCommand(name);
			}
		};
	});

	onDestroy(() => {
		// Reset timeline controller when component is destroyed (e.g., navigating away)
		timelineController.actions.reset();
	});
</script>

<NotificationView
	bind:this={notificationViewComponent}
	apiError={data.apiError}
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
	isLoading={$isLoading}
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
	{timelineController}
	totalCount={$pageData.total}
	listPaneWidth={$listPaneWidth}
	onPaneResize={handlePaneResize}
	tags={data.tags ?? []}
	initialScrollPosition={$savedListScrollPosition}
/>
