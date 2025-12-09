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
import type { Notification, NotificationPage } from "$lib/api/types";
import type { PaginationActions } from "../interfaces/paginationActions";
import type { NavigationState } from "../interfaces/common";
import type { PaginationStore } from "../../stores/paginationStore";
import type { QueryStore } from "../../stores/queryStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { ControllerOptions } from "../interfaces/common";
import type { SharedHelpers } from "./sharedHelpers";

/**
 * Pagination Action Controller
 * Manages pagination state and navigation
 */
export function createPaginationActionController(
	paginationStore: PaginationStore,
	queryStore: QueryStore,
	notificationStore: NotificationStore,
	uiStore: UIStore,
	keyboardStore: KeyboardStore,
	detailStore: DetailStore,
	options: ControllerOptions,
	sharedHelpers: SharedHelpers
): PaginationActions {
	// Helper to get current UI state for history preservation
	function getCurrentNavState(): NavigationState {
		return {
			sidebarCollapsed: get(uiStore.sidebarCollapsed),
			splitModeEnabled: get(uiStore.splitModeEnabled),
			detailNotificationId: get(detailStore.detailNotificationId),
			savedScrollPosition: get(uiStore.savedListScrollPosition),
			keyboardFocusIndex: get(keyboardStore.keyboardFocusIndex),
		};
	}

	function goToPage(nextPage: number): void {
		paginationStore.setPage(nextPage);
	}

	async function handleGoToPage(nextPage: number): Promise<void> {
		const total = get(paginationStore.totalPages);
		const currentPage = get(paginationStore.page);

		if (nextPage === currentPage || nextPage < 1 || nextPage > total) {
			return;
		}

		paginationStore.setPage(nextPage);

		// Pagination from toolbar buttons creates a new history entry
		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);
			const currentQuery = get(queryStore.quickQuery);
			const viewQueryValue = get(queryStore.viewQuery);

			if (currentQuery && currentQuery !== viewQueryValue) {
				url.searchParams.set("query", currentQuery);
			} else {
				url.searchParams.delete("query");
			}

			if (nextPage > 1) {
				url.searchParams.set("page", String(nextPage));
			} else {
				url.searchParams.delete("page");
			}

			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false, // Push for pagination
			});
		}
	}

	async function handleGoToNextPage(): Promise<boolean> {
		const currentPage = get(paginationStore.page);
		const total = get(paginationStore.totalPages);

		if (currentPage >= total) {
			return false;
		}

		const nextPage = currentPage + 1;
		paginationStore.setPage(nextPage);

		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);
			const currentQuery = get(queryStore.quickQuery);
			const viewQueryValue = get(queryStore.viewQuery);

			if (currentQuery && currentQuery !== viewQueryValue) {
				url.searchParams.set("query", currentQuery);
			} else {
				url.searchParams.delete("query");
			}

			url.searchParams.set("page", String(nextPage));
			url.searchParams.set("target", "first");

			// Pagination always creates a new history entry
			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false,
			});
			return true;
		}

		return false;
	}

	async function handleGoToPreviousPage(): Promise<boolean> {
		const currentPage = get(paginationStore.page);

		if (currentPage <= 1) {
			return false;
		}

		const prevPage = currentPage - 1;
		paginationStore.setPage(prevPage);

		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);
			const currentQuery = get(queryStore.quickQuery);
			const viewQueryValue = get(queryStore.viewQuery);

			if (currentQuery && currentQuery !== viewQueryValue) {
				url.searchParams.set("query", currentQuery);
			} else {
				url.searchParams.delete("query");
			}

			if (prevPage > 1) {
				url.searchParams.set("page", String(prevPage));
			} else {
				url.searchParams.delete("page");
			}
			url.searchParams.set("target", "last");

			// Pagination always creates a new history entry
			await options.navigateToUrl(url.pathname + url.search, {
				state: getCurrentNavState(),
				replace: false,
			});
			return true;
		}

		return false;
	}

	function setPageData(newPageData: NotificationPage): void {
		notificationStore.setPageData(newPageData);
	}

	function updateNotification(notification: Notification): void {
		notificationStore.updateNotification(notification);
	}

	return {
		goToPage,
		handleGoToPage,
		handleGoToNextPage,
		handleGoToPreviousPage,
		setPageData,
		updateNotification,
	};
}
