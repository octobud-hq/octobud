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

import { describe, it, expect, vi, beforeEach } from "vitest";
import { writable, get, derived } from "svelte/store";
import { createKeyboardActionController } from "./keyboardActionController";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { PaginationActions } from "../interfaces/paginationActions";
import type { DetailActions } from "../interfaces/detailActions";
import type { ControllerOptions } from "../interfaces/common";

describe("KeyboardActionController", () => {
	let keyboardStore: KeyboardStore;
	let notificationStore: NotificationStore;
	let paginationStore: PaginationStore;
	let uiStore: UIStore;
	let detailStore: DetailStore;
	let options: ControllerOptions;
	let paginationActions: Pick<PaginationActions, "handleGoToNextPage" | "handleGoToPreviousPage">;
	let detailActions: Pick<DetailActions, "handleOpenInlineDetail">;
	let controller: ReturnType<typeof createKeyboardActionController>;

	beforeEach(() => {
		global.window = {
			location: {
				href: "http://localhost:3000/views/inbox",
			},
		} as any;

		const keyboardFocusIndex = writable<number | null>(5);
		const mockNotification = { id: "1", githubId: "gh-1" };
		const pageData = writable({
			items: [mockNotification, { id: "2", githubId: "gh-2" }],
			total: 2,
			page: 1,
			pageSize: 50,
		});

		// Create a simple readable store for focusedNotification that updates based on focus index
		const focusedNotification = derived(
			[keyboardFocusIndex, pageData],
			([$keyboardFocusIndex, $pageData]) => {
				if ($keyboardFocusIndex === null) return null;
				return $pageData.items[$keyboardFocusIndex] ?? null;
			}
		);

		keyboardStore = {
			keyboardFocusIndex,
			focusedNotification,
			focusAt: vi.fn((index: number) => {
				keyboardFocusIndex.set(index);
				return true;
			}),
			setFocusIndex: vi.fn((index: number) => {
				keyboardFocusIndex.set(index);
				return true;
			}),
			resetFocus: vi.fn(() => {
				keyboardFocusIndex.set(null);
				return true;
			}),
			clearFocusIfNeeded: vi.fn(),
			executePendingFocusAction: vi.fn(),
			consumePendingFocus: vi.fn(),
			setPendingFocusAction: vi.fn(),
		} as any;

		notificationStore = {
			pageData,
		} as any;

		const page = writable(1);
		const total = writable(50);
		const pageSize = writable(50);
		const totalPages = derived([total, pageSize], ([$total, $pageSize]) =>
			Math.ceil($total / $pageSize)
		);

		paginationStore = {
			page,
			total,
			pageSize,
			totalPages,
		} as any;

		uiStore = {
			sidebarCollapsed: writable(false),
			splitModeEnabled: writable(false),
			savedListScrollPosition: writable(0),
		} as any;

		detailStore = {
			detailNotificationId: writable(null),
		} as any;

		options = {
			navigateToUrl: vi.fn(async (url: string, opts?: any) => {}),
		};

		paginationActions = {
			handleGoToNextPage: vi.fn(async () => true),
			handleGoToPreviousPage: vi.fn(async () => true),
		};

		detailActions = {
			handleOpenInlineDetail: vi.fn(async (notification: any) => {}),
		};

		controller = createKeyboardActionController(
			keyboardStore,
			notificationStore,
			paginationStore,
			uiStore,
			detailStore,
			options,
			paginationActions,
			detailActions
		);
	});

	describe("focusNotificationAt", () => {
		it("focuses notification at specified index", () => {
			const result = controller.focusNotificationAt(0);
			expect(result).toBe(true);
			expect(keyboardStore.focusAt).toHaveBeenCalledWith(0);
		});
	});

	describe("setFocusIndex", () => {
		it("sets focus index without triggering DOM focus", () => {
			const result = controller.setFocusIndex(2);
			expect(result).toBe(true);
			expect(keyboardStore.setFocusIndex).toHaveBeenCalledWith(2);
		});
	});

	describe("getFocusedNotification", () => {
		it("returns focused notification", () => {
			const pageData = get(notificationStore.pageData);
			// Use the actual store setter
			const keyboardFocusIndex = (keyboardStore as any).keyboardFocusIndex;
			keyboardFocusIndex.set(0);
			// Wait for derived store to update
			const focused = controller.getFocusedNotification();
			expect(focused).toEqual(pageData.items[0]);
		});

		it("returns null when no focus", () => {
			const keyboardFocusIndex = (keyboardStore as any).keyboardFocusIndex;
			keyboardFocusIndex.set(null);
			const focused = controller.getFocusedNotification();
			expect(focused).toBeNull();
		});
	});

	describe("focusNextNotification", () => {
		it("focuses next notification on same page", async () => {
			keyboardStore.keyboardFocusIndex.set(0);
			const result = await controller.focusNextNotification();
			expect(result).toBe(true);
			expect(keyboardStore.focusAt).toHaveBeenCalledWith(1);
		});

		it("navigates to next page when at end of page", async () => {
			const pageData = get(notificationStore.pageData);
			keyboardStore.keyboardFocusIndex.set(pageData.items.length - 1);
			const page = (paginationStore as any).page;
			const total = (paginationStore as any).total;
			page.set(1);
			total.set(100); // 100 items = 2 pages with 50 per page
			await controller.focusNextNotification();
			expect(paginationActions.handleGoToNextPage).toHaveBeenCalled();
		});

		it("returns false when at last page and last item", async () => {
			const pageData = get(notificationStore.pageData);
			keyboardStore.keyboardFocusIndex.set(pageData.items.length - 1);
			const page = (paginationStore as any).page;
			const total = (paginationStore as any).total;
			page.set(1);
			total.set(50); // 50 items = 1 page with 50 per page
			const result = await controller.focusNextNotification();
			expect(result).toBe(false);
		});
	});

	describe("focusPreviousNotification", () => {
		it("focuses previous notification on same page", async () => {
			keyboardStore.keyboardFocusIndex.set(1);
			const result = await controller.focusPreviousNotification();
			expect(result).toBe(true);
			expect(keyboardStore.focusAt).toHaveBeenCalledWith(0);
		});

		it("navigates to previous page when at start of page", async () => {
			keyboardStore.keyboardFocusIndex.set(0);
			const page = (paginationStore as any).page;
			page.set(2);
			await controller.focusPreviousNotification();
			expect(paginationActions.handleGoToPreviousPage).toHaveBeenCalled();
		});

		it("returns false when at first page and first item", async () => {
			keyboardStore.keyboardFocusIndex.set(0);
			const page = (paginationStore as any).page;
			page.set(1);
			const result = await controller.focusPreviousNotification();
			expect(result).toBe(false);
		});
	});

	describe("resetKeyboardFocus", () => {
		it("resets focus", () => {
			const result = controller.resetKeyboardFocus();
			expect(result).toBe(true);
			expect(keyboardStore.resetFocus).toHaveBeenCalled();
		});
	});
});
