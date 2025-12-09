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
import type { Notification, NotificationPage } from "$lib/api/types";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { SelectionStore } from "../../stores/selectionStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { ControllerOptions } from "../interfaces/common";
import type { SharedHelpers } from "./sharedHelpers";

/**
 * Type for optimistic update state snapshot (for rollback)
 */
export interface OptimisticUpdateState {
	removedNotification: Notification | null;
	previousPageData: NotificationPage;
	previousTotal: number;
	previousDetailId: string | null;
	previousFocusIndex: number | null;
	wasViewingDismissedItem: boolean;
	wasSelected: boolean;
	shouldRemoveFromSelection: boolean;
}

export interface OptimisticUpdateHelpers {
	calculateNextItemAfterRemoval: (
		currentIndex: number,
		items: Notification[],
		currentFocus: number | null,
		willBeRemoved: boolean
	) => {
		targetIndex: number | null;
		targetNotification: Notification | null;
		willPageBeEmpty: boolean;
	};
	calculatePaginationStateAfterRemoval: (
		currentPage: number,
		itemsAfterRemoval: number,
		pageSize: number,
		totalAfterRemoval: number
	) => {
		shouldNavigateToPreviousPage: boolean;
		targetPage: number | null;
	};
	createOptimisticUpdateState: (
		notification: Notification,
		currentIndex: number,
		willBeRemoved: boolean,
		pageData: NotificationPage,
		detailId: string | null,
		focusIndex: number | null,
		shouldRemoveFromSelection: boolean
	) => OptimisticUpdateState;
	handlePostAction: (params: {
		notification: Notification;
		currentIndex: number;
		willBeRemoved: boolean;
		updated?: Notification;
		capturedFocusIndex?: number | null;
		shouldCloseDetailInSingleMode?: boolean;
		shouldRemoveFromSelection?: boolean;
	}) => Promise<void>;
}

interface StoreCollection {
	notificationStore: NotificationStore;
	paginationStore: PaginationStore;
	detailStore: DetailStore;
	selectionStore: SelectionStore;
	keyboardStore: KeyboardStore;
	uiStore: UIStore;
}

/**
 * Create optimistic update helpers
 */
