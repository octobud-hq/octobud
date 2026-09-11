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

import { get } from "svelte/store";
import type { RepositoryFilterStore } from "../../stores/repositoryFilterStore";
import { sameRepositoryIds } from "../../stores/repositoryFilterStore";
import type { RepositoryPinsStore } from "../../stores/repositoryPinsStore";
import type { QueryStore } from "../../stores/queryStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { ControllerOptions } from "../interfaces/common";
import type { RepositoryFilterActions } from "../interfaces/repositoryFilterActions";
import type { SharedHelpers } from "./sharedHelpers";

/**
 * Repository Filter Action Controller
 *
 * The selection lives in the URL (`?repos=1,2`), so applying it is a navigation: the
 * route loader refetches the page and counts, and `syncFromData` mirrors the result back
 * into the store. Page resets to 1 on every change; the detail `?id=` is preserved so an
 * open reading pane stays open.
 */
export function createRepositoryFilterActionController(
	repositoryFilterStore: RepositoryFilterStore,
	pinsStore: RepositoryPinsStore,
	queryStore: QueryStore,
	paginationStore: PaginationStore,
	options: ControllerOptions,
	sharedHelpers: SharedHelpers
): RepositoryFilterActions {
	async function navigateWithSelection(repositoryIds: number[]): Promise<void> {
		const previousIds = get(repositoryFilterStore.selectedRepositoryIds);
		const previousPage = get(paginationStore.page);

		// Optimistically mirror so the selector label updates before the loader returns.
		repositoryFilterStore.setSelectedRepositoryIds(repositoryIds);
		paginationStore.setPage(1);

		if (!options.navigateToUrl || typeof window === "undefined") {
			// No router (tests): refresh in place.
			await sharedHelpers.refresh();
			return;
		}

		const url = new URL(window.location.href);

		// Keep the quick query in the URL the same way pagination does.
		const currentQuery = get(queryStore.quickQuery);
		const viewQueryValue = get(queryStore.viewQuery);
		if (currentQuery && currentQuery !== viewQueryValue) {
			url.searchParams.set("query", currentQuery);
		} else {
			url.searchParams.delete("query");
		}

		if (repositoryIds.length > 0) {
			url.searchParams.set("repos", repositoryIds.join(","));
		} else {
			url.searchParams.delete("repos");
		}
		url.searchParams.delete("page");

		try {
			await options.navigateToUrl(url.pathname + url.search, { replace: false });
		} catch (error) {
			// Navigation rejected or was cancelled: put the store back so the label and
			// URL agree, and so re-applying the same selection is not short-circuited.
			repositoryFilterStore.setSelectedRepositoryIds(previousIds);
			paginationStore.setPage(previousPage);
			throw error;
		}
	}

	async function setRepositoryFilter(repositoryIds: number[]): Promise<void> {
		const current = get(repositoryFilterStore.selectedRepositoryIds);
		if (sameRepositoryIds(current, repositoryIds)) return;
		await navigateWithSelection(repositoryIds);
	}

	async function toggleRepositoryInFilter(repositoryId: number): Promise<void> {
		const current = get(repositoryFilterStore.selectedRepositoryIds);
		const next = current.includes(repositoryId)
			? current.filter((id) => id !== repositoryId)
			: [...current, repositoryId];
		await navigateWithSelection(next);
	}

	async function selectOnlyRepository(repositoryId: number): Promise<void> {
		await setRepositoryFilter([repositoryId]);
	}

	async function clearRepositoryFilter(): Promise<boolean> {
		if (get(repositoryFilterStore.selectedRepositoryIds).length === 0) {
			return false;
		}
		await navigateWithSelection([]);
		return true;
	}

	function openRepositoryFilter(): boolean {
		if (get(repositoryFilterStore.dropdownOpen)) {
			return false;
		}
		repositoryFilterStore.openDropdown();
		// The route loader fetches counts for the current query on every navigation, so
		// only refetch here when the store's counts belong to a different query (e.g. the
		// quick query was edited without navigating) or have never been loaded.
		if (get(repositoryFilterStore.countsQuery) !== get(queryStore.quickQuery)) {
			void sharedHelpers.refreshRepositoryCounts();
		}
		return true;
	}

	function closeRepositoryFilter(): void {
		repositoryFilterStore.closeDropdown();
	}

	function toggleRepositoryPin(repositoryId: number): void {
		pinsStore.togglePin(repositoryId);
		// A newly pinned repository with zero matches is not in the counts yet; refetch so
		// it appears in the pinned section immediately. Unpinning needs nothing.
		if (
			pinsStore.isPinned(repositoryId) &&
			!get(repositoryFilterStore.repositoryCounts).some((c) => c.repository.id === repositoryId)
		) {
			void sharedHelpers.refreshRepositoryCounts();
		}
	}

	return {
		setRepositoryFilter,
		toggleRepositoryInFilter,
		selectOnlyRepository,
		clearRepositoryFilter,
		refreshRepositoryCounts: sharedHelpers.refreshRepositoryCounts,
		openRepositoryFilter,
		closeRepositoryFilter,
		toggleRepositoryPin,
	};
}
