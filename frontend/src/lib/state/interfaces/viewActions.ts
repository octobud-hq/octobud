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

import type { NotificationView } from "$lib/api/types";
import type { PageData as ViewPageData } from "../../../routes/views/[slug]/$types";

/**
 * View Actions
 * Actions related to view management and navigation
 */
export interface ViewActions {
	refresh: () => Promise<void>;
	refreshViewCounts: () => Promise<void>;
	setViews: (views: NotificationView[]) => void;
	syncFromData: (data: ViewPageData) => void;
	selectViewBySlug: (slug: string, shouldInvalidate?: boolean) => Promise<void>;
	navigateToNextView: () => boolean;
	navigateToPreviousView: () => boolean;
	handleSyncNewNotifications: (updatedGithubIds?: string[]) => Promise<void>;
	/** Register a callback invoked when the currently-open detail notification is refreshed by a sync. */
	registerTimelineRefreshHandler: (handler: (githubId: string) => void) => void;
	/** Register a callback invoked to scroll the visible list to a given scrollTop. */
	registerListScrollHandler: (handler: (scrollTop: number) => void) => void;
	/** Scroll the list back to the top — used on page change to start each new page at the top. */
	scrollListToTop: () => void;
	/**
	 * Scroll the list just enough to show a specific notification — used for the
	 * desktop-notification-click flow where the target may be at any index on
	 * the current page (or not on it at all). No-op if the notification isn't in
	 * the current pageData, or if not in SplitMode (the list isn't visible).
	 */
	ensureNotificationVisible: (notificationId: string) => void;
}
