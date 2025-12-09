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

import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";

// Mock localStorage before importing the store
const mockLocalStorage = {
	store: {} as Record<string, string>,
	getItem: vi.fn((key: string) => mockLocalStorage.store[key] || null),
	setItem: vi.fn((key: string, value: string) => {
		mockLocalStorage.store[key] = value;
	}),
	clear: () => {
		mockLocalStorage.store = {};
	},
};

Object.defineProperty(global, "window", {
	value: {
		localStorage: mockLocalStorage,
	},
	writable: true,
});

Object.defineProperty(global, "localStorage", {
	value: mockLocalStorage,
	writable: true,
});

// Import after mocking
import { createUIStateStore } from "./uiStateStore";

describe("UIStateStore", () => {
	beforeEach(() => {
		mockLocalStorage.clear();
		vi.clearAllMocks();
	});

	describe("initialization", () => {
		it("initializes with default values when localStorage is empty", () => {
			const store = createUIStateStore();

			expect(get(store.splitModeEnabled)).toBe(false);
			expect(get(store.listPaneWidth)).toBe(40);
			expect(get(store.sidebarCollapsed)).toBe(false);
		});

		it("loads splitModeEnabled from localStorage", () => {
			mockLocalStorage.store["octobud:splitMode:enabled"] = "true";

			const store = createUIStateStore();

			expect(get(store.splitModeEnabled)).toBe(true);
		});

		it("loads listPaneWidth from localStorage", () => {
			mockLocalStorage.store["octobud:splitMode:width"] = "55";

			const store = createUIStateStore();

			expect(get(store.listPaneWidth)).toBe(55);
		});

		it("loads sidebarCollapsed from localStorage", () => {
			mockLocalStorage.store["octobud:sidebar:collapsed"] = "true";

			const store = createUIStateStore();

			expect(get(store.sidebarCollapsed)).toBe(true);
		});

		it("initializes transient state to null/false", () => {
			const store = createUIStateStore();

			expect(get(store.snoozeDropdownState)).toBeNull();
			expect(get(store.tagDropdownState)).toBeNull();
			expect(get(store.customDateDialogOpen)).toBe(false);
			expect(get(store.customDateDialogNotificationId)).toBeNull();
			expect(get(store.bulkUpdating)).toBe(false);
			expect(get(store.savedListScrollPosition)).toBe(0);
		});
	});

	describe("toggleSplitMode", () => {
		it("toggles split mode on", () => {
			const store = createUIStateStore();

			store.toggleSplitMode();

			expect(get(store.splitModeEnabled)).toBe(true);
		});

		it("toggles split mode off", () => {
			mockLocalStorage.store["octobud:splitMode:enabled"] = "true";
			const store = createUIStateStore();

			store.toggleSplitMode();

			expect(get(store.splitModeEnabled)).toBe(false);
		});

		it("persists to localStorage", () => {
			const store = createUIStateStore();

			store.toggleSplitMode();

			expect(mockLocalStorage.setItem).toHaveBeenCalledWith("octobud:splitMode:enabled", "true");
		});
	});

	describe("setListPaneWidth", () => {
		it("sets width within valid range", () => {
			const store = createUIStateStore();

			store.setListPaneWidth(50);

			expect(get(store.listPaneWidth)).toBe(50);
		});

		it("clamps width to minimum of 30", () => {
			const store = createUIStateStore();

			store.setListPaneWidth(20);

			expect(get(store.listPaneWidth)).toBe(30);
		});

		it("clamps width to maximum of 70", () => {
			const store = createUIStateStore();

			store.setListPaneWidth(80);

			expect(get(store.listPaneWidth)).toBe(70);
		});

		it("persists to localStorage", () => {
			const store = createUIStateStore();

			store.setListPaneWidth(55);

			expect(mockLocalStorage.setItem).toHaveBeenCalledWith("octobud:splitMode:width", "55");
		});
	});

	describe("toggleSidebar", () => {
		it("toggles sidebar collapsed state", () => {
			const store = createUIStateStore();

			store.toggleSidebar();
			expect(get(store.sidebarCollapsed)).toBe(true);

			store.toggleSidebar();
			expect(get(store.sidebarCollapsed)).toBe(false);
		});

		it("persists to localStorage", () => {
			const store = createUIStateStore();

			store.toggleSidebar();

			expect(mockLocalStorage.setItem).toHaveBeenCalledWith("octobud:sidebar:collapsed", "true");
		});
	});

	describe("snooze dropdown", () => {
		it("opens snooze dropdown", () => {
			const store = createUIStateStore();

			store.openSnoozeDropdown("notif-1", "row");

			expect(get(store.snoozeDropdownState)).toEqual({
				notificationId: "notif-1",
				source: "row",
			});
		});

		it("closes snooze dropdown", () => {
			const store = createUIStateStore();

			store.openSnoozeDropdown("notif-1", "row");
			store.closeSnoozeDropdown();

			expect(get(store.snoozeDropdownState)).toBeNull();
		});
	});

	describe("tag dropdown", () => {
		it("opens tag dropdown", () => {
			const store = createUIStateStore();

			store.openTagDropdown("notif-1", "detail");

			expect(get(store.tagDropdownState)).toEqual({
				notificationId: "notif-1",
				source: "detail",
			});
		});

		it("closes tag dropdown", () => {
			const store = createUIStateStore();

			store.openTagDropdown("notif-1", "detail");
			store.closeTagDropdown();

			expect(get(store.tagDropdownState)).toBeNull();
		});
	});

	describe("custom snooze dialog", () => {
		it("opens custom snooze dialog", () => {
			const store = createUIStateStore();

			store.openCustomSnoozeDialog("notif-1");

			expect(get(store.customDateDialogOpen)).toBe(true);
			expect(get(store.customDateDialogNotificationId)).toBe("notif-1");
		});

		it("closes custom snooze dialog", () => {
			const store = createUIStateStore();

			store.openCustomSnoozeDialog("notif-1");
			store.closeCustomSnoozeDialog();

			expect(get(store.customDateDialogOpen)).toBe(false);
			expect(get(store.customDateDialogNotificationId)).toBeNull();
		});
	});
});
