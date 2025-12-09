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
import { fetchNotifications } from "$lib/api/notifications";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { QueryStore } from "../../stores/queryStore";
import type { ControllerOptions } from "../interfaces/common";
import type { DebounceManager } from "./debounceManager";

export interface SharedHelpers {
	refresh: () => Promise<void>;
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
	debounceManager: DebounceManager
): SharedHelpers {
	/**
	 * Refresh notifications from API
	 */
	async function refresh(): Promise<void> {
		paginationStore.setLoading(true);
		try {
			const currentPage = get(paginationStore.page);
			const currentQuery = get(queryStore.quickQuery);

			const response = await fetchNotifications({
				page: currentPage,
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
		syncQueryToUrl,
		scheduleDebouncedRefresh,
		updateUrlWithDetailId,
		updateUrlWithoutDetailId,
	};
}
