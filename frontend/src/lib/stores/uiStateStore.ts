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

import { writable, readable } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import type { SnoozeDropdownState, TagDropdownState } from "$lib/state/types";

/**
 * UI State Store
 * Manages UI preferences (persisted to localStorage) and transient UI state
 */
export function createUIStateStore() {
	// Persistent UI state keys
	const READING_PANE_KEY = "octobud:splitMode:enabled";
	const READING_PANE_WIDTH_KEY = "octobud:splitMode:width";
	const SIDEBAR_COLLAPSED_KEY = "octobud:sidebar:collapsed";

	// Track if localStorage has been read (true if we're in browser, false during SSR)
	// This prevents flash of default content before preferences are loaded
	const isHydrated = typeof window !== "undefined";

	// Load from localStorage if available
	const storedSplitModeEnabled =
		typeof window !== "undefined" ? localStorage.getItem(READING_PANE_KEY) === "true" : false;
	const storedListPaneWidth =
		typeof window !== "undefined"
			? parseInt(localStorage.getItem(READING_PANE_WIDTH_KEY) || "40", 10)
			: 40;
	const storedSidebarCollapsed =
		typeof window !== "undefined" ? localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "true" : false;

	// Hydration state - true once localStorage values have been read
	const hydrated = readable<boolean>(isHydrated, () => {});

	// Create writable stores
	const splitModeEnabled = writable<boolean>(storedSplitModeEnabled);
	const listPaneWidth = writable<number>(storedListPaneWidth);
	const sidebarCollapsed = writable<boolean>(storedSidebarCollapsed);
	const snoozeDropdownState = writable<SnoozeDropdownState | null>(null);
	const tagDropdownState = writable<TagDropdownState | null>(null);
	const customDateDialogOpen = writable<boolean>(false);
	const customDateDialogNotificationId = writable<string | null>(null);
	const bulkUpdating = writable<boolean>(false);
	const savedListScrollPosition = writable<number>(0);

	// Persist to localStorage
	if (typeof window !== "undefined") {
		splitModeEnabled.subscribe((value) => {
			localStorage.setItem(READING_PANE_KEY, value.toString());
		});
		listPaneWidth.subscribe((value) => {
			localStorage.setItem(READING_PANE_WIDTH_KEY, value.toString());
		});
		sidebarCollapsed.subscribe((value) => {
			localStorage.setItem(SIDEBAR_COLLAPSED_KEY, value.toString());
		});
	}

	return {
		// Expose stores
		hydrated: hydrated as Readable<boolean>,
		splitModeEnabled: splitModeEnabled as Writable<boolean>,
		listPaneWidth: listPaneWidth as Writable<number>,
		sidebarCollapsed: sidebarCollapsed as Writable<boolean>,
		snoozeDropdownState: snoozeDropdownState as Writable<SnoozeDropdownState | null>,
		tagDropdownState: tagDropdownState as Writable<TagDropdownState | null>,
		customDateDialogOpen: customDateDialogOpen as Writable<boolean>,
		customDateDialogNotificationId: customDateDialogNotificationId as Writable<string | null>,
		bulkUpdating: bulkUpdating as Writable<boolean>,
		savedListScrollPosition: savedListScrollPosition as Writable<number>,

		// Actions
		toggleSplitMode: () => {
			splitModeEnabled.update((enabled) => !enabled);
		},
		setListPaneWidth: (width: number) => {
			const clamped = Math.max(30, Math.min(70, width));
			listPaneWidth.set(clamped);
		},
		toggleSidebar: () => {
			sidebarCollapsed.update((collapsed) => !collapsed);
		},
		openSnoozeDropdown: (notificationId: string, source: string) => {
			snoozeDropdownState.set({ notificationId, source });
		},
		closeSnoozeDropdown: () => {
			snoozeDropdownState.set(null);
		},
		openTagDropdown: (notificationId: string, source: string) => {
			tagDropdownState.set({ notificationId, source });
		},
		closeTagDropdown: () => {
			tagDropdownState.set(null);
		},
		openCustomSnoozeDialog: (notificationId: string) => {
			customDateDialogOpen.set(true);
			customDateDialogNotificationId.set(notificationId);
		},
		closeCustomSnoozeDialog: () => {
			customDateDialogOpen.set(false);
			customDateDialogNotificationId.set(null);
		},
	};
}

export type UIStore = ReturnType<typeof createUIStateStore>;
