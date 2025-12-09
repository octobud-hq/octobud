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
import { createSharedHelpers } from "./sharedHelpers";
import { createDebounceManager } from "./debounceManager";
import { fetchNotifications } from "$lib/api/notifications";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { QueryStore } from "../../stores/queryStore";
import type { ControllerOptions } from "../interfaces/common";

// Mock fetchNotifications
vi.mock("$lib/api/notifications", () => ({
	fetchNotifications: vi.fn(),
}));

describe("SharedHelpers", () => {
	let notificationStore: NotificationStore;
	let paginationStore: PaginationStore;
	let queryStore: QueryStore;
	let options: ControllerOptions;
	let helpers: ReturnType<typeof createSharedHelpers>;

	beforeEach(() => {
		global.window = {
			location: {
				href: "http://localhost:3000/views/inbox",
			},
		} as any;

		notificationStore = {
			pageData: writable({
				items: [],
				total: 0,
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
			total: writable(0),
			pageSize: writable(50),
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

		queryStore = {
			quickQuery: writable(""),
			viewQuery: writable(""),
		} as any;

		options = {
			navigateToUrl: vi.fn(async (url: string, opts?: any) => {}),
			onAfterRefresh: vi.fn(),
		};

		const debounceManager = createDebounceManager();
		helpers = createSharedHelpers(
			notificationStore,
			paginationStore,
			queryStore,
			options,
			debounceManager
		);
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	describe("refresh", () => {
		it("fetches notifications and updates stores", async () => {
			const mockResponse = {
				items: [
					{
						id: "1",
						githubId: "gh-1",
						repoFullName: "owner/repo",
						reason: "review_requested",
						subjectTitle: "Test",
						subjectType: "pull_request",
						updatedAt: "2024-01-15T10:00:00Z",
						isRead: false,
						archived: false,
						muted: false,
						starred: false,
						labels: [],
						viewIds: [],
					},
				],
				total: 1,
				page: 1,
				pageSize: 50,
			};
			vi.mocked(fetchNotifications).mockResolvedValue(mockResponse);

			await helpers.refresh();

			expect(fetchNotifications).toHaveBeenCalledWith({
				page: 1,
				filters: {
					query: undefined,
					filters: [],
				},
			});
			expect(notificationStore.setPageData).toHaveBeenCalledWith(mockResponse);
			expect(paginationStore.setTotal).toHaveBeenCalledWith(1);
			expect(paginationStore.setPageSize).toHaveBeenCalledWith(50);
			expect(options.onAfterRefresh).toHaveBeenCalled();
		});

		it("uses current page and query from stores", async () => {
			paginationStore.page.set(2);
			queryStore.quickQuery.set("repo:cli");
			const mockResponse = {
				items: [],
				total: 0,
				page: 2,
				pageSize: 50,
			};
			vi.mocked(fetchNotifications).mockResolvedValue(mockResponse);

			await helpers.refresh();

			expect(fetchNotifications).toHaveBeenCalledWith({
				page: 2,
				filters: {
					query: "repo:cli",
					filters: [],
				},
			});
		});

		it("handles API errors gracefully", async () => {
			const error = new Error("Network error");
			vi.mocked(fetchNotifications).mockRejectedValue(error);

			await helpers.refresh();

			expect(paginationStore.setLoading).toHaveBeenCalledWith(false);
			// Should not crash, just log error
		});

		it("sets loading state during refresh", async () => {
			let resolvePromise: (value: any) => void;
			const promise = new Promise((resolve) => {
				resolvePromise = resolve;
			});
			vi.mocked(fetchNotifications).mockReturnValue(promise as any);

			const refreshPromise = helpers.refresh();
			expect(paginationStore.setLoading).toHaveBeenCalledWith(true);

			resolvePromise!({
				items: [],
				total: 0,
				page: 1,
				pageSize: 50,
			});
			await refreshPromise;

			expect(paginationStore.setLoading).toHaveBeenCalledWith(false);
		});
	});

	describe("syncQueryToUrl", () => {
		it("adds query param when query differs from view query", async () => {
			queryStore.quickQuery.set("repo:cli");
			queryStore.viewQuery.set("");
			paginationStore.page.set(1);

			await helpers.syncQueryToUrl();

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).toContain("query=repo%3Acli");
			expect(call[1]?.replace).toBe(true);
		});

		it("removes query param when query matches view query", async () => {
			queryStore.quickQuery.set("repo:cli");
			queryStore.viewQuery.set("repo:cli");
			paginationStore.page.set(1);

			await helpers.syncQueryToUrl();

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).not.toContain("query=");
		});

		it("adds page param when page > 1", async () => {
			queryStore.quickQuery.set("");
			paginationStore.page.set(3);

			await helpers.syncQueryToUrl();

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).toContain("page=3");
		});

		it("removes page param when page is 1", async () => {
			queryStore.quickQuery.set("");
			paginationStore.page.set(1);

			await helpers.syncQueryToUrl();

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).not.toContain("page=");
		});

		it("does nothing when navigateToUrl is not provided", async () => {
			options.navigateToUrl = undefined;
			await helpers.syncQueryToUrl();
			// Should not throw
		});
	});

	describe("scheduleDebouncedRefresh", () => {
		it("schedules a debounced refresh", async () => {
			vi.useFakeTimers();
			const mockResponse = {
				items: [],
				total: 0,
				page: 1,
				pageSize: 50,
			};
			vi.mocked(fetchNotifications).mockResolvedValue(mockResponse);

			helpers.scheduleDebouncedRefresh();
			expect(fetchNotifications).not.toHaveBeenCalled();

			await vi.advanceTimersByTimeAsync(300); // SEARCH_DEBOUNCE_MS

			expect(fetchNotifications).toHaveBeenCalled();
			vi.useRealTimers();
		});
	});

	describe("updateUrlWithDetailId", () => {
		it("adds id param to URL", async () => {
			await helpers.updateUrlWithDetailId("gh-123");

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).toContain("id=gh-123");
			expect(call[1]?.replace).toBe(true);
		});

		it("does nothing when navigateToUrl is not provided", async () => {
			options.navigateToUrl = undefined;
			await helpers.updateUrlWithDetailId("gh-123");
			// Should not throw
		});
	});

	describe("updateUrlWithoutDetailId", () => {
		it("removes id param from URL", async () => {
			await helpers.updateUrlWithoutDetailId();

			expect(options.navigateToUrl).toHaveBeenCalled();
			const call = vi.mocked(options.navigateToUrl).mock.calls[0];
			expect(call[0]).not.toContain("id=");
			expect(call[1]?.replace).toBe(true);
		});

		it("does nothing when navigateToUrl is not provided", async () => {
			options.navigateToUrl = undefined;
			await helpers.updateUrlWithoutDetailId();
			// Should not throw
		});
	});
});
