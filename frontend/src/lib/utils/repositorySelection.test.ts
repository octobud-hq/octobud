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
import { buildRepositoryOptions, formatRepositorySelection } from "./repositorySelection";
import type { RepositoryCount } from "$lib/api/types";

const count = (
	id: number,
	fullName: string,
	total: number,
	unread: number,
	avatar: string | null = null
): RepositoryCount => ({
	repository: { id, name: fullName.split("/")[1] ?? fullName, fullName, ownerAvatarUrl: avatar },
	total,
	unread,
});

describe("buildRepositoryOptions", () => {
	const counts = [
		count(1, "microsoft/vscode", 61, 12, "https://avatars/ms"),
		count(2, "octobud-hq/octobud", 48, 3),
		count(3, "golang/go", 22, 0),
		count(9, "org/quiet", 0, 0), // included by the server because it is pinned
	];

	it("puts pins first in pin order, keeps server order for the rest, and numbers rows", () => {
		const groups = buildRepositoryOptions(counts, [3], [3, 1, 9]);
		expect(groups.pinned.map((o) => o.id)).toEqual([3, 1, 9]);
		expect(groups.others.map((o) => o.id)).toEqual([2]);
		expect(groups.all.map((o) => o.row)).toEqual([1, 2, 3, 4]);
		expect(groups.pinned[0].selected).toBe(true);
		expect(groups.pinned[1]).toMatchObject({
			pinned: true,
			selected: false,
			ownerAvatarUrl: "https://avatars/ms",
		});
		expect(groups.pinned[2]).toMatchObject({ total: 0, unread: 0, pinned: true });
	});

	it("skips pinned ids the server did not return", () => {
		const groups = buildRepositoryOptions(counts, [], [42, 2]);
		expect(groups.pinned.map((o) => o.id)).toEqual([2]);
		expect(groups.others.map((o) => o.id)).toEqual([1, 3, 9]);
	});

	it("filters both groups by case-insensitive substring and renumbers rows", () => {
		const groups = buildRepositoryOptions(counts, [], [3], "GO");
		expect(groups.pinned.map((o) => o.fullName)).toEqual(["golang/go"]);
		expect(groups.others).toEqual([]);

		const octo = buildRepositoryOptions(counts, [], [], "octo");
		expect(octo.all.map((o) => [o.fullName, o.row])).toEqual([["octobud-hq/octobud", 1]]);
	});
});

describe("formatRepositorySelection", () => {
	it("returns no names for an empty selection", () => {
		expect(formatRepositorySelection([])).toEqual({ names: [], overflow: 0 });
	});

	it("shows up to maxNames and counts the overflow", () => {
		const selected = [{ fullName: "a/a" }, { fullName: "b/b" }, { fullName: "c/c" }];
		expect(formatRepositorySelection(selected)).toEqual({ names: ["a/a", "b/b"], overflow: 1 });
		expect(formatRepositorySelection(selected, 1)).toEqual({ names: ["a/a"], overflow: 2 });
		expect(formatRepositorySelection(selected, 5)).toEqual({
			names: ["a/a", "b/b", "c/c"],
			overflow: 0,
		});
	});
});
