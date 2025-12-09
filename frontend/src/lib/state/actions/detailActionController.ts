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

import { get } from "svelte/store";
import { tick } from "svelte";
import type { Notification } from "$lib/api/types";
import type { DetailActions } from "../interfaces/detailActions";
import type { NavigationState, NavigateOptions, ControllerOptions } from "../interfaces/common";
import type { DetailStore } from "../../stores/detailViewStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { SharedHelpers } from "./sharedHelpers";

/**
 * Detail Action Controller
 * Manages detail view state and navigation
 */
export function createDetailActionController(
	detailStore: DetailStore,
	keyboardStore: KeyboardStore,
	notificationStore: NotificationStore,
	paginationStore: PaginationStore,
	uiStore: UIStore,
	options: ControllerOptions,
	sharedHelpers: SharedHelpers,
	markReadAction: (notification: Notification) => Promise<void>
): DetailActions {
	// Helper to get current UI state for history preservation
	function getCurrentNavState(): NavigationState {
		return {
			sidebarCollapsed: get(uiStore.sidebarCollapsed),
			splitModeEnabled: get(uiStore.splitModeEnabled),
			detailNotificationId: get(detailStore.detailNotificationId),
			savedScrollPosition: get(uiStore.savedListScrollPosition),
			keyboardFocusIndex: get(keyboardStore.keyboardFocusIndex),
		};
	}

	function openDetail(notificationId: string): void {
		// Index is now derived automatically from detailNotificationId + pageData
		detailStore.openDetail(notificationId);
	}

	function closeDetail(): void {
		detailStore.closeDetail();
	}

	async function handleOpenInlineDetail(notification: Notification): Promise<void> {
		const notificationId = notification.githubId ?? notification.id;
		const isSplitModeMode = get(uiStore.splitModeEnabled);

		// Find the index of this notification in the current page
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;
		const notificationIndex = items.findIndex((n) => (n.githubId ?? n.id) === notificationId);

		// Set keyboard focus index (but don't trigger DOM focus - the browser handles that for clicks)
		if (notificationIndex !== -1) {
			keyboardStore.setFocusIndex(notificationIndex);
		}

		// Open detail - index is now derived automatically
		detailStore.openDetail(notificationId);

		// Wait for reactive updates to settle
		await tick();

		// Update URL with ?id= param for deep linking
		// SingleMode: push new history entry so back button works
		// SplitMode: replace state to avoid cluttering history
		if (typeof window !== "undefined" && options.navigateToUrl) {
			const url = new URL(window.location.href);
			url.searchParams.set("id", notificationId);
			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: isSplitModeMode, // Push for SingleMode, replace for SplitMode
			});
		} else {
			// Fallback if navigateToUrl not available (e.g., tests)
			openDetail(notificationId);
		}
	}

	async function handleCloseInlineDetail(opts: { clearFocus?: boolean } = {}): Promise<void> {
		const closingNotificationId = get(detailStore.detailNotificationId);
		const closingNotificationIndex = get(detailStore.detailNotificationIndex); // Now derived
		const isSplitMode = get(uiStore.splitModeEnabled);
		const originalIndex = closingNotificationIndex;

		// Update URL first to prevent reactive statement from reopening detail
		// SingleMode: push new history entry so back button works
		// SplitMode: replace state to avoid cluttering history
		if (typeof window !== "undefined" && options.navigateToUrl) {
			try {
				const url = new URL(window.location.href);
				url.searchParams.delete("id");
				await options.navigateToUrl(url.pathname + url.search, {
					state: getCurrentNavState(),
					replace: isSplitMode,
				});
			} catch (err) {
				console.error("Failed to update URL on detail close", err);
				// Continue with close even if URL update fails
			}
		}

		// Now close detail after URL is updated (or if no navigateToUrl available)
		detailStore.closeDetail();

		// Wait for DOM to update
		await tick();

		// If explicitly requested to clear focus (e.g., user clicked back button with mouse),
		// don't restore keyboard focus - they're using mouse navigation
		if (opts.clearFocus) {
			keyboardStore.resetFocus();
			return;
		}

		// Restore keyboard focus to the notification that was just viewed
		if (closingNotificationId) {
			const pageData = get(notificationStore.pageData);
			const items = pageData.items;
			const isSplitMode = get(uiStore.splitModeEnabled);

			// Try to find the notification at the original index first (handles duplicates)
			let notificationIndex = -1;
			if (originalIndex !== null && originalIndex >= 0 && originalIndex < items.length) {
				const notifAtOriginalIndex = items[originalIndex];
				if (
					notifAtOriginalIndex &&
					(notifAtOriginalIndex.githubId ?? notifAtOriginalIndex.id) === closingNotificationId
				) {
					notificationIndex = originalIndex;
				}
			}

			// If not at original index, search for it (list may have been reordered)
			if (notificationIndex === -1) {
				notificationIndex = items.findIndex((n) => (n.githubId ?? n.id) === closingNotificationId);
			}

			if (notificationIndex !== -1) {
				// In SingleMode, just set the focus index without scrolling
				// (scroll position is restored separately from savedListScrollPosition)
				// In SplitMode, use focusAt to also scroll to the item
				if (isSplitMode) {
					keyboardStore.focusAt(notificationIndex);
				} else {
					keyboardStore.setFocusIndex(notificationIndex);
				}
			} else {
				// Notification was dismissed - handle accordingly
				const currentFocusIndex = get(keyboardStore.keyboardFocusIndex);
				if (currentFocusIndex !== null && items.length > 0) {
					const safeIndex = Math.min(currentFocusIndex, items.length - 1);
					const clampedIndex = Math.max(0, safeIndex);

					if (clampedIndex !== currentFocusIndex) {
						if (isSplitMode) {
							keyboardStore.focusAt(clampedIndex);
						} else {
							keyboardStore.setFocusIndex(clampedIndex);
						}
					}
				} else if (items.length > 0) {
					if (isSplitMode) {
						keyboardStore.focusAt(0);
					} else {
						keyboardStore.setFocusIndex(0);
					}
				}
			}
		}
	}

	async function navigateToNextNotification(): Promise<void> {
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;
		const currentPage = get(paginationStore.page);
		const totalPagesCount = get(paginationStore.totalPages);
		const isDetailViewOpen = get(detailStore.detailOpen);
		const isSplitMode = get(uiStore.splitModeEnabled);
		const currentDetailId = get(detailStore.detailNotificationId);

		// Find current index based on open detail, fallback to keyboard focus
		let currentIndex = get(keyboardStore.keyboardFocusIndex) ?? -1;

		// If detail is open, use that to determine current position
		if (currentDetailId && currentIndex === -1) {
			currentIndex = items.findIndex((n) => (n.githubId ?? n.id) === currentDetailId);
		}

		// If we can move within the page
		if (currentIndex < items.length - 1) {
			const nextIndex = currentIndex + 1;
			keyboardStore.focusAt(nextIndex);

			if (isSplitMode || isDetailViewOpen) {
				const notification = items[nextIndex];
				if (notification) {
					await handleOpenInlineDetail(notification);
				}
			}
			return;
		}

		// At end of page - go to next page
		// Pagination always creates a new history entry
		if (currentPage < totalPagesCount && options.navigateToUrl) {
			const url = new URL(window.location.href);
			url.searchParams.set("page", (currentPage + 1).toString());
			keyboardStore.setPendingFocusAction("first");
			if (isSplitMode || isDetailViewOpen) {
				url.searchParams.set("id", "__first__");
			}
			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false, // Always push for pagination
			});
		}
	}

	async function navigateToPreviousNotification(): Promise<void> {
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;
		const currentPage = get(paginationStore.page);
		const isDetailViewOpen = get(detailStore.detailOpen);
		const isSplitMode = get(uiStore.splitModeEnabled);
		const currentDetailId = get(detailStore.detailNotificationId);

		// Find current index based on open detail, fallback to keyboard focus
		let currentIndex = get(keyboardStore.keyboardFocusIndex) ?? -1;

		// If detail is open, use that to determine current position
		if (currentDetailId && currentIndex === -1) {
			currentIndex = items.findIndex((n) => (n.githubId ?? n.id) === currentDetailId);
		}

		// If we can move within the page
		if (currentIndex > 0) {
			const prevIndex = currentIndex - 1;
			keyboardStore.focusAt(prevIndex);

			if (isSplitMode || isDetailViewOpen) {
				const notification = items[prevIndex];
				if (notification) {
					await handleOpenInlineDetail(notification);
				}
			}
			return;
		}

		// At start of page - go to previous page
		// Pagination always creates a new history entry
		if (currentPage > 1 && options.navigateToUrl) {
			const url = new URL(window.location.href);
			const prevPage = currentPage - 1;
			if (prevPage > 1) {
				url.searchParams.set("page", prevPage.toString());
			} else {
				url.searchParams.delete("page");
			}
			keyboardStore.setPendingFocusAction("last");
			if (isSplitMode || isDetailViewOpen) {
				url.searchParams.set("id", "__last__");
			}
			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false, // Always push for pagination
			});
		}
	}

	async function toggleSplitMode(): Promise<void> {
		const wasInSplitMode = get(uiStore.splitModeEnabled);

		// Toggle the mode first so getCurrentNavState captures the NEW state
		uiStore.toggleSplitMode();

		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);

			// If we were in split mode (now turning it off), clear detail
			if (wasInSplitMode) {
				url.searchParams.delete("id");
				closeDetail();
			}

			// Update history with new mode state (replace current entry)
			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(), // Captures the NEW state after toggle
				replace: true,
			});
		} else if (wasInSplitMode) {
			// Fallback if navigateToUrl not available
			closeDetail();
		}
	}

	function handlePaneResize(deltaX: number): void {
		const containerWidth = typeof window !== "undefined" ? window.innerWidth : 1920;
		const percentChange = (deltaX / containerWidth) * 100;
		const currentWidth = get(uiStore.listPaneWidth);
		let newWidth = currentWidth + percentChange;
		uiStore.setListPaneWidth(newWidth);
	}

	return {
		openDetail,
		closeDetail,
		handleOpenInlineDetail,
		handleCloseInlineDetail,
		navigateToNextNotification,
		navigateToPreviousNotification,
		toggleSplitMode,
		handlePaneResize,
	};
}
