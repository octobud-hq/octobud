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
import { updateViewRepositoryDefault } from "$lib/api/views";
import { toastStore } from "$lib/stores/toastStore";
import { ALL_REPOSITORIES_PARAM } from "$lib/utils/repositorySelection";
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
 *
 * Views can store a default selection. No `?repos=` means "use the view default", so a
 * selection equal to the default drops the param, and clearing a view that has a default
 * writes `?repos=all` so the loader does not snap back to the default.
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

		const viewDefault = get(repositoryFilterStore.viewDefaultRepositoryIds);
		if (sameRepositoryIds(repositoryIds, viewDefault)) {
			url.searchParams.delete("repos");
		} else if (repositoryIds.length > 0) {
			url.searchParams.set("repos", repositoryIds.join(","));
		} else {
			url.searchParams.set("repos", ALL_REPOSITORIES_PARAM);
		}
		url.searchParams.delete("page");

		try {
			await options.navigateToUrl(url.pathname + url.search, { replace: false });
		} catch (error) {
			// Navigation rejected or was cancelled. Only roll back if nothing newer has been
			// requested since (a superseding selection navigation aborts this one and must
			// keep its own optimistic state). Callers fire-and-forget, so don't rethrow.
			if (sameRepositoryIds(get(repositoryFilterStore.selectedRepositoryIds), repositoryIds)) {
				repositoryFilterStore.setSelectedRepositoryIds(previousIds);
				paginationStore.setPage(previousPage);
				console.warn("Repository selection navigation failed; selection restored", error);
			}
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

	async function setViewRepositoryDefault(): Promise<void> {
		const viewKey = get(repositoryFilterStore.viewKey);
		if (!viewKey) return;
		const selection = get(repositoryFilterStore.selectedRepositoryIds);

		let stored: number[];
		try {
			stored = await updateViewRepositoryDefault(viewKey, selection);
		} catch (error) {
			console.error("Failed to save view repository default", error);
			toastStore.error("Failed to save the view's default repositories");
			return;
		}

		toastStore.success(
			stored.length === 0
				? "Cleared this view's default repositories"
				: `Saved ${stored.length} ${stored.length === 1 ? "repository" : "repositories"} as this view's default`
		);

		// The user may have switched views while the request was in flight; the saved
		// default belongs to the original view and must not be applied to the current one.
		if (get(repositoryFilterStore.viewKey) !== viewKey) return;
		repositoryFilterStore.setViewDefault(viewKey, stored);
		// The selection now equals the default, so the URL no longer needs ?repos=. The
		// navigation also reloads views so the stored default reaches the sidebar data.
		await navigateWithSelection(stored);
	}

	async function resetToViewDefault(): Promise<void> {
		await navigateWithSelection(get(repositoryFilterStore.viewDefaultRepositoryIds));
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
		setViewRepositoryDefault,
		resetToViewDefault,
	};
}
