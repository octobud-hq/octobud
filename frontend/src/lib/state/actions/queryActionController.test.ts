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

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { writable, get } from "svelte/store";
import { createQueryActionController } from "./queryActionController";
import { createDebounceManager, SEARCH_DEBOUNCE_MS } from "./debounceManager";
import { createSharedHelpers } from "./sharedHelpers";
import type { QueryStore } from "../../stores/queryStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { ControllerOptions } from "../interfaces/common";
import { fetchNotifications } from "$lib/api/notifications";

// Mock fetchNotifications
vi.mock("$lib/api/notifications", () => ({
	fetchNotifications: vi.fn(),
}));

describe("QueryActionController", () => {
	let queryStore: QueryStore;
	let paginationStore: PaginationStore;
	let notificationStore: NotificationStore;
	let debounceManager: ReturnType<typeof createDebounceManager>;
	let sharedHelpers: ReturnType<typeof createSharedHelpers>;
	let options: ControllerOptions;
	let controller: ReturnType<typeof createQueryActionController>;

	beforeEach(() => {
		vi.useFakeTimers();

		// Create mock stores
		queryStore = {
			searchTerm: writable(""),
			quickQuery: writable(""),
			viewQuery: writable(""),
			quickFilters: writable([]),
			setSearchTerm: vi.fn((value: string) => {
				queryStore.searchTerm.set(value);
			}),
			setQuickQuery: vi.fn((value: string) => {
				queryStore.quickQuery.set(value);
			}),
			setViewQuery: vi.fn((value: string) => {
				queryStore.viewQuery.set(value);
			}),
			addQuickFilter: vi.fn(() => {
				queryStore.quickFilters.update((filters) => [
					...filters,
					{ field: "isRead", operator: "equals", value: "true" },
				]);
			}),
			updateQuickFilter: vi.fn((index: number, patch: any) => {
				queryStore.quickFilters.update((filters) => {
					const updated = [...filters];
					updated[index] = { ...updated[index], ...patch };
					return updated;
				});
			}),
			removeQuickFilter: vi.fn((index: number) => {
				queryStore.quickFilters.update((filters) => filters.filter((_, i) => i !== index));
			}),
			clearFilters: vi.fn(() => {
				queryStore.quickQuery.set("");
				queryStore.quickFilters.set([]);
			}),
		} as any;

		paginationStore = {
			page: writable(2),
			isLoading: writable(false),
			total: writable(100),
			pageSize: writable(50),
			setPage: vi.fn((page: number) => {
				paginationStore.page.set(page);
			}),
			setLoading: vi.fn((loading: boolean) => {
				paginationStore.isLoading.set(loading);
			}),
			setTotal: vi.fn((total: number) => {
				paginationStore.total.set(total);
			}),
			setPageSize: vi.fn((size: number) => {
				paginationStore.pageSize.set(size);
			}),
		} as any;

		notificationStore = {
			pageData: writable({ items: [], total: 0, page: 1, pageSize: 50 }),
			setPageData: vi.fn((data: any) => {
				notificationStore.pageData.set(data);
			}),
		} as any;

		debounceManager = createDebounceManager();

		options = {
			navigateToUrl: vi.fn(async (url: string, opts?: any) => {}),
		};

		sharedHelpers = createSharedHelpers(
			notificationStore,
			paginationStore,
			queryStore,
			options,
			debounceManager
		);

		controller = createQueryActionController(
			queryStore,
			paginationStore,
			sharedHelpers,
			debounceManager
		);
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.clearAllMocks();
	});

	describe("handleSearchInput", () => {
		it("sets search term in store", () => {
			controller.handleSearchInput("test search");
			expect(queryStore.setSearchTerm).toHaveBeenCalledWith("test search");
			expect(get(queryStore.searchTerm)).toBe("test search");
		});

		it("resets page to 1", () => {
			controller.handleSearchInput("test");
			expect(paginationStore.setPage).toHaveBeenCalledWith(1);
		});

		it("schedules debounced refresh", () => {
			const scheduleSpy = vi.spyOn(sharedHelpers, "scheduleDebouncedRefresh");
			controller.handleSearchInput("test");
			expect(scheduleSpy).toHaveBeenCalled();
		});
	});

	describe("handleQuickQueryChange", () => {
		it("sets quick query in store", () => {
			controller.handleQuickQueryChange("repo:cli");
			expect(queryStore.setQuickQuery).toHaveBeenCalledWith("repo:cli");
			expect(get(queryStore.quickQuery)).toBe("repo:cli");
		});

		it("resets page to 1", () => {
			controller.handleQuickQueryChange("repo:cli");
			expect(paginationStore.setPage).toHaveBeenCalledWith(1);
		});

		it("debounces URL sync and refresh", async () => {
			vi.mocked(fetchNotifications).mockResolvedValue({
				items: [],
				total: 0,
				page: 1,
				pageSize: 50,
			});

			const syncSpy = vi.spyOn(sharedHelpers, "syncQueryToUrl");
			const refreshSpy = vi.spyOn(sharedHelpers, "refresh");

			controller.handleQuickQueryChange("repo:cli");

			// Should not be called immediately
			expect(syncSpy).not.toHaveBeenCalled();
			expect(refreshSpy).not.toHaveBeenCalled();

			// Advance timers to trigger debounce
			await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

			// Should be called after debounce
			expect(syncSpy).toHaveBeenCalled();
			expect(refreshSpy).toHaveBeenCalled();
		});

		it("cancels previous debounce when called again", async () => {
			vi.mocked(fetchNotifications).mockResolvedValue({
				items: [],
				total: 0,
				page: 1,
				pageSize: 50,
			});

			const syncSpy = vi.spyOn(sharedHelpers, "syncQueryToUrl");

			controller.handleQuickQueryChange("repo:cli");
			await vi.advanceTimersByTimeAsync(100);
			controller.handleQuickQueryChange("repo:cli-cli");

			// Advance timers - should only call once for the latest query
			await vi.advanceTimersByTimeAsync(SEARCH_DEBOUNCE_MS);

			expect(syncSpy).toHaveBeenCalledTimes(1);
		});
	});

	describe("addQuickFilter", () => {
		it("adds a filter to the store", () => {
			controller.addQuickFilter();
			expect(queryStore.addQuickFilter).toHaveBeenCalled();
			expect(get(queryStore.quickFilters).length).toBe(1);
		});
	});

	describe("updateQuickFilter", () => {
		beforeEach(() => {
			// Add a filter first
			queryStore.quickFilters.set([{ field: "isRead", operator: "equals", value: "true" }]);
		});

		it("updates filter at specified index", () => {
			controller.updateQuickFilter(0, { value: "unread" });
			expect(queryStore.updateQuickFilter).toHaveBeenCalledWith(0, { value: "unread" });
			expect(get(queryStore.quickFilters)[0].value).toBe("unread");
		});

		it("resets page to 1", () => {
			controller.updateQuickFilter(0, { value: "unread" });
			expect(paginationStore.setPage).toHaveBeenCalledWith(1);
		});

		it("triggers immediate refresh", async () => {
			const refreshSpy = vi.spyOn(sharedHelpers, "refresh");
			controller.updateQuickFilter(0, { value: "unread" });
			expect(refreshSpy).toHaveBeenCalled();
		});
	});

	describe("removeQuickFilter", () => {
		beforeEach(() => {
			queryStore.quickFilters.set([
				{ field: "isRead", operator: "equals", value: "true" },
				{ field: "repo", operator: "equals", value: "cli" },
			]);
		});

		it("removes filter at specified index", () => {
			controller.removeQuickFilter(0);
			expect(queryStore.removeQuickFilter).toHaveBeenCalledWith(0);
			expect(get(queryStore.quickFilters).length).toBe(1);
		});

		it("resets page to 1", () => {
			controller.removeQuickFilter(0);
			expect(paginationStore.setPage).toHaveBeenCalledWith(1);
		});

		it("triggers immediate refresh", async () => {
			const refreshSpy = vi.spyOn(sharedHelpers, "refresh");
			controller.removeQuickFilter(0);
			expect(refreshSpy).toHaveBeenCalled();
		});
	});

	describe("clearFilters", () => {
		beforeEach(() => {
			queryStore.quickQuery.set("repo:cli");
			queryStore.quickFilters.set([{ field: "isRead", operator: "equals", value: "true" }]);
		});

		it("clears all filters and query", async () => {
			await controller.clearFilters();
			expect(queryStore.clearFilters).toHaveBeenCalled();
			expect(get(queryStore.quickQuery)).toBe("");
			expect(get(queryStore.quickFilters).length).toBe(0);
		});

		it("resets page to 1", async () => {
			await controller.clearFilters();
			expect(paginationStore.setPage).toHaveBeenCalledWith(1);
		});

		it("syncs query to URL", async () => {
			const syncSpy = vi.spyOn(sharedHelpers, "syncQueryToUrl");
			await controller.clearFilters();
			expect(syncSpy).toHaveBeenCalled();
		});

		it("triggers refresh", async () => {
			const refreshSpy = vi.spyOn(sharedHelpers, "refresh");
			await controller.clearFilters();
			expect(refreshSpy).toHaveBeenCalled();
		});
	});
});
