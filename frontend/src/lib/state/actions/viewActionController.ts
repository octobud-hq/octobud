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
import { page } from "$app/stores";
import type { NotificationView } from "$lib/api/types";
import type { PageData as ViewPageData } from "../../../routes/views/[slug]/$types";
import type { ViewActions } from "../interfaces/viewActions";
import { fetchViews } from "$lib/api/views";
import {
	getAllViewsInOrder,
	getCurrentViewIndex,
	findNextView,
	findPreviousView,
} from "$lib/utils/viewNavigation";
import { BUILT_IN_VIEWS, normalizeViewId, normalizeViewSlug } from "../types";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { SelectionStore } from "../../stores/selectionStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { QueryStore } from "../../stores/queryStore";
import type { ViewStore } from "../../stores/viewStore";
import type { ControllerOptions } from "../interfaces/common";
import type { SharedHelpers } from "./sharedHelpers";
import type { DebounceManager } from "./debounceManager";

interface StoreCollection {
	notificationStore: NotificationStore;
	paginationStore: PaginationStore;
	detailStore: DetailStore;
	selectionStore: SelectionStore;
	keyboardStore: KeyboardStore;
	uiStore: UIStore;
	queryStore: QueryStore;
	viewStore: ViewStore;
}

/**
 * View Action Controller
 * Manages view navigation and synchronization
 */
