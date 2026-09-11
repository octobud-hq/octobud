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

import { writable, derived, get, type Writable, type Readable } from "svelte/store";
import type { RepositoryCount } from "$lib/api/types";

/** Minimal repository identity used for selector labels. */
export interface SelectedRepository {
	id: number;
	fullName: string;
	ownerAvatarUrl: string | null;
}

export function normalizeRepositoryIds(ids: readonly number[]): number[] {
	const seen = new Set<number>();
	const result: number[] = [];
	for (const id of ids) {
		if (!Number.isInteger(id) || id <= 0 || seen.has(id)) continue;
		seen.add(id);
		result.push(id);
	}
	return result;
}

export function sameRepositoryIds(a: readonly number[], b: readonly number[]): boolean {
	if (a.length !== b.length) return false;
	const setB = new Set(b);
	return a.every((id) => setB.has(id));
}

/**
 * Repository Filter Store
 *
 * Holds the list's repository selection (a set of repository IDs applied on top of the
 * current query), the per-repository counts that back the selector dropdown (which the
 * server guarantees include every selected and pinned repository, so identity always
 * comes from the server), and the dropdown's open state. The selection is URL state
 * (`?repos=`); this store mirrors it.
 */
export function createRepositoryFilterStore(
	initialIds: readonly number[] = [],
	initialCounts: RepositoryCount[] = [],
	initialCountsQuery: string | null = null
) {
	const selectedRepositoryIds = writable<number[]>(normalizeRepositoryIds(initialIds));
	const repositoryCounts = writable<RepositoryCount[]>(initialCounts);
	// The query the current counts were fetched for, so callers can skip redundant refetches.
	const countsQuery = writable<string | null>(initialCountsQuery);
	const countsLoading = writable<boolean>(false);
	const dropdownOpen = writable<boolean>(false);

	const hasRepositoryFilter = derived(selectedRepositoryIds, ($ids) => $ids.length > 0);

	const selectedRepositories = derived(
		[selectedRepositoryIds, repositoryCounts],
		([$ids, $counts]): SelectedRepository[] =>
			$ids.map((id) => {
				const match = $counts.find((count) => count.repository.id === id);
				// Only reachable before the first counts payload arrives.
				return match
					? toSelectedRepository(match)
					: { id, fullName: `Repository ${id}`, ownerAvatarUrl: null };
			})
	);

	function setSelectedRepositoryIds(ids: readonly number[]): void {
		const next = normalizeRepositoryIds(ids);
		if (sameRepositoryIds(next, get(selectedRepositoryIds))) return;
		selectedRepositoryIds.set(next);
	}

	function setRepositoryCounts(counts: RepositoryCount[], query: string | null): void {
		repositoryCounts.set(counts);
		countsQuery.set(query);
	}

	return {
		selectedRepositoryIds: selectedRepositoryIds as Readable<number[]>,
		repositoryCounts: repositoryCounts as Readable<RepositoryCount[]>,
		countsQuery: countsQuery as Readable<string | null>,
		countsLoading: countsLoading as Writable<boolean>,
		dropdownOpen: dropdownOpen as Writable<boolean>,
		hasRepositoryFilter,
		selectedRepositories,

		setSelectedRepositoryIds,
		setRepositoryCounts,
		openDropdown: () => dropdownOpen.set(true),
		closeDropdown: () => dropdownOpen.set(false),
	};
}

function toSelectedRepository(count: RepositoryCount): SelectedRepository {
	return {
		id: count.repository.id,
		fullName: count.repository.fullName || count.repository.name,
		ownerAvatarUrl: count.repository.ownerAvatarUrl ?? null,
	};
}

export type RepositoryFilterStore = ReturnType<typeof createRepositoryFilterStore>;
