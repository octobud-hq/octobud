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

/**
 * Shared types, constants, and utilities for v2 controller
 */

import type { Writable, Readable } from "svelte/store";
import type {
	Notification,
	NotificationView,
	NotificationViewFilter,
	NotificationPage,
	Tag,
} from "$lib/api/types";
import type { PageData as ViewPageData } from "../../routes/views/[slug]/$types";
import type { NotificationPageControllerActions } from "./interfaces";

// ========================================================================
// TYPES
// ========================================================================

export type SelectAllMode = "none" | "page" | "all";

export interface SnoozeDropdownState {
	notificationId: string;
	source: string;
}

export interface TagDropdownState {
	notificationId: string;
	source: string;
}

export interface NotificationRowComponent {
	focus: () => void;
	scrollIntoView: () => void;
}

// ========================================================================
// BUILT-IN VIEW CONSTANTS
// ========================================================================

export const BUILT_IN_VIEWS = {
	inbox: {
		id: "__inbox__",
		slug: "inbox",
		name: "Inbox",
		icon: "inbox",
		query: "",
	},
	starred: {
		id: "__starred__",
		slug: "starred",
		name: "Starred",
		icon: "star",
		query: "starred:true -muted:true",
	},
	archive: {
		id: "__archive__",
		slug: "archive",
		name: "Archive",
		icon: "archive",
		query: "archived:true -muted:true",
	},
	snoozed: {
		id: "__snoozed__",
		slug: "snoozed",
		name: "Snoozed",
		icon: "snooze",
		query: "snoozed:true -muted:true",
	},
	everything: {
		id: "__everything__",
		slug: "everything",
		name: "Everything",
		icon: "infinity",
		query: "-muted:true",
	},
} as const;

// ========================================================================
// UTILITY FUNCTIONS
// ========================================================================

/**
 * Normalize view ID - converts slug to ID or returns ID if already an ID
 */
export function normalizeViewId(viewIdOrSlug: string | null | undefined): string {
	if (!viewIdOrSlug) {
		return BUILT_IN_VIEWS.inbox.id;
	}

	// Map slugs to IDs using BUILT_IN_VIEWS
	for (const view of Object.values(BUILT_IN_VIEWS)) {
		if (view.slug === viewIdOrSlug || view.id === viewIdOrSlug) {
			return view.id;
		}
	}

	return viewIdOrSlug;
}

/**
 * Normalize view slug - converts ID to slug or returns slug if already a slug
 */
export function normalizeViewSlug(viewIdOrSlug: string | null | undefined): string {
	if (!viewIdOrSlug) {
		return BUILT_IN_VIEWS.inbox.slug;
	}

	// Map IDs to slugs using BUILT_IN_VIEWS
	for (const view of Object.values(BUILT_IN_VIEWS)) {
		if (view.id === viewIdOrSlug || view.slug === viewIdOrSlug) {
			return view.slug;
		}
	}

	return viewIdOrSlug;
}

// ========================================================================
// CONTROLLER INTERFACES
// ========================================================================

export interface NotificationPageControllerStores {
	// View state
	views: Writable<NotificationView[]>;
	selectedViewId: Writable<string>;
	pageData: Writable<NotificationPage>;
	page: Writable<number>;
	isLoading: Writable<boolean>;

	// Query & filters
	searchTerm: Writable<string>;
	quickQuery: Writable<string>;
	viewQuery: Writable<string>;
	quickFilters: Writable<NotificationViewFilter[]>;

	// Selection & multiselect
	selectedIds: Writable<Set<string>>;
	multiselectMode: Writable<boolean>;
	selectAllMode: Writable<SelectAllMode>;

	// Keyboard focus
	keyboardFocusIndex: Writable<number | null>;
	notificationRowComponents: Writable<Array<NotificationRowComponent>>;
	pendingFocusIndex: Readable<number | null>;

	// Detail view
	detailOpen: Readable<boolean>;
	detailNotificationId: Writable<string | null>;
	currentDetailNotification: Readable<Notification | null>;
	currentDetail: Writable<import("$lib/api/types").NotificationDetail | null>;
	detailLoading: Writable<boolean>;
	detailShowingStaleData: Writable<boolean>;
	detailIsRefreshing: Writable<boolean>;
	splitModeEnabled: Writable<boolean>;
	listPaneWidth: Writable<number>;
	savedListScrollPosition: Writable<number>;

	// UI state
	hydrated: Readable<boolean>;
	sidebarCollapsed: Writable<boolean>;
	snoozeDropdownState: Writable<SnoozeDropdownState | null>;
	tagDropdownState: Writable<TagDropdownState | null>;
	customDateDialogOpen: Writable<boolean>;
	customDateDialogNotificationId: Writable<string | null>;
	bulkUpdating: Writable<boolean>;
}

export interface NotificationPageControllerDerived {
	// View-related derived
	builtInViewList: Readable<NotificationView[]>;
	defaultViewId: Readable<string>;
	defaultViewSlug: Readable<string>;
	defaultViewDisplayName: Readable<string>;
	selectedView: Readable<NotificationView | null>;
	selectedViewSlug: Readable<string>;
	isBuiltInArchiveView: Readable<boolean>;
	isBuiltInSnoozedView: Readable<boolean>;
	isBuiltInEverythingView: Readable<boolean>;

	// Query-related derived
	isQueryModified: Readable<boolean>;
	hasActiveFilters: Readable<boolean>;

	// Pagination-related derived
	totalPages: Readable<number>;
	pageRangeStart: Readable<number>;
	pageRangeEnd: Readable<number>;

	// Selection-related derived
	selectionMap: Readable<Map<string, boolean>>;
	selectedCount: Readable<number>;
	individualSelectionDisabled: Readable<boolean>;
	selectionEnabled: Readable<boolean>;
}

// Re-export controller options and actions interface
export type { ControllerOptions, NavigationState, NavigateOptions } from "./interfaces";
// Re-export the composed interface (already imported above)
export type { NotificationPageControllerActions };

// Re-export store types
export type { NotificationStore } from "../stores/notificationStore";
export type { PaginationStore } from "../stores/paginationStore";
export type { DetailStore } from "../stores/detailViewStore";
export type { SelectionStore } from "../stores/selectionStore";
export type { KeyboardStore } from "../stores/keyboardNavigationStore";
export type { UIStore } from "../stores/uiStateStore";
export type { QueryStore } from "../stores/queryStore";
export type { ViewStore } from "../stores/viewStore";
export type { EventBus } from "../stores/eventBus";

// Re-export event types
export type { NotificationEvent } from "../stores/eventBus";

// NotificationPageControllerActions interface - defined in interfaces/index.ts

export interface NotificationPageController {
	stores: NotificationPageControllerStores;
	derived: NotificationPageControllerDerived;
	actions: NotificationPageControllerActions;
}
