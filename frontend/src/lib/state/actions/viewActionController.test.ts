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
import { writable, get } from "svelte/store";
import { tick } from "svelte";
import { createViewActionController } from "./viewActionController";
import { createSharedHelpers } from "./sharedHelpers";
import { createDebounceManager } from "./debounceManager";
import { fetchViews } from "$lib/api/views";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { SelectionStore } from "../../stores/selectionStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { QueryStore } from "../../stores/queryStore";
import type { ViewStore } from "../../stores/viewStore";
import type { ControllerOptions } from "../interfaces/common";

// Mock fetchViews
vi.mock("$lib/api/views", () => ({
	fetchViews: vi.fn(),
}));

describe("ViewActionController", () => {
	let notificationStore: NotificationStore;
	let paginationStore: PaginationStore;
	let detailStore: DetailStore;
	let selectionStore: SelectionStore;
	let keyboardStore: KeyboardStore;
	let uiStore: UIStore;
	let queryStore: QueryStore;
	let viewStore: ViewStore;
	let options: ControllerOptions;
	let sharedHelpers: ReturnType<typeof createSharedHelpers>;
	let debounceManager: ReturnType<typeof createDebounceManager>;
	let controller: ReturnType<typeof createViewActionController>;

	beforeEach(() => {
		global.window = {
			location: {
				href: "http://localhost:3000/views/inbox",
			},
		} as any;

		notificationStore = {
			pageData: writable({
				items: [{ id: "1", githubId: "gh-1" }],
				total: 1,
				page: 1,
				pageSize: 50,
			}),
			setPageData: vi.fn((data: any) => {
				notificationStore.pageData.set(data);
			}),
		} as any;

		paginationStore = {
			page: writable(1),
			isLoading: writable(false),
			total: writable(1),
			pageSize: writable(50),
			setPage: vi.fn((page: number) => {
				paginationStore.page.set(page);
			}),
			setTotal: vi.fn((total: number) => {
				paginationStore.total.set(total);
			}),
			setPageSize: vi.fn((size: number) => {
				paginationStore.pageSize.set(size);
			}),
			setLoading: vi.fn((loading: boolean) => {
				paginationStore.isLoading.set(loading);
			}),
		} as any;

		detailStore = {
			detailOpen: writable(false),
			detailNotificationId: writable(null),
		} as any;

		selectionStore = {
			clearSelection: vi.fn(),
			exitMultiselectMode: vi.fn(),
		} as any;

		keyboardStore = {
			keyboardFocusIndex: writable(0),
			setFocusIndex: vi.fn((index: number) => {
				keyboardStore.keyboardFocusIndex.set(index);
				return true;
			}),
			resetFocus: vi.fn(() => {
				keyboardStore.keyboardFocusIndex.set(null);
				return true;
			}),
		} as any;

		const splitModeEnabled = writable(false);
		uiStore = {
			splitModeEnabled,
		} as any;

		queryStore = {
			quickQuery: writable(""),
			viewQuery: writable(""),
			searchTerm: writable(""),
			setQuickQuery: vi.fn((query: string) => {
				queryStore.quickQuery.set(query);
			}),
			setViewQuery: vi.fn((query: string) => {
				queryStore.viewQuery.set(query);
			}),
			setSearchTerm: vi.fn((term: string) => {
				queryStore.searchTerm.set(term);
			}),
		} as any;

		viewStore = {
			views: writable([]),
			selectedViewId: writable("inbox"),
			selectedViewSlug: writable("inbox"),
			builtInViewList: writable([
				{ id: "inbox", slug: "inbox", name: "Inbox" },
				{ id: "archive", slug: "archive", name: "Archive" },
			]),
			selectView: vi.fn((id: string) => {
				viewStore.selectedViewId.set(id);
			}),
			setViews: vi.fn((views: any[]) => {
				viewStore.views.set(views);
			}),
		} as any;

		options = {
			navigateToUrl: vi.fn(async (url: string, opts?: any) => {}),
		};

		debounceManager = createDebounceManager();
		sharedHelpers = createSharedHelpers(
			notificationStore,
			paginationStore,
			queryStore,
			options,
			debounceManager
		);

		controller = createViewActionController(
			{
				notificationStore,
				paginationStore,
				detailStore,
				selectionStore,
				keyboardStore,
				uiStore,
				queryStore,
				viewStore,
			},
			options,
			sharedHelpers,
			debounceManager
		);
	});

	describe("selectViewBySlug", () => {
		it("selects built-in view by slug", async () => {
			await controller.selectViewBySlug("archive");

			expect(selectionStore.clearSelection).toHaveBeenCalled();
			expect(selectionStore.exitMultiselectMode).toHaveBeenCalled();
			expect(viewStore.selectView).toHaveBeenCalled();
			expect(options.navigateToUrl).toHaveBeenCalled();
		});

		it("clears selection when switching views", async () => {
			await controller.selectViewBySlug("archive");

			expect(selectionStore.clearSelection).toHaveBeenCalled();
			expect(selectionStore.exitMultiselectMode).toHaveBeenCalled();
		});

		it("navigates to view URL", async () => {
			await controller.selectViewBySlug("archive");

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).toContain("/views/archive");
			expect(call[1]?.replace).toBe(false);
		});
	});

	describe("refreshViewCounts", () => {
		it("fetches and updates views", async () => {
			const mockViews = [
				{ id: "1", slug: "custom", name: "Custom", query: "repo:cli", unreadCount: 5 },
			];
			vi.mocked(fetchViews).mockResolvedValue(mockViews);

			await controller.refreshViewCounts();

			expect(fetchViews).toHaveBeenCalled();
			expect(viewStore.setViews).toHaveBeenCalledWith(mockViews);
		});

		it("handles errors gracefully", async () => {
			vi.mocked(fetchViews).mockRejectedValue(new Error("Network error"));

			await controller.refreshViewCounts();

			// Should not throw, just log error
			expect(viewStore.setViews).not.toHaveBeenCalled();
		});
	});

	describe("handleSyncNewNotifications", () => {
		it("refreshes view counts", async () => {
			vi.mocked(fetchViews).mockResolvedValue([]);
			paginationStore.page.set(1);
			(uiStore.splitModeEnabled as any).set(false);
			(detailStore.detailOpen as any).set(false);

			await controller.handleSyncNewNotifications();

			expect(fetchViews).toHaveBeenCalled();
		});

		it("refreshes notifications when on page 1 and not in detail view", async () => {
			vi.mocked(fetchViews).mockResolvedValue([]);
			paginationStore.page.set(1);
			(uiStore.splitModeEnabled as any).set(false);
			(detailStore.detailOpen as any).set(false);
			const refreshSpy = vi.spyOn(sharedHelpers, "refresh");

			await controller.handleSyncNewNotifications();

			expect(refreshSpy).toHaveBeenCalled();
		});

		it("does not refresh notifications when not on page 1", async () => {
			vi.mocked(fetchViews).mockResolvedValue([]);
			paginationStore.page.set(2);
			const refreshSpy = vi.spyOn(sharedHelpers, "refresh");

			await controller.handleSyncNewNotifications();

			expect(refreshSpy).not.toHaveBeenCalled();
		});

		it("restores keyboard focus after refresh", async () => {
			vi.mocked(fetchViews).mockResolvedValue([]);
			paginationStore.page.set(1);
			(uiStore.splitModeEnabled as any).set(false);
			(detailStore.detailOpen as any).set(false);
			keyboardStore.keyboardFocusIndex.set(0);

			await controller.handleSyncNewNotifications();
			await tick();

			// Focus should be restored if notification still exists
			expect(keyboardStore.setFocusIndex).toHaveBeenCalled();
		});
	});

	describe("syncFromData", () => {
		it("syncs all state from page data", () => {
			const pageData = {
				items: [],
				total: 10,
				page: 2,
				pageSize: 50,
			};
			const viewData = {
				selectedViewId: "archive",
				initialPage: pageData,
				initialPageNumber: 2,
				initialSearchTerm: "test",
			} as any;

			controller.syncFromData(viewData);

			expect(viewStore.selectView).toHaveBeenCalled();
			expect(notificationStore.setPageData).toHaveBeenCalledWith(pageData);
			expect(paginationStore.setPage).toHaveBeenCalledWith(2);
			expect(paginationStore.setTotal).toHaveBeenCalledWith(10);
			expect(queryStore.setSearchTerm).toHaveBeenCalledWith("test");
		});
	});
});
