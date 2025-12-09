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
import type { KeyboardActions } from "../interfaces/keyboardActions";
import type { NavigationState, ControllerOptions } from "../interfaces/common";
import type { PaginationActions } from "../interfaces/paginationActions";
import type { DetailActions } from "../interfaces/detailActions";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { DetailStore } from "../../stores/detailViewStore";

/**
 * Keyboard Action Controller
 * Manages keyboard focus and navigation
 */
export function createKeyboardActionController(
	keyboardStore: KeyboardStore,
	notificationStore: NotificationStore,
	paginationStore: PaginationStore,
	uiStore: UIStore,
	detailStore: DetailStore,
	options: ControllerOptions,
	paginationActions: Pick<PaginationActions, "handleGoToNextPage" | "handleGoToPreviousPage">,
	detailActions: Pick<DetailActions, "handleOpenInlineDetail">
): KeyboardActions {
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

	function focusNotificationAt(index: number, _options?: { scroll?: boolean }): boolean {
		// Always use focusAt which triggers DOM focus via pendingFocusIndex
		return keyboardStore.focusAt(index);
	}

	function setFocusIndex(index: number): boolean {
		// Only updates visual focus index, does NOT trigger DOM focus
		return keyboardStore.setFocusIndex(index);
	}

	function getFocusedNotification() {
		const focused = get(keyboardStore.focusedNotification);
		return focused;
	}

	async function focusNextNotification(): Promise<boolean> {
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;

		if (items.length === 0) {
			return false;
		}

		const currentFocusIndex = get(keyboardStore.keyboardFocusIndex);
		let nextIndex: number;

		if (currentFocusIndex === null) {
			nextIndex = 0;
		} else if (currentFocusIndex < items.length - 1) {
			nextIndex = currentFocusIndex + 1;
		} else {
			// At end of page - try to go to next page
			const currentPage = get(paginationStore.page);
			const total = get(paginationStore.totalPages);

			if (currentPage < total) {
				await paginationActions.handleGoToNextPage();
				return true;
			}

			return false;
		}

		keyboardStore.focusAt(nextIndex);
		return true;
	}

	async function focusPreviousNotification(): Promise<boolean> {
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;

		if (items.length === 0) {
			return false;
		}

		const currentFocusIndex = get(keyboardStore.keyboardFocusIndex);
		let prevIndex: number;

		if (currentFocusIndex === null || currentFocusIndex === 0) {
			// At start of page - try to go to previous page
			const currentPage = get(paginationStore.page);

			if (currentPage > 1) {
				await paginationActions.handleGoToPreviousPage();
				return true;
			}

			return false;
		} else {
			prevIndex = currentFocusIndex - 1;
		}

		keyboardStore.focusAt(prevIndex);
		return true;
	}

	async function focusFirstNotification(): Promise<boolean> {
		const currentPage = get(paginationStore.page);
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;
		const isSplitMode = get(uiStore.splitModeEnabled);

		// If already on page 1, just focus first item
		if (currentPage === 1) {
			if (items.length === 0) {
				return false;
			}
			keyboardStore.focusAt(0);

			// In split mode, also open the detail
			if (isSplitMode && items[0]) {
				await detailActions.handleOpenInlineDetail(items[0]);
			}
			return true;
		}

		// Navigate to page 1 and focus first item (creates history entry)
		if (options.navigateToUrl) {
			const url = new URL(window.location.href);
			url.searchParams.delete("page");
			keyboardStore.setPendingFocusAction("first");

			// In split mode, set up to open the first notification
			if (isSplitMode) {
				url.searchParams.set("id", "__first__");
			} else {
				url.searchParams.delete("id");
			}

			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false,
			});
			return true;
		}

		return false;
	}

	async function focusLastNotification(): Promise<boolean> {
		const currentPage = get(paginationStore.page);
		const totalPagesCount = get(paginationStore.totalPages);
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;
		const isSplitMode = get(uiStore.splitModeEnabled);

		// If already on last page, just focus last item
		if (currentPage === totalPagesCount) {
			if (items.length === 0) {
				return false;
			}
			const lastIndex = items.length - 1;
			keyboardStore.focusAt(lastIndex);

			// In split mode, also open the detail
			if (isSplitMode && items[lastIndex]) {
				await detailActions.handleOpenInlineDetail(items[lastIndex]);
			}
			return true;
		}

		// Navigate to last page and focus last item (creates history entry)
		if (totalPagesCount > 1 && options.navigateToUrl) {
			const url = new URL(window.location.href);
			url.searchParams.set("page", totalPagesCount.toString());
			keyboardStore.setPendingFocusAction("last");

			// In split mode, set up to open the last notification
			if (isSplitMode) {
				url.searchParams.set("id", "__last__");
			} else {
				url.searchParams.delete("id");
			}

			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false,
			});
			return true;
		}

		return false;
	}

	function resetKeyboardFocus(): boolean {
		return keyboardStore.resetFocus();
	}

	function clearFocusIfNeeded(): void {
		keyboardStore.clearFocusIfNeeded();
	}

	function executePendingFocusAction(): void {
		keyboardStore.executePendingFocusAction();
	}

	function consumePendingFocus(): void {
		keyboardStore.consumePendingFocus();
	}

	return {
		focusNotificationAt,
		setFocusIndex,
		getFocusedNotification,
		focusNextNotification,
		focusPreviousNotification,
		focusFirstNotification,
		focusLastNotification,
		resetKeyboardFocus,
		clearFocusIfNeeded,
		executePendingFocusAction,
		consumePendingFocus,
	};
}
