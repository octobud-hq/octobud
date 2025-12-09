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

import { writable, derived } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import type { NotificationPage } from "$lib/api/types";

/**
 * Pagination Store
 * Manages pagination state and derived values
 */
export function createPaginationStore(initialPage: number, initialPageData: NotificationPage) {
	const page = writable<number>(initialPage);
	const pageSize = writable<number>(initialPageData.pageSize);
	const total = writable<number>(initialPageData.total);
	const isLoading = writable<boolean>(false);

	// Derived stores
	const totalPages = derived([total, pageSize], ([$total, $pageSize]) =>
		Math.max(1, Math.ceil($total / $pageSize))
	);

	const pageRangeStart = derived([page, pageSize, total], ([$page, $pageSize, $total]) => {
		if ($total === 0) return 0;
		return ($page - 1) * $pageSize + 1;
	});

	const pageRangeEnd = derived([page, pageSize, total], ([$page, $pageSize, $total]) => {
		if ($total === 0) return 0;
		return Math.min($page * $pageSize, $total);
	});

	return {
		// Expose stores
		page: page as Writable<number>,
		pageSize: pageSize as Writable<number>,
		total: total as Writable<number>,
		isLoading: isLoading as Writable<boolean>,

		// Derived stores
		totalPages: totalPages as Readable<number>,
		pageRangeStart: pageRangeStart as Readable<number>,
		pageRangeEnd: pageRangeEnd as Readable<number>,

		// Actions
		setPage: (n: number) => {
			page.set(n);
		},
		setTotal: (n: number) => {
			total.set(n);
		},
		setPageSize: (size: number) => {
			pageSize.set(size);
		},
		setLoading: (loading: boolean) => {
			isLoading.set(loading);
		},
	};
}

export type PaginationStore = ReturnType<typeof createPaginationStore>;
