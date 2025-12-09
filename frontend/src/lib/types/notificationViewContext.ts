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

import type { Notification, NotificationDetail } from "$lib/api/types";
import type { TimelineController } from "$lib/state/timelineController";

/**
 * Context key for NotificationView state
 */
export const NOTIFICATION_VIEW_CONTEXT_KEY = "notificationViewContext";

/**
 * List view context - shared state for list-related props
 */
export interface ListViewContext {
	hasActiveFilters: boolean;
	totalCount: number;
	pageRangeStart: number;
	pageRangeEnd: number;
	items: Notification[];
	isLoading: boolean;
	selectionEnabled: boolean;
	individualSelectionDisabled: boolean;
	detailNotificationId: string | null;
	selectionMap: Map<string, boolean>;
}

/**
 * Detail view context - shared state for detail-related props
 */
export interface DetailViewContext {
	detailOpen: boolean;
	currentDetailNotification: Notification | null;
	currentDetail: NotificationDetail | null;
	detailLoading: boolean;
	detailShowingStaleData: boolean;
	detailIsRefreshing: boolean;
	timelineController: TimelineController | undefined;
}

/**
 * Split view context - shared state for split view specific props
 */
export interface SplitViewContext {
	listPaneWidth: number;
	onPaneResize: (event: CustomEvent<{ deltaX: number }>) => void;
}

/**
 * Complete NotificationView context
 */
export interface NotificationViewContext {
	listView: ListViewContext;
	detailView: DetailViewContext;
	splitView: SplitViewContext;
	initialScrollPosition: number;
	apiError: string | null;
}
