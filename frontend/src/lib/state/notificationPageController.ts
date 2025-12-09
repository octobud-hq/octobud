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

import type { NotificationPageController } from "./types";
import type { PageData as ViewPageData } from "../../routes/views/[slug]/$types";
import { createNotificationStore } from "../stores/notificationStore";
import { createPaginationStore } from "../stores/paginationStore";
import { createDetailViewStore } from "../stores/detailViewStore";
import { createSelectionStore } from "../stores/selectionStore";
import { createKeyboardNavigationStore } from "../stores/keyboardNavigationStore";
import { createUIStateStore } from "../stores/uiStateStore";
import { createQueryStore } from "../stores/queryStore";
import { createViewStore } from "../stores/viewStore";
import { createEventBus } from "../stores/eventBus";
import type { ControllerOptions, NotificationPageControllerActions } from "./interfaces";
import { BUILT_IN_VIEWS, normalizeViewId } from "./types";

// Import action controllers and helpers
import { createDebounceManager } from "./actions/debounceManager";
import { createSharedHelpers } from "./actions/sharedHelpers";
import { createOptimisticUpdateHelpers } from "./actions/optimisticUpdates";
import { createViewActionController } from "./actions/viewActionController";
import { createQueryActionController } from "./actions/queryActionController";
import { createPaginationActionController } from "./actions/paginationActionController";
import { createSelectionActionController } from "./actions/selectionActionController";
import { createKeyboardActionController } from "./actions/keyboardActionController";
import { createDetailActionController } from "./actions/detailActionController";
import { createNotificationActionController } from "./actions/notificationActionController";
import { createBulkActionController } from "./actions/bulkActionController";
import { createKeyboardShortcutActionController } from "./actions/keyboardShortcutActionController";

/**
 * V2 Notification Page Controller
 * Composed architecture with separated stores and action controllers
 */
