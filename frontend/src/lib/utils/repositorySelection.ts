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

import type { RepositoryCount } from "$lib/api/types";
import { normalizeRepositoryIds } from "$lib/stores/repositoryFilterStore";

/** One row in the repository selector dropdown. */
export interface RepositoryOption {
	id: number;
	fullName: string;
	ownerAvatarUrl: string | null;
	total: number;
	unread: number;
	pinned: boolean;
	selected: boolean;
	/** 1-based keyboard row index (row 0 is "All repositories"). */
	row: number;
}

export interface RepositoryOptionGroups {
	pinned: RepositoryOption[];
	others: RepositoryOption[];
	/** Flat list in display order (pinned first), for rendering and keyboard navigation. */
	all: RepositoryOption[];
}

/**
 * Build the dropdown rows from the server's counts: pinned repositories first in pin
 * order (the server includes pinned repositories even at zero matches, so pins never
 * jump around), then the rest in the server's order (unread, then total, then name).
 * `filterText` narrows both groups by substring. A pinned id absent from the counts
 * (deleted or unknown repository) is skipped.
 */
export function buildRepositoryOptions(
	counts: readonly RepositoryCount[],
	selectedIds: readonly number[],
	pinnedIds: readonly number[],
	filterText = ""
): RepositoryOptionGroups {
	const selected = new Set(selectedIds);
	const pinnedSet = new Set(pinnedIds);
	const countsById = new Map(counts.map((count) => [count.repository.id, count]));
	const needle = filterText.trim().toLowerCase();

	const toOption = (count: RepositoryCount, row: number): RepositoryOption => ({
		id: count.repository.id,
		fullName: count.repository.fullName || count.repository.name,
		ownerAvatarUrl: count.repository.ownerAvatarUrl ?? null,
		total: count.total,
		unread: count.unread,
		pinned: pinnedSet.has(count.repository.id),
		selected: selected.has(count.repository.id),
		row,
	});
	const matches = (count: RepositoryCount) =>
		needle === "" ||
		(count.repository.fullName || count.repository.name).toLowerCase().includes(needle);

	const all: RepositoryOption[] = [];
	for (const id of pinnedIds) {
		const count = countsById.get(id);
		if (count && matches(count)) all.push(toOption(count, all.length + 1));
	}
	const pinnedCount = all.length;
	for (const count of counts) {
		if (pinnedSet.has(count.repository.id) || !matches(count)) continue;
		all.push(toOption(count, all.length + 1));
	}

	return { pinned: all.slice(0, pinnedCount), others: all.slice(pinnedCount), all };
}

/** URL value meaning "explicitly all repositories" on a view that has a default selection. */
export const ALL_REPOSITORIES_PARAM = "all";

/**
 * Resolve the effective repository selection from the `?repos=` URL value and the
 * view's stored default. No URL value means "use the view default"; `all` means the
 * user explicitly cleared it; anything else is an explicit id list.
 */
export function resolveRepositorySelection(
	raw: string | null,
	viewDefaultIds: readonly number[]
): number[] {
	const value = raw?.trim() ?? "";
	if (value === "") {
		return [...viewDefaultIds];
	}
	if (value.toLowerCase() === ALL_REPOSITORIES_PARAM) {
		return [];
	}
	return normalizeRepositoryIds(value.split(",").map((part) => Number.parseInt(part.trim(), 10)));
}

/** Total unread across the counted repositories. */
export function totalUnread(counts: readonly RepositoryCount[]): number {
	return counts.reduce((sum, count) => sum + count.unread, 0);
}

/** Unread notifications in repositories that are not part of the current selection. */
export function unreadOutsideSelection(
	counts: readonly RepositoryCount[],
	selectedIds: readonly number[]
): number {
	if (selectedIds.length === 0) return 0;
	const selected = new Set(selectedIds);
	return totalUnread(counts.filter((count) => !selected.has(count.repository.id)));
}

export interface RepositorySelectionLabel {
	/** Names to render, at most `maxNames`. Empty means "all repositories". */
	names: string[];
	/** How many selected repositories are not represented in `names`. */
	overflow: number;
}

/**
 * Compact label for the selector button: the first `maxNames` selected repositories
 * plus an overflow count for the rest.
 */
export function formatRepositorySelection(
	selected: readonly { fullName: string }[],
	maxNames = 2
): RepositorySelectionLabel {
	const names = selected.slice(0, Math.max(1, maxNames)).map((repo) => repo.fullName);
	return { names, overflow: Math.max(0, selected.length - names.length) };
}
