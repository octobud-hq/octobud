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
import type { NotificationViewFilter } from "$lib/api/types";
import type { QueryActions } from "../interfaces/queryActions";
import type { QueryStore } from "../../stores/queryStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { SharedHelpers } from "./sharedHelpers";
import type { DebounceManager } from "./debounceManager";

/**
 * Query Action Controller
 * Manages search and filter state
 */
export function createQueryActionController(
	queryStore: QueryStore,
	paginationStore: PaginationStore,
	sharedHelpers: SharedHelpers,
	debounceManager: DebounceManager
): QueryActions {
	function handleSearchInput(value: string): void {
		queryStore.setSearchTerm(value);
		paginationStore.setPage(1);
		sharedHelpers.scheduleDebouncedRefresh();
	}

	function handleQuickQueryChange(value: string): void {
		queryStore.setQuickQuery(value);
		paginationStore.setPage(1);

		// Debounce query changes
		debounceManager.setQueryDebounce(async () => {
			await sharedHelpers.syncQueryToUrl();
			void sharedHelpers.refresh();
		});
	}

	function addQuickFilter(): void {
		queryStore.addQuickFilter();
	}

	function updateQuickFilter(index: number, patch: Partial<NotificationViewFilter>): void {
		queryStore.updateQuickFilter(index, patch);
		paginationStore.setPage(1);
		void sharedHelpers.refresh();
	}

	function removeQuickFilter(index: number): void {
		queryStore.removeQuickFilter(index);
		paginationStore.setPage(1);
		void sharedHelpers.refresh();
	}

	async function clearFilters(): Promise<void> {
		queryStore.clearFilters();
		paginationStore.setPage(1);
		await sharedHelpers.syncQueryToUrl();
		void sharedHelpers.refresh();
	}

	return {
		handleSearchInput,
		handleQuickQueryChange,
		addQuickFilter,
		updateQuickFilter,
		removeQuickFilter,
		clearFilters,
	};
}