export function createNotificationPageController(
	initialData: ViewPageData,
	options: ControllerOptions = {}
): NotificationPageController {
	// Initialize query stores
	const initialQuery = (initialData as any).initialQuery ?? "";
	const initialViewQuery = (initialData as any).viewQuery ?? "";

	// Create all stores in dependency order
	const notificationStore = createNotificationStore(initialData.initialPage);
	const paginationStore = createPaginationStore(
		initialData.initialPageNumber ?? initialData.initialPage.page,
		initialData.initialPage
	);
	const detailStore = createDetailViewStore(notificationStore);
	const selectionStore = createSelectionStore(notificationStore);
	const keyboardStore = createKeyboardNavigationStore(notificationStore);
	const uiStore = createUIStateStore();
	const queryStore = createQueryStore(initialQuery, initialViewQuery);
	const viewStore = createViewStore(
		initialData.views,
		normalizeViewId(initialData.selectedViewId ?? BUILT_IN_VIEWS.inbox.id)
	);
	const eventBus = createEventBus();

	// Create shared infrastructure
	const debounceManager = createDebounceManager();
	const sharedHelpers = createSharedHelpers(
		notificationStore,
		paginationStore,
		queryStore,
		options,
		debounceManager
	);

	// Store collection for controllers that need all stores
	const storeCollection = {
		notificationStore,
		paginationStore,
		detailStore,
		selectionStore,
		keyboardStore,
		uiStore,
		queryStore,
		viewStore,
	};

	// Create action controllers in dependency order
	// 1. Controllers with no cross-dependencies
	const viewActions = createViewActionController(
		storeCollection,
		options,
		sharedHelpers,
		debounceManager
	);
	const queryActions = createQueryActionController(
		queryStore,
		paginationStore,
		sharedHelpers,
		debounceManager
	);
	const paginationActions = createPaginationActionController(
		paginationStore,
		queryStore,
		notificationStore,
		uiStore,
		keyboardStore,
		detailStore,
		options,
		sharedHelpers
	);
	const toggleSelection = createSelectionActionController(selectionStore);

	// 2. Controllers that depend on other controllers
	// Create optimistic update helpers first (no dependencies on other controllers)
	const optimisticUpdateHelpers = createOptimisticUpdateHelpers(
		{
			notificationStore,
			paginationStore,
			detailStore,
			selectionStore,
			keyboardStore,
			uiStore,
		},
		options,
		sharedHelpers
	);

	// Create notification controller with optimistic updates (no dependencies on detailActions)
	const notificationActions = createNotificationActionController(
		{
			notificationStore,
			paginationStore,
			detailStore,
			selectionStore,
			keyboardStore,
		},
		options,
		optimisticUpdateHelpers,
		sharedHelpers
	);

	// Create detail controller with markRead from notificationActions (clean dependency order)
	const detailActions = createDetailActionController(
		detailStore,
		keyboardStore,
		notificationStore,
		paginationStore,
		uiStore,
		options,
		sharedHelpers,
		notificationActions.markRead
	);

	// Create keyboard controller with pagination and detail actions
	const keyboardActions = createKeyboardActionController(
		keyboardStore,
		notificationStore,
		paginationStore,
		uiStore,
		detailStore,
		options,
		{
			handleGoToNextPage: paginationActions.handleGoToNextPage,
			handleGoToPreviousPage: paginationActions.handleGoToPreviousPage,
		},
		{
			handleOpenInlineDetail: detailActions.handleOpenInlineDetail,
		}
	);

	// Create bulk controller
	const bulkActions = createBulkActionController(
		{
			notificationStore,
			paginationStore,
			selectionStore,
			queryStore,
			uiStore,
		},
		options,
		sharedHelpers
	);

	// Create keyboard shortcut controller with notification actions
	const keyboardShortcutActions = createKeyboardShortcutActionController(
		keyboardStore,
		uiStore,
		notificationActions
	);

	// Compose all actions into single object
	const actions: NotificationPageControllerActions = {
		// View actions
		...viewActions,
		// Query & filter actions
		...queryActions,
		// Pagination actions
		...paginationActions,
		// Selection actions (toggleSelection has logic, others exposed directly from store)
		toggleSelection,
		clearSelection: selectionStore.clearSelection,
		selectAllPage: selectionStore.selectAllPage,
		selectAllAcrossPages: selectionStore.selectAllAcrossPages,
		clearSelectAllMode: selectionStore.clearSelectAllMode,
		toggleMultiselectMode: selectionStore.toggleMultiselectMode,
		exitMultiselectMode: selectionStore.exitMultiselectMode,
		setSelectedIds: (ids: Set<string>) => selectionStore.selectedIds.set(ids),
		// Focus actions
		...keyboardActions,
		// Detail view actions
		...detailActions,
		// Individual notification actions
		...notificationActions,
		// Keyboard action handlers
		...keyboardShortcutActions,
		// Bulk operation actions
		...bulkActions,
		// UI state actions (exposed directly from store)
		toggleSidebar: uiStore.toggleSidebar,
		openSnoozeDropdown: uiStore.openSnoozeDropdown,
		closeSnoozeDropdown: uiStore.closeSnoozeDropdown,
		openCustomSnoozeDialog: uiStore.openCustomSnoozeDialog,
		closeCustomSnoozeDialog: uiStore.closeCustomSnoozeDialog,
		openTagDropdown: uiStore.openTagDropdown,
		closeTagDropdown: uiStore.closeTagDropdown,
		// Utility/helper actions
		destroy: () => {
			debounceManager.clearAll();
		},
	};

	// Return controller with same interface as v1
	return {
		stores: {
			// View state
			views: viewStore.views,
			selectedViewId: viewStore.selectedViewId,
			pageData: notificationStore.pageData,
			page: paginationStore.page,
			isLoading: paginationStore.isLoading,

			// Query & filters
			searchTerm: queryStore.searchTerm,
			quickQuery: queryStore.quickQuery,
			viewQuery: queryStore.viewQuery,
			quickFilters: queryStore.quickFilters,

			// Selection & multiselect
			selectedIds: selectionStore.selectedIds,
			multiselectMode: selectionStore.multiselectMode,
			selectAllMode: selectionStore.selectAllMode,

			// Keyboard focus
			keyboardFocusIndex: keyboardStore.keyboardFocusIndex,
			notificationRowComponents: keyboardStore.notificationRowComponents,
			pendingFocusIndex: keyboardStore.pendingFocusIndex,

			// Detail view
			detailOpen: detailStore.detailOpen,
			detailNotificationId: detailStore.detailNotificationId,
			currentDetailNotification: detailStore.currentDetailNotification,
			currentDetail: detailStore.currentDetail,
			detailLoading: detailStore.detailLoading,
			detailShowingStaleData: detailStore.detailShowingStaleData,
			detailIsRefreshing: detailStore.detailIsRefreshing,
			splitModeEnabled: uiStore.splitModeEnabled,
			listPaneWidth: uiStore.listPaneWidth,
			savedListScrollPosition: uiStore.savedListScrollPosition,

			// UI state
			hydrated: uiStore.hydrated,
			sidebarCollapsed: uiStore.sidebarCollapsed,
			snoozeDropdownState: uiStore.snoozeDropdownState,
			tagDropdownState: uiStore.tagDropdownState,
			customDateDialogOpen: uiStore.customDateDialogOpen,
			customDateDialogNotificationId: uiStore.customDateDialogNotificationId,
			bulkUpdating: uiStore.bulkUpdating,
		},
		derived: {
			// View-related derived
			builtInViewList: viewStore.builtInViewList,
			defaultViewId: viewStore.defaultViewId,
			defaultViewSlug: viewStore.defaultViewSlug,
			defaultViewDisplayName: viewStore.defaultViewDisplayName,
			selectedView: viewStore.selectedView,
			selectedViewSlug: viewStore.selectedViewSlug,
			isBuiltInArchiveView: viewStore.isBuiltInArchiveView,
			isBuiltInSnoozedView: viewStore.isBuiltInSnoozedView,
			isBuiltInEverythingView: viewStore.isBuiltInEverythingView,

			// Query-related derived
			isQueryModified: queryStore.isQueryModified,
			hasActiveFilters: queryStore.hasActiveFilters,

			// Pagination-related derived
			totalPages: paginationStore.totalPages,
			pageRangeStart: paginationStore.pageRangeStart,
			pageRangeEnd: paginationStore.pageRangeEnd,

			// Selection-related derived
			selectionMap: selectionStore.selectionMap,
			selectedCount: selectionStore.selectedCount,
			individualSelectionDisabled: selectionStore.individualSelectionDisabled,
			selectionEnabled: selectionStore.selectionEnabled,
		},
		actions,
	};
}
