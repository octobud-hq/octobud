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

import type { Notification, NotificationPage } from "$lib/api/types";

/**
 * Pagination Actions
 * Actions related to pagination and page data management
 */
export interface PaginationActions {
	goToPage: (nextPage: number) => void;
	handleGoToPage: (nextPage: number) => Promise<void>;
	handleGoToNextPage: () => Promise<boolean>;
	handleGoToPreviousPage: () => Promise<boolean>;
	setPageData: (pageData: NotificationPage) => void;
	updateNotification: (notification: Notification) => void;
}
