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

import { describe, it, expect, beforeEach } from "vitest";
import { get } from "svelte/store";
import { createPaginationStore } from "./paginationStore";
import type { NotificationPage } from "$lib/api/types";

describe("PaginationStore", () => {
	const createMockPageData = (overrides: Partial<NotificationPage> = {}): NotificationPage => ({
		items: [],
		total: 100,
		page: 1,
		pageSize: 25,
		...overrides,
	});

	describe("initialization", () => {
		it("initializes with provided values", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));

			expect(get(store.page)).toBe(1);
			expect(get(store.total)).toBe(100);
			expect(get(store.pageSize)).toBe(25);
			expect(get(store.isLoading)).toBe(false);
		});

		it("initializes with different page number", () => {
			const store = createPaginationStore(5, createMockPageData());

			expect(get(store.page)).toBe(5);
		});
	});

	describe("derived stores", () => {
		it("calculates totalPages correctly", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));
			expect(get(store.totalPages)).toBe(4);
		});

		it("calculates totalPages with partial last page", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 101, pageSize: 25 }));
			expect(get(store.totalPages)).toBe(5);
		});

		it("returns minimum 1 for totalPages when empty", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 0, pageSize: 25 }));
			expect(get(store.totalPages)).toBe(1);
		});

		it("calculates pageRangeStart correctly for first page", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));
			expect(get(store.pageRangeStart)).toBe(1);
		});

		it("calculates pageRangeStart correctly for second page", () => {
			const store = createPaginationStore(2, createMockPageData({ total: 100, pageSize: 25 }));
			expect(get(store.pageRangeStart)).toBe(26);
		});

		it("returns 0 for pageRangeStart when empty", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 0, pageSize: 25 }));
			expect(get(store.pageRangeStart)).toBe(0);
		});

		it("calculates pageRangeEnd correctly for first page", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));
			expect(get(store.pageRangeEnd)).toBe(25);
		});

		it("calculates pageRangeEnd correctly for last partial page", () => {
			const store = createPaginationStore(5, createMockPageData({ total: 101, pageSize: 25 }));
			expect(get(store.pageRangeEnd)).toBe(101);
		});

		it("returns 0 for pageRangeEnd when empty", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 0, pageSize: 25 }));
			expect(get(store.pageRangeEnd)).toBe(0);
		});
	});

	describe("actions", () => {
		it("setPage updates the current page", () => {
			const store = createPaginationStore(1, createMockPageData());

			store.setPage(3);

			expect(get(store.page)).toBe(3);
		});

		it("setTotal updates the total count", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100 }));

			store.setTotal(200);

			expect(get(store.total)).toBe(200);
		});

		it("setPageSize updates the page size", () => {
			const store = createPaginationStore(1, createMockPageData({ pageSize: 25 }));

			store.setPageSize(50);

			expect(get(store.pageSize)).toBe(50);
		});

		it("setLoading updates the loading state", () => {
			const store = createPaginationStore(1, createMockPageData());

			store.setLoading(true);
			expect(get(store.isLoading)).toBe(true);

			store.setLoading(false);
			expect(get(store.isLoading)).toBe(false);
		});
	});

	describe("derived values update reactively", () => {
		it("totalPages updates when total changes", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));

			expect(get(store.totalPages)).toBe(4);

			store.setTotal(50);

			expect(get(store.totalPages)).toBe(2);
		});

		it("totalPages updates when pageSize changes", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));

			expect(get(store.totalPages)).toBe(4);

			store.setPageSize(50);

			expect(get(store.totalPages)).toBe(2);
		});

		it("pageRangeStart updates when page changes", () => {
			const store = createPaginationStore(1, createMockPageData({ total: 100, pageSize: 25 }));

			expect(get(store.pageRangeStart)).toBe(1);

			store.setPage(2);

			expect(get(store.pageRangeStart)).toBe(26);
		});
	});
});
