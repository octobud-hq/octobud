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
import { tick } from "svelte";
import { fetchNotifications, fetchRepositoryCounts } from "$lib/api/notifications";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { QueryStore } from "../../stores/queryStore";
import type { RepositoryFilterStore } from "../../stores/repositoryFilterStore";
import type { RepositoryPinsStore } from "../../stores/repositoryPinsStore";
import type { ControllerOptions } from "../interfaces/common";
import type { DebounceManager } from "./debounceManager";

export interface SharedHelpers {
	refresh: () => Promise<void>;
	/** Refetch per-repository counts for the current query (ignores the repository filter). */
	refreshRepositoryCounts: () => Promise<void>;
	syncQueryToUrl: () => Promise<void>;
	scheduleDebouncedRefresh: () => void;
	updateUrlWithDetailId: (notificationId: string) => Promise<void>;
	updateUrlWithoutDetailId: () => Promise<void>;
}

/**
 * Create shared action helpers used by multiple controllers
 */
export function createSharedHelpers(
	notificationStore: NotificationStore,
	paginationStore: PaginationStore,
	queryStore: QueryStore,
	options: ControllerOptions,
	debounceManager: DebounceManager,
	repositoryFilterStore?: RepositoryFilterStore,
	repositoryPinsStore?: RepositoryPinsStore
): SharedHelpers {
	// Sequence number so an older counts response never overwrites a newer one.
	let repositoryCountsRequest = 0;
	/**
	 * Refresh notifications from API
	 */
	async function refresh(): Promise<void> {
		paginationStore.setLoading(true);
		try {
			const currentPage = get(paginationStore.page);
			const currentQuery = get(queryStore.quickQuery);
			const repositoryIds = repositoryFilterStore
				? get(repositoryFilterStore.selectedRepositoryIds)
				: [];

			const response = await fetchNotifications({
				page: currentPage,
				...(repositoryIds.length > 0 ? { repositoryIds } : {}),
				filters: {
					query: currentQuery || undefined,
					filters: [],
				},
			});

			notificationStore.setPageData(response);
			paginationStore.setTotal(response.total);
			paginationStore.setPageSize(response.pageSize);
			options.onAfterRefresh?.();
		} catch (error) {
			console.error("Failed to refresh notifications", error);
		} finally {
			paginationStore.setLoading(false);
		}
	}

	/**
	 * Refresh the per-repository counts that back the repository selector. Uses the
	 * current query without the repository filter so every candidate repository is
	 * listed. Failures are logged and leave the previous counts in place.
	 */
	async function refreshRepositoryCounts(): Promise<void> {
		if (!repositoryFilterStore || typeof window === "undefined") {
			return;
		}
		const requestId = ++repositoryCountsRequest;
		const query = get(queryStore.quickQuery) ?? "";
		const includeIds = [
			...get(repositoryFilterStore.selectedRepositoryIds),
			...(repositoryPinsStore ? get(repositoryPinsStore.pinnedRepositoryIds) : []),
		];
		repositoryFilterStore.countsLoading.set(true);
		try {
			const counts = await fetchRepositoryCounts(query, includeIds);
			if (requestId === repositoryCountsRequest) {
				repositoryFilterStore.setRepositoryCounts(counts, query);
			}
		} catch (error) {
			console.error("Failed to refresh repository counts", error);
		} finally {
			if (requestId === repositoryCountsRequest) {
				repositoryFilterStore.countsLoading.set(false);
			}
		}
	}

	/**
	 * Sync query to URL
	 */
	async function syncQueryToUrl(): Promise<void> {
		if (!options.navigateToUrl || typeof window === "undefined") {
			return;
		}

		const url = new URL(window.location.href);
		const currentQuery = get(queryStore.quickQuery);
		const viewQueryValue = get(queryStore.viewQuery);

		// Only add query param if it differs from the view's base query
		if (currentQuery && currentQuery !== viewQueryValue) {
			url.searchParams.set("query", currentQuery);
		} else {
			// If query matches view query, remove the query param (use view default)
			url.searchParams.delete("query");
		}

		// Preserve page number
		const currentPage = get(paginationStore.page);
		if (currentPage > 1) {
			url.searchParams.set("page", String(currentPage));
		} else {
			url.searchParams.delete("page");
		}

		// Query sync replaces state (doesn't create new history entry)
		await options.navigateToUrl(url.pathname + url.search, { replace: true });
	}

	/**
	 * Schedule debounced refresh
	 */
	function scheduleDebouncedRefresh(): void {
		debounceManager.setSearchDebounce(() => {
			void refresh();
		});
	}

	/**
	 * Update URL with detail ID parameter (for optimistic updates, replaces state)
	 */
	async function updateUrlWithDetailId(notificationId: string): Promise<void> {
		await tick();
		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);
			url.searchParams.set("id", notificationId);
			await options.navigateToUrl(url.pathname + url.search, { replace: true });
		}
	}

	/**
	 * Update URL without detail ID parameter (for optimistic updates, replaces state)
	 */
	async function updateUrlWithoutDetailId(): Promise<void> {
		await tick();
		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);
			url.searchParams.delete("id");
			await options.navigateToUrl(url.pathname + url.search, { replace: true });
		}
	}

	return {
		refresh,
		refreshRepositoryCounts,
		syncQueryToUrl,
		scheduleDebouncedRefresh,
		updateUrlWithDetailId,
		updateUrlWithoutDetailId,
	};
}
