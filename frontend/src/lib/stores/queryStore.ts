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

import { writable, derived, get } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import type { NotificationViewFilter } from "$lib/api/types";

/**
 * Query Store
 * Manages search and filter state
 */
export function createQueryStore(initialQuery: string = "", initialViewQuery: string = "") {
	const searchTerm = writable<string>("");
	const quickQuery = writable<string>(initialQuery || initialViewQuery);
	const viewQuery = writable<string>(initialViewQuery);
	const quickFilters = writable<NotificationViewFilter[]>([]);

	// Derived stores
	const isQueryModified = derived([quickQuery, viewQuery], ([$quickQuery, $viewQuery]) => {
		const normalized = $quickQuery.trim();
		const baseNormalized = $viewQuery.trim();
		return normalized !== baseNormalized;
	});

	const hasActiveFilters = derived([quickQuery, viewQuery], ([$quickQuery, $viewQuery]) => {
		const normalized = $quickQuery.trim();
		const baseNormalized = $viewQuery.trim();
		return normalized !== baseNormalized && normalized !== "";
	});

	return {
		// Expose stores
		searchTerm: searchTerm as Writable<string>,
		quickQuery: quickQuery as Writable<string>,
		viewQuery: viewQuery as Writable<string>,
		quickFilters: quickFilters as Writable<NotificationViewFilter[]>,

		// Derived stores
		isQueryModified: isQueryModified as Readable<boolean>,
		hasActiveFilters: hasActiveFilters as Readable<boolean>,

		// Actions
		setSearchTerm: (term: string) => {
			searchTerm.set(term);
		},
		setQuickQuery: (query: string) => {
			quickQuery.set(query);
		},
		setViewQuery: (query: string) => {
			viewQuery.set(query);
		},
		addQuickFilter: () => {
			quickFilters.update((filters) => [...filters, { field: "", operator: "equals", value: "" }]);
		},
		updateQuickFilter: (index: number, patch: Partial<NotificationViewFilter>) => {
			quickFilters.update((filters) =>
				filters.map((filter, i) => (i === index ? { ...filter, ...patch } : filter))
			);
		},
		removeQuickFilter: (index: number) => {
			quickFilters.update((filters) => filters.filter((_, i) => i !== index));
		},
		clearFilters: () => {
			searchTerm.set("");
			quickQuery.set(get(viewQuery));
			quickFilters.set([]);
		},
	};
}

export type QueryStore = ReturnType<typeof createQueryStore>;
