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

import { writable, derived, get } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import { tick } from "svelte";
import type { Notification } from "$lib/api/types";
import type { NotificationRowComponent } from "$lib/state/types";
import type { NotificationStore } from "./notificationStore";

/**
 * Keyboard Navigation Store
 * Manages keyboard focus and navigation
 */
export function createKeyboardNavigationStore(notificationStore: NotificationStore) {
	const keyboardFocusIndex = writable<number | null>(null);
	const notificationRowComponents = writable<Array<NotificationRowComponent>>([]);
	const pendingFocusAction = writable<"first" | "last" | null>(null);
	// When set, indicates a row should focus itself (used for keyboard/button nav only)
	const pendingFocusIndex = writable<number | null>(null);

	// Derived: focused notification
	const focusedNotification = derived(
		[keyboardFocusIndex, notificationStore.pageData],
		([$keyboardFocusIndex, $pageData]) => {
			if ($keyboardFocusIndex === null) return null;
			return $pageData.items[$keyboardFocusIndex] ?? null;
		}
	);

	/**
	 * Clear the pending focus index (called after a row focuses itself)
	 */
	function consumePendingFocus(): void {
		pendingFocusIndex.set(null);
	}

	/**
	 * Set keyboard focus to a specific index AND focus the DOM element.
	 * Use this for keyboard/button navigation.
	 */
	function focusAt(index: number): boolean {
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;

		if (items.length === 0) {
			keyboardFocusIndex.set(null);
			return false;
		}

		const clamped = Math.max(0, Math.min(index, items.length - 1));
		keyboardFocusIndex.set(clamped);

		// Signal that we want to focus this element
		pendingFocusIndex.set(clamped);
		return true;
	}

	/**
	 * Update the keyboard focus index WITHOUT focusing the DOM element.
	 * Use this for click selection where the browser handles focus.
	 */
	function setFocusIndex(index: number): boolean {
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;

		if (items.length === 0) {
			keyboardFocusIndex.set(null);
			return false;
		}

		const clamped = Math.max(0, Math.min(index, items.length - 1));
		keyboardFocusIndex.set(clamped);
		return true;
	}

	/**
	 * Reset keyboard focus
	 */
	function resetFocus(): boolean {
		const currentFocusIndex = get(keyboardFocusIndex);

		// Blur the currently focused row component if any
		if (currentFocusIndex !== null) {
			const components = get(notificationRowComponents);
			const rowComponent = components[currentFocusIndex];
			// Note: blur is not part of the interface, but may exist on DOM elements
			if (
				rowComponent &&
				"blur" in rowComponent &&
				typeof (rowComponent as any).blur === "function"
			) {
				(rowComponent as any).blur();
			}
		}

		keyboardFocusIndex.set(null);
		return true;
	}

	/**
	 * Clear focus if index is out of bounds
	 */
	function clearFocusIfNeeded(): void {
		const currentFocusIndex = get(keyboardFocusIndex);
		const pageData = get(notificationStore.pageData);
		const items = pageData.items;

		if (
			currentFocusIndex !== null &&
			(currentFocusIndex < 0 || currentFocusIndex >= items.length)
		) {
			keyboardFocusIndex.set(null);
		}
	}

	/**
	 * Execute pending focus action after page load
	 */
	function executePendingFocusAction(): void {
		const action = get(pendingFocusAction);
		if (!action) return;

		const pageData = get(notificationStore.pageData);
		const items = pageData.items;
		if (items.length === 0) return;

		if (action === "first") {
			focusAt(0);
		} else if (action === "last") {
			focusAt(items.length - 1);
		}

		// Clear the pending action
		pendingFocusAction.set(null);
	}

	return {
		// Expose stores
		keyboardFocusIndex: keyboardFocusIndex as Writable<number | null>,
		notificationRowComponents: notificationRowComponents as Writable<
			Array<NotificationRowComponent>
		>,
		pendingFocusAction: pendingFocusAction as Writable<"first" | "last" | null>,
		pendingFocusIndex: pendingFocusIndex as Readable<number | null>,

		// Derived stores
		focusedNotification: focusedNotification as Readable<Notification | null>,

		// Actions
		focusAt,
		setFocusIndex,
		resetFocus,
		clearFocusIfNeeded,
		executePendingFocusAction,
		consumePendingFocus,
		setPendingFocusAction: (action: "first" | "last" | null) => {
			pendingFocusAction.set(action);
		},
	};
}

export type KeyboardStore = ReturnType<typeof createKeyboardNavigationStore>;