export function createViewActionController(
	stores: StoreCollection,
	options: ControllerOptions,
	sharedHelpers: SharedHelpers,
	debounceManager: DebounceManager
): ViewActions {
	const {
		notificationStore,
		paginationStore,
		detailStore,
		selectionStore,
		keyboardStore,
		queryStore,
		viewStore,
		uiStore,
	} = stores;

	async function refresh(): Promise<void> {
		return sharedHelpers.refresh();
	}

	async function refreshViewCounts(): Promise<void> {
		if (typeof window === "undefined") {
			return;
		}

		try {
			const updatedViews = await fetchViews();
			viewStore.setViews(updatedViews);
		} catch (error) {
			console.error("Failed to refresh view counts:", error);
		}
	}

	function setViews(views: NotificationView[]): void {
		viewStore.setViews(views);
	}

	function syncFromData(data: ViewPageData): void {
		viewStore.selectView(normalizeViewId(data.selectedViewId ?? "inbox"));
		notificationStore.setPageData(data.initialPage);
		paginationStore.setPage(data.initialPageNumber ?? data.initialPage.page);
		paginationStore.setTotal(data.initialPage.total);
		paginationStore.setPageSize(data.initialPage.pageSize);
		queryStore.setSearchTerm(data.initialSearchTerm ?? "");

		const nextQuery = (data as any).initialQuery ?? "";
		const nextViewQuery = (data as any).viewQuery ?? "";
		queryStore.setQuickQuery(nextQuery || nextViewQuery);
		queryStore.setViewQuery(nextViewQuery);

		// Clear debounce timeouts
		debounceManager.clearAll();

		paginationStore.setLoading(false);
	}

	async function selectViewBySlug(slug: string, shouldInvalidate?: boolean): Promise<void> {
		const targetSlug = normalizeViewSlug(slug) || BUILT_IN_VIEWS.inbox.slug;

		// Exit multiselect mode and clear selections when switching views
		selectionStore.clearSelection();
		selectionStore.exitMultiselectMode();

		// Immediately update selectedViewId in the store to keep state consistent
		// This is especially important for newly created views
		const currentViews = get(viewStore.views);
		const builtInViews = get(viewStore.builtInViewList);

		// Check if it's a built-in view
		const builtInView = builtInViews.find((v) => v.slug === targetSlug);
		if (builtInView) {
			viewStore.selectView(normalizeViewId(builtInView.id));
		} else {
			// Check if it's a custom view
			const customView = currentViews.find((v) => v.slug === targetSlug);
			if (customView) {
				viewStore.selectView(normalizeViewId(customView.id));
			} else {
				// Fallback to slug if view not found (shouldn't happen, but be safe)
				viewStore.selectView(normalizeViewId(targetSlug));
			}
		}

		// View navigation creates a new history entry
		if (options.navigateToUrl && typeof window !== "undefined") {
			const url = new URL(window.location.href);
			url.pathname = `/views/${encodeURIComponent(targetSlug)}`;
			url.search = "";
			await options.navigateToUrl(url.pathname + url.search, { replace: false });
		}
	}

	function getCurrentSlug(): string {
		// Get current slug from URL pathname to handle tag views
		// Tag views use format: /views/tag-{slug}
		const currentPath = get(page).url.pathname;
		if (currentPath.startsWith("/views/")) {
			const slug = currentPath.replace("/views/", "");
			return slug;
		}
		// Fallback to viewStore selectedViewSlug
		return get(viewStore.selectedViewSlug);
	}

	function navigateToNextView(): boolean {
		const tags = options.getTags?.() ?? [];
		const allViews = getAllViewsInOrder(get(viewStore.builtInViewList), get(viewStore.views), tags);
		const currentSlug = getCurrentSlug();
		const currentIndex = getCurrentViewIndex(allViews, currentSlug);
		const nextView = findNextView(allViews, currentIndex);
		if (nextView) {
			void selectViewBySlug(nextView.slug);
			return true;
		}
		return false;
	}

	function navigateToPreviousView(): boolean {
		const tags = options.getTags?.() ?? [];
		const allViews = getAllViewsInOrder(get(viewStore.builtInViewList), get(viewStore.views), tags);
		const currentSlug = getCurrentSlug();
		const currentIndex = getCurrentViewIndex(allViews, currentSlug);
		const previousView = findPreviousView(allViews, currentIndex);
		if (previousView) {
			void selectViewBySlug(previousView.slug);
			return true;
		}
		return false;
	}

	async function handleSyncNewNotifications(): Promise<void> {
		const currentPage = get(paginationStore.page);
		const isSplitMode = get(uiStore.splitModeEnabled);
		const isDetailOpen = get(detailStore.detailOpen);
		const currentFocusIndex = get(keyboardStore.keyboardFocusIndex);
		const currentDetailId = get(detailStore.detailNotificationId);
		const currentPageData = get(notificationStore.pageData);

		// Always refresh views/tags for up-to-date unread counts
		await refreshViewCounts();

		// Determine if we should refresh notifications
		// Always refresh if page data is empty (fresh load)
		const hasEmptyPageData = currentPageData.items.length === 0;
		const shouldRefreshNotifications =
			hasEmptyPageData || (currentPage === 1 && ((!isSplitMode && !isDetailOpen) || isSplitMode));

		if (!shouldRefreshNotifications) {
			return;
		}

		// Capture current state before refresh
		const focusedNotification =
			currentFocusIndex !== null &&
			currentFocusIndex >= 0 &&
			currentFocusIndex < currentPageData.items.length
				? currentPageData.items[currentFocusIndex]
				: null;
		const focusedNotificationId = focusedNotification
			? (focusedNotification.githubId ?? focusedNotification.id)
			: null;

		const detailNotification =
			currentDetailId && currentPageData.items.length > 0
				? currentPageData.items.find((n) => (n.githubId ?? n.id) === currentDetailId)
				: null;
		const capturedDetailNotificationId = detailNotification
			? (detailNotification.githubId ?? detailNotification.id)
			: null;

		// Capture current notification IDs to detect new ones
		const currentNotificationIds = new Set(currentPageData.items.map((n) => n.githubId ?? n.id));

		// Refresh notifications
		await sharedHelpers.refresh();
		await tick();

		const refreshedItems = get(notificationStore.pageData).items;
		const refreshedFocusIndex =
			focusedNotificationId !== null
				? refreshedItems.findIndex((n) => (n.githubId ?? n.id) === focusedNotificationId)
				: -1;

		// Restore keyboard focus if the notification is still on the page
		if (refreshedFocusIndex !== -1) {
			keyboardStore.setFocusIndex(refreshedFocusIndex);
		} else if (focusedNotificationId !== null) {
			keyboardStore.resetFocus();
		}

		// In split mode, preserve the detail notification even if its list card is pushed off
		if (isSplitMode && capturedDetailNotificationId !== null) {
			const detailStillOnPage = refreshedItems.some(
				(n) => (n.githubId ?? n.id) === capturedDetailNotificationId
			);
			// If detail is still on page, it will remain open automatically
			// If not, detail view will show stale data (acceptable)
		}
	}

	return {
		refresh,
		refreshViewCounts,
		setViews,
		syncFromData,
		selectViewBySlug,
		navigateToNextView,
		navigateToPreviousView,
		handleSyncNewNotifications,
	};
}