export function createOptimisticUpdateHelpers(
	stores: StoreCollection,
	options: ControllerOptions,
	sharedHelpers: SharedHelpers
): OptimisticUpdateHelpers {
	const {
		notificationStore,
		paginationStore,
		detailStore,
		selectionStore,
		keyboardStore,
		uiStore,
	} = stores;

	/**
	 * Calculate the next item to focus/open after removing an item from the list
	 * Returns target index, target notification, and whether page will be empty
	 */
	function calculateNextItemAfterRemoval(
		currentIndex: number,
		items: Notification[],
		currentFocus: number | null,
		willBeRemoved: boolean
	): {
		targetIndex: number | null;
		targetNotification: Notification | null;
		willPageBeEmpty: boolean;
	} {
		// If item not in current page, no target
		if (currentIndex === -1) {
			return {
				targetIndex: null,
				targetNotification: null,
				willPageBeEmpty: items.length === 0,
			};
		}

		// Calculate items after removal
		const itemsAfterRemoval = items.length - (willBeRemoved ? 1 : 0);
		const willPageBeEmpty = itemsAfterRemoval === 0;

		if (willPageBeEmpty) {
			return {
				targetIndex: null,
				targetNotification: null,
				willPageBeEmpty: true,
			};
		}

		// Calculate target index: item that slides into removed item's position
		// If removing last item, focus previous item; otherwise focus item at currentIndex
		let targetIndex: number;
		if (currentIndex >= 0) {
			if (currentIndex < itemsAfterRemoval) {
				// Item slides into this position
				targetIndex = currentIndex;
			} else {
				// Removing last item, focus previous item
				targetIndex = Math.max(0, itemsAfterRemoval - 1);
			}
		} else if (currentFocus !== null && currentFocus < itemsAfterRemoval) {
			// Use current keyboard focus
			targetIndex = currentFocus;
		} else {
			// Fallback to last item if available, otherwise first
			targetIndex = itemsAfterRemoval > 0 ? itemsAfterRemoval - 1 : 0;
		}

		const targetNotification = items[targetIndex] ?? null;

		return {
			targetIndex,
			targetNotification,
			willPageBeEmpty: false,
		};
	}

	/**
	 * Calculate pagination state after removing an item
	 * Determines if we should navigate to previous page
	 */
	function calculatePaginationStateAfterRemoval(
		currentPage: number,
		itemsAfterRemoval: number,
		pageSize: number,
		totalAfterRemoval: number
	): {
		shouldNavigateToPreviousPage: boolean;
		targetPage: number | null;
	} {
		// If page is empty and we're not on page 1, navigate to previous page
		if (itemsAfterRemoval === 0 && currentPage > 1) {
			return {
				shouldNavigateToPreviousPage: true,
				targetPage: currentPage - 1,
			};
		}

		return {
			shouldNavigateToPreviousPage: false,
			targetPage: null,
		};
	}

	/**
	 * Create a snapshot of state before optimistic update for potential rollback
	 */
	function createOptimisticUpdateState(
		notification: Notification,
		currentIndex: number,
		willBeRemoved: boolean,
		pageData: NotificationPage,
		detailId: string | null,
		focusIndex: number | null,
		shouldRemoveFromSelection: boolean
	): OptimisticUpdateState {
		const key = notification.githubId ?? notification.id;
		const wasViewingDismissedItem = detailId === key;
		const wasSelected = get(selectionStore.selectedIds).has(key);

		return {
			removedNotification: willBeRemoved ? notification : null,
			previousPageData: pageData,
			previousTotal: pageData.total,
			previousDetailId: detailId,
			previousFocusIndex: focusIndex,
			wasViewingDismissedItem,
			wasSelected,
			shouldRemoveFromSelection,
		};
	}

	/**
	 * Unified post-action handler
	 * Now does optimistic UI updates before refresh for better perceived performance
	 */
	async function handlePostAction(params: {
		notification: Notification;
		currentIndex: number;
		willBeRemoved: boolean;
		updated?: Notification;
		capturedFocusIndex?: number | null;
		shouldCloseDetailInSingleMode?: boolean;
		shouldRemoveFromSelection?: boolean;
	}): Promise<void> {
		const {
			notification,
			currentIndex,
			willBeRemoved,
			updated,
			capturedFocusIndex,
			shouldCloseDetailInSingleMode = false,
			shouldRemoveFromSelection = false,
		} = params;

		const key = notification.githubId ?? notification.id;
		if (!key) return;

		// Get the most up-to-date notification from store to ensure consistency
		const notificationsById = get(notificationStore.notificationsById);
		const pageData = get(notificationStore.pageData);

		// Try to get the notification from normalized store first
		let notificationFromStore = notificationsById.get(key);

		// Fallback to pageData if not in normalized store
		if (!notificationFromStore) {
			notificationFromStore = pageData.items.find((item) => (item.githubId ?? item.id) === key);
		}

		// Use store notification if available, otherwise use passed notification
		const currentNotification = notificationFromStore ?? notification;

		// Get action hints from pageData item if available (most fresh)
		const pageItem = pageData.items.find((item) => (item.githubId ?? item.id) === key);
		const notificationWithHints = pageItem?.actionHints ? pageItem : currentNotification;
		const dismissedOn = notificationWithHints.actionHints?.dismissedOn ?? [];

		// Capture state BEFORE any optimistic updates for potential rollback
		const currentPage = get(paginationStore.page);
		const pageSize = get(paginationStore.pageSize);
		const currentTotal = get(paginationStore.total);
		const detailId = get(detailStore.detailNotificationId);
		const focusIndex = capturedFocusIndex ?? get(keyboardStore.keyboardFocusIndex);
		const isInSplitMode = get(uiStore.splitModeEnabled);
		const wasViewingDismissedItem = detailId === key;
		const affectsFocusedItem = focusIndex !== null && focusIndex === currentIndex;

		// Create snapshot for rollback using currentNotification from store
		const rollbackState = createOptimisticUpdateState(
			currentNotification,
			currentIndex,
			willBeRemoved,
			pageData,
			detailId,
			focusIndex,
			shouldRemoveFromSelection
		);

		try {
			if (willBeRemoved) {
				// DISMISSIVE ACTION - Optimistic updates before refresh

				// 1. Optimistically remove notification from store (only if in current page)
				let removedNotification: Notification | null = null;
				if (currentIndex !== -1) {
					removedNotification = notificationStore.removeNotification(key);

					// Optimistically update pagination total
					const newTotal = Math.max(0, currentTotal - 1);
					paginationStore.setTotal(newTotal);
				}

				// 2. Remove from selection if requested and selected
				if (shouldRemoveFromSelection) {
					const currentSelectedIds = get(selectionStore.selectedIds);
					if (currentSelectedIds.has(key)) {
						selectionStore.toggleSelection(key);
					}
				}

				// 3. Calculate next item and pagination state using helpers
				// Get items after optimistic removal from store
				const pageDataAfterRemoval = get(notificationStore.pageData);
				const itemsAfterRemoval = pageDataAfterRemoval.items;
				const totalAfterRemoval = get(paginationStore.total);

				const nextItemInfo = calculateNextItemAfterRemoval(
					currentIndex,
					pageData.items, // Use original items for calculation
					focusIndex,
					true // willBeRemoved
				);

				// Get the actual target notification from items after removal
				const actualTargetNotification =
					nextItemInfo.targetIndex !== null && nextItemInfo.targetIndex < itemsAfterRemoval.length
						? itemsAfterRemoval[nextItemInfo.targetIndex]
						: null;

				const paginationInfo = calculatePaginationStateAfterRemoval(
					currentPage,
					itemsAfterRemoval.length,
					pageSize,
					totalAfterRemoval
				);

				// 4. Handle pagination navigation if needed
				if (paginationInfo.shouldNavigateToPreviousPage && paginationInfo.targetPage) {
					paginationStore.setPage(paginationInfo.targetPage);
				}

				// 5. Update detail and focus optimistically
				if (wasViewingDismissedItem) {
					if (isInSplitMode && actualTargetNotification) {
						// Split mode: open the next focused item
						const targetNotificationId =
							actualTargetNotification.githubId ?? actualTargetNotification.id;
						if (targetNotificationId && nextItemInfo.targetIndex !== null) {
							keyboardStore.focusAt(nextItemInfo.targetIndex);
							detailStore.openDetail(targetNotificationId);
							await sharedHelpers.updateUrlWithDetailId(targetNotificationId);
						}
					} else {
						// Single mode or no items: close detail
						detailStore.closeDetail();
						await sharedHelpers.updateUrlWithoutDetailId();
					}
				}

				// 6. Handle focus for dismissed items
				if (affectsFocusedItem || wasViewingDismissedItem) {
					if (nextItemInfo.willPageBeEmpty) {
						if (paginationInfo.shouldNavigateToPreviousPage) {
							// Will navigate to previous page, focus will be handled after refresh
							// Just reset for now, refresh will set it
							keyboardStore.resetFocus();
						} else {
							// Page 1 is now empty
							keyboardStore.resetFocus();
						}
					} else if (nextItemInfo.targetIndex !== null) {
						// Focus the item that slid into the removed item's position
						// focusAt will clamp the index to valid range, so we don't need to check < itemsAfterRemoval.length
						keyboardStore.focusAt(nextItemInfo.targetIndex);
						// Wait for focus to be applied before refresh
						await tick();
					}
				}

				// 7. Now refresh in background to get authoritative state
				// onRefresh() invalidates all load functions including view counts, so we don't need onRefreshViewCounts()
				paginationStore.setLoading(true);
				await options.onRefresh?.();
				paginationStore.setLoading(false);
				await tick();

				// 8. After refresh, re-validate and set focus
				// Only re-set focus if we handled it optimistically (affectsFocusedItem || wasViewingDismissedItem)
				if (affectsFocusedItem || wasViewingDismissedItem) {
					const refreshedPageData = get(notificationStore.pageData);
					const refreshedItems = refreshedPageData.items;

					if (paginationInfo.shouldNavigateToPreviousPage && paginationInfo.targetPage) {
						// Navigated to previous page - focus last item
						if (refreshedItems.length > 0) {
							const lastIndex = refreshedItems.length - 1;
							keyboardStore.focusAt(lastIndex);
							// If in split mode and viewing detail, open the last item
							if (isInSplitMode && wasViewingDismissedItem) {
								const lastItem = refreshedItems[lastIndex];
								const lastItemId = lastItem.githubId ?? lastItem.id;
								if (lastItemId) {
									detailStore.openDetail(lastItemId);
									await sharedHelpers.updateUrlWithDetailId(lastItemId);
								}
							}
						}
					} else if (nextItemInfo.targetIndex !== null && refreshedItems.length > 0) {
						// Re-validate focus after refresh - ensure target index is still valid
						const validTargetIndex = Math.min(nextItemInfo.targetIndex, refreshedItems.length - 1);
						keyboardStore.focusAt(validTargetIndex);
						await tick();

						// If in split mode and we opened detail optimistically, ensure it's still open
						if (isInSplitMode && wasViewingDismissedItem && actualTargetNotification) {
							const targetNotificationId =
								actualTargetNotification.githubId ?? actualTargetNotification.id;
							// Check if the target notification is still in the refreshed list
							const stillInList = refreshedItems.some(
								(item) => (item.githubId ?? item.id) === targetNotificationId
							);
							if (stillInList && targetNotificationId) {
								// Re-open detail to ensure it's synced
								detailStore.openDetail(targetNotificationId);
							}
						}
					} else if (
						nextItemInfo.targetIndex !== null &&
						refreshedItems.length === 0 &&
						!paginationInfo.shouldNavigateToPreviousPage
					) {
						// Page became empty after refresh, reset focus
						keyboardStore.resetFocus();
					}
				}
			} else {
				// NON-DISMISSIVE ACTION
				// Update in place - notificationStore.updateNotification handles both normalized store and pageData
				if (updated) {
					notificationStore.updateNotification(updated);
					// currentDetailNotification is derived, so it updates automatically!
					// No manual sync needed

					// Check if we should close detail in single mode
					// Close BEFORE refresh to avoid showing "Loading..." flash
					if (shouldCloseDetailInSingleMode && !isInSplitMode) {
						const currentDetailId = get(detailStore.detailNotificationId);
						const currentKey = currentNotification.githubId ?? currentNotification.id;
						if (currentDetailId === currentKey) {
							// Close detail immediately (optimistic updates don't need focus restoration)
							detailStore.closeDetail();
							if (options.navigateToUrl && typeof window !== "undefined") {
								await sharedHelpers.updateUrlWithoutDetailId();
							}
							// Wait for DOM to update so list view appears before refresh
							await tick();
						}
					}

					// Refresh view counts (this is async but doesn't affect the detail view)
					await options.onRefreshViewCounts?.();

					// Restore keyboard focus if this item was focused
					if (affectsFocusedItem) {
						await tick();
						keyboardStore.focusAt(focusIndex ?? 0);
					}
				}
			}
		} catch (err) {
			// Rollback optimistic updates on error
			console.error("Error in post-action handler, rolling back optimistic updates", err);

			if (willBeRemoved && rollbackState.removedNotification) {
				// Restore removed notification
				notificationStore.restoreNotification(rollbackState.removedNotification);

				// Restore pagination total
				paginationStore.setTotal(rollbackState.previousTotal);

				// Restore page if it was changed
				const currentPageAfterError = get(paginationStore.page);
				if (currentPageAfterError !== currentPage) {
					paginationStore.setPage(currentPage);
				}

				// Restore selection if it was removed
				if (rollbackState.shouldRemoveFromSelection && rollbackState.wasSelected) {
					const currentSelectedIds = get(selectionStore.selectedIds);
					if (!currentSelectedIds.has(key)) {
						selectionStore.toggleSelection(key);
					}
				}

				// Restore detail and focus state
				if (rollbackState.previousDetailId !== get(detailStore.detailNotificationId)) {
					if (rollbackState.previousDetailId) {
						detailStore.openDetail(rollbackState.previousDetailId);
					} else {
						detailStore.closeDetail();
					}
				}

				if (rollbackState.previousFocusIndex !== null) {
					keyboardStore.focusAt(rollbackState.previousFocusIndex);
				} else {
					keyboardStore.resetFocus();
				}
			}

			// Re-throw to let handleAction show error toast
			throw err;
		}
	}

	return {
		calculateNextItemAfterRemoval,
		calculatePaginationStateAfterRemoval,
		createOptimisticUpdateState,
		handlePostAction,
	};
}
