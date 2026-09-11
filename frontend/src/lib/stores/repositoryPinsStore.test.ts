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
import { createRepositoryPinsStore, readPinnedRepositoryIds } from "./repositoryPinsStore";

const KEY = "octobud:repositories:pinned";

describe("repositoryPinsStore", () => {
	beforeEach(() => {
		localStorage.clear();
	});

	it("starts empty when nothing is stored", () => {
		const store = createRepositoryPinsStore(localStorage);
		expect(get(store.pinnedRepositoryIds)).toEqual([]);
	});

	it("toggles pins in pin order and persists ids", () => {
		const store = createRepositoryPinsStore(localStorage);
		expect(store.togglePin(2)).toBe(true);
		expect(store.togglePin(1)).toBe(true);
		expect(get(store.pinnedRepositoryIds)).toEqual([2, 1]);
		expect(store.isPinned(1)).toBe(true);
		expect(store.isPinned(3)).toBe(false);
		expect(JSON.parse(localStorage.getItem(KEY) ?? "[]")).toEqual([2, 1]);

		expect(store.togglePin(2)).toBe(false);
		expect(get(store.pinnedRepositoryIds)).toEqual([1]);
	});

	it("rehydrates ids, migrates the old object shape, and ignores junk", () => {
		localStorage.setItem(
			KEY,
			JSON.stringify([{ id: 1, fullName: "org/a" }, 3, "bad", { id: "x" }, 1, null, 0, -2])
		);
		expect(readPinnedRepositoryIds(localStorage)).toEqual([1, 3]);
		expect(get(createRepositoryPinsStore(localStorage).pinnedRepositoryIds)).toEqual([1, 3]);

		localStorage.setItem(KEY, "{not json");
		expect(readPinnedRepositoryIds(localStorage)).toEqual([]);
	});
});
