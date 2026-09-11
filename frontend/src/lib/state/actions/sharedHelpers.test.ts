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
import { fetchNotifications, fetchRepositoryCounts } from "$lib/api/notifications";
import { createRepositoryFilterStore } from "../../stores/repositoryFilterStore";
import { createRepositoryPinsStore } from "../../stores/repositoryPinsStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { QueryStore } from "../../stores/queryStore";
import type { ControllerOptions } from "../interfaces/common";

// Mock fetchNotifications
vi.mock("$lib/api/notifications", () => ({
	fetchNotifications: vi.fn(),
	fetchRepositoryCounts: vi.fn(),
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

		it("scopes the fetch to the repository filter when one is set", async () => {
			const repositoryFilterStore = createRepositoryFilterStore([7, 9]);
			const scopedHelpers = createSharedHelpers(
				notificationStore,
				paginationStore,
				queryStore,
				options,
				createDebounceManager(),
				repositoryFilterStore
			);
			queryStore.quickQuery.set("in:inbox");
			vi.mocked(fetchNotifications).mockResolvedValue({
				items: [],
				total: 0,
				page: 1,
				pageSize: 50,
			});

			await scopedHelpers.refresh();

			expect(fetchNotifications).toHaveBeenCalledWith({
				page: 1,
				repositoryIds: [7, 9],
				filters: {
					query: "in:inbox",
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

	describe("refreshRepositoryCounts", () => {
		it("is a no-op without a repository filter store", async () => {
			await helpers.refreshRepositoryCounts();
			expect(fetchRepositoryCounts).not.toHaveBeenCalled();
		});

		it("fetches counts for the current query and stores them", async () => {
			const repositoryFilterStore = createRepositoryFilterStore([1]);
			const scopedHelpers = createSharedHelpers(
				notificationStore,
				paginationStore,
				queryStore,
				options,
				createDebounceManager(),
				repositoryFilterStore
			);
			queryStore.quickQuery.set("is:unread");
			const counts = [{ repository: { id: 1, name: "a", fullName: "org/a" }, total: 3, unread: 1 }];
			vi.mocked(fetchRepositoryCounts).mockResolvedValue(counts);

			await scopedHelpers.refreshRepositoryCounts();

			// Selected ids are always included so the server returns their identity.
			expect(fetchRepositoryCounts).toHaveBeenCalledWith("is:unread", [1]);
			expect(get(repositoryFilterStore.repositoryCounts)).toEqual(counts);
			expect(get(repositoryFilterStore.countsQuery)).toBe("is:unread");
			expect(get(repositoryFilterStore.countsLoading)).toBe(false);
		});

		it("includes pinned ids and ignores out-of-order responses", async () => {
			const repositoryFilterStore = createRepositoryFilterStore([1]);
			const pinsStore = createRepositoryPinsStore(null);
			pinsStore.togglePin(7);
			const scopedHelpers = createSharedHelpers(
				notificationStore,
				paginationStore,
				queryStore,
				options,
				createDebounceManager(),
				repositoryFilterStore,
				pinsStore
			);
			const first = [{ repository: { id: 1, name: "a", fullName: "org/a" }, total: 1, unread: 0 }];
			const second = [{ repository: { id: 2, name: "b", fullName: "org/b" }, total: 2, unread: 0 }];
			let resolveFirst: (value: typeof first) => void = () => {};
			vi.mocked(fetchRepositoryCounts)
				.mockImplementationOnce(
					() =>
						new Promise((resolve) => {
							resolveFirst = resolve;
						})
				)
				.mockResolvedValueOnce(second);

			queryStore.quickQuery.set("in:inbox");
			const slow = scopedHelpers.refreshRepositoryCounts();
			queryStore.quickQuery.set("in:archive");
			await scopedHelpers.refreshRepositoryCounts();
			resolveFirst(first);
			await slow;

			expect(fetchRepositoryCounts).toHaveBeenNthCalledWith(1, "in:inbox", [1, 7]);
			expect(get(repositoryFilterStore.repositoryCounts)).toEqual(second);
			expect(get(repositoryFilterStore.countsQuery)).toBe("in:archive");
			expect(get(repositoryFilterStore.countsLoading)).toBe(false);
		});

		it("keeps previous counts when the fetch fails", async () => {
			const previous = [
				{ repository: { id: 2, name: "b", fullName: "org/b" }, total: 1, unread: 0 },
			];
			const repositoryFilterStore = createRepositoryFilterStore([], previous, "in:inbox");
			const scopedHelpers = createSharedHelpers(
				notificationStore,
				paginationStore,
				queryStore,
				options,
				createDebounceManager(),
				repositoryFilterStore
			);
			vi.mocked(fetchRepositoryCounts).mockRejectedValue(new Error("boom"));

			await scopedHelpers.refreshRepositoryCounts();

			expect(get(repositoryFilterStore.repositoryCounts)).toEqual(previous);
			expect(get(repositoryFilterStore.countsLoading)).toBe(false);
		});
	});
});
