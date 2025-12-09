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
	handleSyncNewNotifications: () => Promise<void>;
}
