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

import { describe, it, expect } from "vitest";
import { get } from "svelte/store";
import {
	createRepositoryFilterStore,
	normalizeRepositoryIds,
	sameRepositoryIds,
} from "./repositoryFilterStore";
import type { RepositoryCount } from "$lib/api/types";

const count = (id: number, fullName: string, total = 1, unread = 0): RepositoryCount => ({
	repository: { id, name: fullName.split("/")[1] ?? fullName, fullName, ownerAvatarUrl: null },
	total,
	unread,
});

describe("normalizeRepositoryIds", () => {
	it("drops invalid and duplicate ids while preserving order", () => {
		expect(normalizeRepositoryIds([3, 0, -1, 3, 1.5, 7, NaN])).toEqual([3, 7]);
	});
});

describe("sameRepositoryIds", () => {
	it("compares as sets", () => {
		expect(sameRepositoryIds([1, 2], [2, 1])).toBe(true);
		expect(sameRepositoryIds([1, 2], [1])).toBe(false);
		expect(sameRepositoryIds([], [])).toBe(true);
	});
});

describe("repositoryFilterStore", () => {
	it("initialises from ids and counts, labelling selected repositories from the counts", () => {
		const store = createRepositoryFilterStore([2, 2, 9], [count(2, "org/b", 4, 1)], "in:inbox");
		expect(get(store.selectedRepositoryIds)).toEqual([2, 9]);
		expect(get(store.hasRepositoryFilter)).toBe(true);
		expect(get(store.countsQuery)).toBe("in:inbox");
		expect(get(store.selectedRepositories)).toEqual([
			{ id: 2, fullName: "org/b", ownerAvatarUrl: null },
			{ id: 9, fullName: "Repository 9", ownerAvatarUrl: null },
		]);
	});

	it("does not emit when the selection is unchanged as a set", () => {
		const store = createRepositoryFilterStore([1, 2]);
		let emissions = 0;
		const unsubscribe = store.selectedRepositoryIds.subscribe(() => {
			emissions += 1;
		});
		store.setSelectedRepositoryIds([2, 1]);
		expect(emissions).toBe(1);
		store.setSelectedRepositoryIds([2]);
		expect(emissions).toBe(2);
		unsubscribe();
	});

	it("setRepositoryCounts records the query the counts belong to", () => {
		const store = createRepositoryFilterStore([5]);
		store.setRepositoryCounts([count(5, "org/e", 0, 0)], "is:unread");
		expect(get(store.countsQuery)).toBe("is:unread");
		expect(get(store.selectedRepositories)[0].fullName).toBe("org/e");
	});

	it("dropdown open state", () => {
		const store = createRepositoryFilterStore();
		expect(get(store.dropdownOpen)).toBe(false);
		store.openDropdown();
		expect(get(store.dropdownOpen)).toBe(true);
		store.closeDropdown();
		expect(get(store.dropdownOpen)).toBe(false);
	});

	it("tracks the view default and derives default/outside-unread indicators", () => {
		const store = createRepositoryFilterStore(
			[1],
			[count(1, "org/a", 5, 2), count(2, "org/b", 4, 3), count(3, "org/c", 1, 1)],
			"in:inbox"
		);
		expect(get(store.hasViewDefault)).toBe(false);
		expect(get(store.isViewDefaultSelection)).toBe(false);
		expect(get(store.unreadOutside)).toBe(4);
		expect(get(store.totalUnread)).toBe(6);

		store.setViewDefault("inbox", [1, 1, 0]);
		expect(get(store.viewKey)).toBe("inbox");
		expect(get(store.viewDefaultRepositoryIds)).toEqual([1]);
		expect(get(store.hasViewDefault)).toBe(true);
		expect(get(store.isViewDefaultSelection)).toBe(true);

		store.setSelectedRepositoryIds([]);
		expect(get(store.isViewDefaultSelection)).toBe(false);
		expect(get(store.unreadOutside)).toBe(0);
	});
});
