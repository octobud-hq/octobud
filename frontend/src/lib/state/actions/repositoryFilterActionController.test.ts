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

import { describe, it, expect, beforeEach, vi, type Mock } from "vitest";
import { get } from "svelte/store";
import { createRepositoryFilterActionController } from "./repositoryFilterActionController";
import { createRepositoryFilterStore } from "../../stores/repositoryFilterStore";
import { createRepositoryPinsStore } from "../../stores/repositoryPinsStore";
import { createPaginationStore } from "../../stores/paginationStore";
import { createQueryStore } from "../../stores/queryStore";
import type { ControllerOptions, NavigateOptions } from "../interfaces/common";
import type { SharedHelpers } from "./sharedHelpers";
import { updateViewRepositoryDefault } from "$lib/api/views";

vi.mock("$lib/api/views", () => ({
	updateViewRepositoryDefault: vi.fn(),
}));
vi.mock("$lib/stores/toastStore", () => ({
	toastStore: { success: vi.fn(), error: vi.fn() },
}));

describe("RepositoryFilterActionController", () => {
	let navigateToUrl: Mock<(url: string, options?: NavigateOptions) => Promise<void>>;
	let invalidateViews: Mock<() => Promise<void>>;
	let options: ControllerOptions;
	let sharedHelpers: SharedHelpers;

	function build(
		initialIds: number[] = [],
		viewQuery = "in:inbox",
		quickQuery = viewQuery,
		countsQuery: string | null = null
	) {
		const repositoryFilterStore = createRepositoryFilterStore(initialIds, [], countsQuery);
		const pinsStore = createRepositoryPinsStore(null);
		const paginationStore = createPaginationStore(3, {
			items: [],
			total: 100,
			page: 3,
			pageSize: 30,
		});
		const queryStore = createQueryStore(quickQuery, viewQuery);
		const controller = createRepositoryFilterActionController(
			repositoryFilterStore,
			pinsStore,
			queryStore,
			paginationStore,
			options,
			sharedHelpers
		);
		return { controller, repositoryFilterStore, pinsStore, paginationStore, queryStore };
	}

	beforeEach(() => {
		global.window = {
			location: { href: "http://localhost:3000/views/inbox?page=3&id=abc" },
		} as any;
		vi.mocked(updateViewRepositoryDefault).mockReset();
		navigateToUrl = vi.fn<(url: string, options?: NavigateOptions) => Promise<void>>();
		navigateToUrl.mockResolvedValue(undefined);
		invalidateViews = vi.fn<() => Promise<void>>().mockResolvedValue(undefined);
		options = { navigateToUrl, invalidateViews };
		sharedHelpers = {
			refresh: vi.fn().mockResolvedValue(undefined),
			refreshRepositoryCounts: vi.fn().mockResolvedValue(undefined),
			syncQueryToUrl: vi.fn(),
			scheduleDebouncedRefresh: vi.fn(),
			updateUrlWithDetailId: vi.fn(),
			updateUrlWithoutDetailId: vi.fn(),
		} as unknown as SharedHelpers;
	});

	it("setRepositoryFilter writes ?repos=, resets the page and keeps the detail id", async () => {
		const { controller, repositoryFilterStore, paginationStore } = build();

		await controller.setRepositoryFilter([12, 34]);

		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([12, 34]);
		expect(get(paginationStore.page)).toBe(1);
		expect(navigateToUrl).toHaveBeenCalledTimes(1);
		const [url, navOptions] = navigateToUrl.mock.calls[0];
		const parsed = new URL(url, "http://localhost:3000");
		expect(parsed.pathname).toBe("/views/inbox");
		expect(parsed.searchParams.get("repos")).toBe("12,34");
		expect(parsed.searchParams.get("page")).toBeNull();
		expect(parsed.searchParams.get("id")).toBe("abc");
		expect(parsed.searchParams.get("query")).toBeNull();
		expect(navOptions).toEqual({ replace: false, invalidateViews: false });
	});

	it("preserves a modified quick query in the URL", async () => {
		const { controller } = build([], "in:inbox", "in:inbox is:unread");

		await controller.selectOnlyRepository(5);

		const parsed = new URL(navigateToUrl.mock.calls[0][0], "http://localhost:3000");
		expect(parsed.searchParams.get("query")).toBe("in:inbox is:unread");
		expect(parsed.searchParams.get("repos")).toBe("5");
	});

	it("setRepositoryFilter is a no-op when the selection is unchanged as a set", async () => {
		const { controller } = build([1, 2]);

		await controller.setRepositoryFilter([2, 1]);

		expect(navigateToUrl).not.toHaveBeenCalled();
	});

	it("toggleRepositoryInFilter adds and removes ids", async () => {
		const { controller, repositoryFilterStore } = build([1]);

		await controller.toggleRepositoryInFilter(2);
		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([1, 2]);

		await controller.toggleRepositoryInFilter(1);
		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([2]);

		// Removing the last id returns to "all repositories" and drops the param.
		await controller.toggleRepositoryInFilter(2);
		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([]);
		const parsed = new URL(navigateToUrl.mock.calls[2][0], "http://localhost:3000");
		expect(parsed.searchParams.has("repos")).toBe(false);
	});

	it("clearRepositoryFilter reports whether anything was cleared", async () => {
		const { controller } = build([9]);
		expect(await controller.clearRepositoryFilter()).toBe(true);
		expect(navigateToUrl).toHaveBeenCalledTimes(1);

		const empty = build([]);
		expect(await empty.controller.clearRepositoryFilter()).toBe(false);
		expect(navigateToUrl).toHaveBeenCalledTimes(1);
	});

	it("openRepositoryFilter opens once and refreshes counts only when they are stale", () => {
		const { controller, repositoryFilterStore } = build();

		expect(controller.openRepositoryFilter()).toBe(true);
		expect(get(repositoryFilterStore.dropdownOpen)).toBe(true);
		expect(sharedHelpers.refreshRepositoryCounts).toHaveBeenCalledTimes(1);

		expect(controller.openRepositoryFilter()).toBe(false);
		expect(sharedHelpers.refreshRepositoryCounts).toHaveBeenCalledTimes(1);

		controller.closeRepositoryFilter();
		expect(get(repositoryFilterStore.dropdownOpen)).toBe(false);

		// Counts already loaded for the current query (by the route loader): no refetch.
		const fresh = build([], "in:inbox", "in:inbox", "in:inbox");
		expect(fresh.controller.openRepositoryFilter()).toBe(true);
		expect(sharedHelpers.refreshRepositoryCounts).toHaveBeenCalledTimes(1);

		// Quick query edited since the counts were fetched: refetch.
		const stale = build([], "in:inbox", "in:inbox is:unread", "in:inbox");
		expect(stale.controller.openRepositoryFilter()).toBe(true);
		expect(sharedHelpers.refreshRepositoryCounts).toHaveBeenCalledTimes(2);
	});

	it("toggleRepositoryPin delegates to the pins store and fetches an unknown pin's identity", () => {
		const { controller, pinsStore } = build();
		controller.toggleRepositoryPin(4);
		expect(pinsStore.isPinned(4)).toBe(true);
		// Repo 4 is not in the (empty) counts, so its identity must be fetched.
		expect(sharedHelpers.refreshRepositoryCounts).toHaveBeenCalledTimes(1);
		controller.toggleRepositoryPin(4);
		expect(pinsStore.isPinned(4)).toBe(false);
		expect(sharedHelpers.refreshRepositoryCounts).toHaveBeenCalledTimes(1);
	});

	it("rolls the optimistic selection back when navigation fails, without throwing", async () => {
		navigateToUrl.mockRejectedValueOnce(new Error("cancelled"));
		const { controller, repositoryFilterStore, paginationStore } = build([1]);

		await controller.setRepositoryFilter([2]);

		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([1]);
		expect(get(paginationStore.page)).toBe(3);

		// The same selection can be applied again afterwards.
		await controller.setRepositoryFilter([2]);
		expect(navigateToUrl).toHaveBeenCalledTimes(2);
		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([2]);
	});

	it("does not roll back a superseded navigation over a newer selection", async () => {
		let rejectFirst: (error: Error) => void = () => {};
		navigateToUrl.mockImplementationOnce(
			() =>
				new Promise<void>((_, reject) => {
					rejectFirst = reject;
				})
		);
		const { controller, repositoryFilterStore } = build([]);

		const first = controller.toggleRepositoryInFilter(1);
		const second = controller.toggleRepositoryInFilter(2); // aborts the first navigation
		rejectFirst(new Error("navigation aborted"));
		await Promise.all([first, second]);

		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([1, 2]);
	});

	it("setViewRepositoryDefault ignores the response if the view changed meanwhile", async () => {
		let resolveSave: (ids: number[]) => void = () => {};
		vi.mocked(updateViewRepositoryDefault).mockImplementationOnce(
			() =>
				new Promise<number[]>((resolve) => {
					resolveSave = resolve;
				})
		);
		const { controller, repositoryFilterStore } = build([3]);
		repositoryFilterStore.setViewDefault("view-a", []);

		const saving = controller.setViewRepositoryDefault();
		repositoryFilterStore.setViewDefault("view-b", [9]); // user switched views
		resolveSave([3]);
		await saving;

		expect(get(repositoryFilterStore.viewDefaultRepositoryIds)).toEqual([9]);
		expect(navigateToUrl).not.toHaveBeenCalled();
	});

	it("falls back to an in-place refresh without a router", async () => {
		options = {};
		const { controller, repositoryFilterStore, paginationStore } = build();

		await controller.setRepositoryFilter([7]);

		expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([7]);
		expect(get(paginationStore.page)).toBe(1);
		expect(sharedHelpers.refresh).toHaveBeenCalledTimes(1);
	});

	describe("view defaults", () => {
		it("drops ?repos= when the selection equals the view default and writes 'all' when clearing it", async () => {
			const { controller, repositoryFilterStore } = build([9]);
			repositoryFilterStore.setViewDefault("inbox", [1, 2]);

			await controller.setRepositoryFilter([2, 1]);
			let parsed = new URL(navigateToUrl.mock.calls[0][0], "http://localhost:3000");
			expect(parsed.searchParams.has("repos")).toBe(false);

			await controller.clearRepositoryFilter();
			parsed = new URL(navigateToUrl.mock.calls[1][0], "http://localhost:3000");
			expect(parsed.searchParams.get("repos")).toBe("all");

			await controller.setRepositoryFilter([5]);
			parsed = new URL(navigateToUrl.mock.calls[2][0], "http://localhost:3000");
			expect(parsed.searchParams.get("repos")).toBe("5");
		});

		it("resetToViewDefault navigates to the stored default", async () => {
			const { controller, repositoryFilterStore } = build([9]);
			repositoryFilterStore.setViewDefault("inbox", [4]);

			await controller.resetToViewDefault();

			expect(get(repositoryFilterStore.selectedRepositoryIds)).toEqual([4]);
			const parsed = new URL(navigateToUrl.mock.calls[0][0], "http://localhost:3000");
			expect(parsed.searchParams.has("repos")).toBe(false);
		});

		it("setViewRepositoryDefault saves the selection under the view key and updates the store", async () => {
			vi.mocked(updateViewRepositoryDefault).mockResolvedValue([3, 4]);
			const { controller, repositoryFilterStore } = build([3, 4]);
			repositoryFilterStore.setViewDefault("reviews-id", []);

			await controller.setViewRepositoryDefault();

			expect(updateViewRepositoryDefault).toHaveBeenCalledWith("reviews-id", [3, 4]);
			expect(get(repositoryFilterStore.viewDefaultRepositoryIds)).toEqual([3, 4]);
			// Loader-owned view data is refreshed as part of the navigation itself.
			expect(navigateToUrl).toHaveBeenCalledWith(expect.any(String), {
				replace: false,
				invalidateViews: true,
			});
			expect(get(repositoryFilterStore.isViewDefaultSelection)).toBe(true);
			// Selection now equals the default: the URL drops the param.
			const parsed = new URL(navigateToUrl.mock.calls[0][0], "http://localhost:3000");
			expect(parsed.searchParams.has("repos")).toBe(false);
		});

		it("setViewRepositoryDefault is a no-op without a view key", async () => {
			const { controller } = build([3]);
			await controller.setViewRepositoryDefault();
			expect(updateViewRepositoryDefault).not.toHaveBeenCalled();
		});
	});
});
