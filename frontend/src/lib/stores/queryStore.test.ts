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
import { createQueryStore } from "./queryStore";

describe("QueryStore", () => {
	describe("initialization", () => {
		it("initializes with empty values by default", () => {
			const store = createQueryStore();

			expect(get(store.searchTerm)).toBe("");
			expect(get(store.quickQuery)).toBe("");
			expect(get(store.viewQuery)).toBe("");
			expect(get(store.quickFilters)).toEqual([]);
		});

		it("initializes with provided initialQuery", () => {
			const store = createQueryStore("is:unread");

			expect(get(store.quickQuery)).toBe("is:unread");
		});

		it("initializes with provided initialViewQuery", () => {
			const store = createQueryStore("", "type:PullRequest");

			expect(get(store.quickQuery)).toBe("type:PullRequest");
			expect(get(store.viewQuery)).toBe("type:PullRequest");
		});

		it("prefers initialQuery over initialViewQuery for quickQuery", () => {
			const store = createQueryStore("is:unread", "type:PullRequest");

			expect(get(store.quickQuery)).toBe("is:unread");
			expect(get(store.viewQuery)).toBe("type:PullRequest");
		});
	});

	describe("derived stores", () => {
		it("isQueryModified is false when quickQuery equals viewQuery", () => {
			const store = createQueryStore("is:unread", "is:unread");

			expect(get(store.isQueryModified)).toBe(false);
		});

		it("isQueryModified is true when quickQuery differs from viewQuery", () => {
			const store = createQueryStore("is:unread type:Issue", "is:unread");

			expect(get(store.isQueryModified)).toBe(true);
		});

		it("isQueryModified ignores whitespace differences", () => {
			const store = createQueryStore("  is:unread  ", "is:unread");

			expect(get(store.isQueryModified)).toBe(false);
		});

		it("hasActiveFilters is false when queries match", () => {
			const store = createQueryStore("is:unread", "is:unread");

			expect(get(store.hasActiveFilters)).toBe(false);
		});

		it("hasActiveFilters is true when queries differ and quickQuery is not empty", () => {
			const store = createQueryStore("is:unread type:Issue", "is:unread");

			expect(get(store.hasActiveFilters)).toBe(true);
		});

		it("hasActiveFilters is false when quickQuery is empty", () => {
			const store = createQueryStore("", "is:unread");

			expect(get(store.hasActiveFilters)).toBe(false);
		});
	});

	describe("actions", () => {
		it("setSearchTerm updates search term", () => {
			const store = createQueryStore();

			store.setSearchTerm("test search");

			expect(get(store.searchTerm)).toBe("test search");
		});

		it("setQuickQuery updates quick query", () => {
			const store = createQueryStore();

			store.setQuickQuery("is:starred");

			expect(get(store.quickQuery)).toBe("is:starred");
		});

		it("setViewQuery updates view query", () => {
			const store = createQueryStore();

			store.setViewQuery("type:PullRequest");

			expect(get(store.viewQuery)).toBe("type:PullRequest");
		});

		it("addQuickFilter adds a new empty filter", () => {
			const store = createQueryStore();

			store.addQuickFilter();

			const filters = get(store.quickFilters);
			expect(filters).toHaveLength(1);
			expect(filters[0]).toEqual({ field: "", operator: "equals", value: "" });
		});

		it("addQuickFilter appends to existing filters", () => {
			const store = createQueryStore();

			store.addQuickFilter();
			store.addQuickFilter();

			expect(get(store.quickFilters)).toHaveLength(2);
		});

		it("updateQuickFilter updates filter at index", () => {
			const store = createQueryStore();

			store.addQuickFilter();
			store.updateQuickFilter(0, { field: "repo", value: "test/repo" });

			const filters = get(store.quickFilters);
			expect(filters[0]).toEqual({ field: "repo", operator: "equals", value: "test/repo" });
		});

		it("updateQuickFilter only updates specified filter", () => {
			const store = createQueryStore();

			store.addQuickFilter();
			store.addQuickFilter();
			store.updateQuickFilter(0, { field: "repo" });

			const filters = get(store.quickFilters);
			expect(filters[0].field).toBe("repo");
			expect(filters[1].field).toBe("");
		});

		it("removeQuickFilter removes filter at index", () => {
			const store = createQueryStore();

			store.addQuickFilter();
			store.addQuickFilter();
			store.updateQuickFilter(0, { field: "repo" });
			store.updateQuickFilter(1, { field: "type" });

			store.removeQuickFilter(0);

			const filters = get(store.quickFilters);
			expect(filters).toHaveLength(1);
			expect(filters[0].field).toBe("type");
		});

		it("clearFilters resets all filter state", () => {
			const store = createQueryStore("is:unread type:Issue", "is:unread");

			store.setSearchTerm("test");
			store.addQuickFilter();

			store.clearFilters();

			expect(get(store.searchTerm)).toBe("");
			expect(get(store.quickQuery)).toBe("is:unread"); // Reset to viewQuery
			expect(get(store.quickFilters)).toEqual([]);
		});
	});

	describe("derived values update reactively", () => {
		it("isQueryModified updates when quickQuery changes", () => {
			const store = createQueryStore("is:unread", "is:unread");

			expect(get(store.isQueryModified)).toBe(false);

			store.setQuickQuery("is:unread type:Issue");

			expect(get(store.isQueryModified)).toBe(true);
		});

		it("hasActiveFilters updates when quickQuery changes", () => {
			const store = createQueryStore("is:unread", "is:unread");

			expect(get(store.hasActiveFilters)).toBe(false);

			store.setQuickQuery("is:unread type:Issue");

			expect(get(store.hasActiveFilters)).toBe(true);
		});
	});
});
