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

import type { Notification } from "$lib/api/types";

/**
 * Detail View Actions
 * Actions related to the detail/inline view panel
 */
export interface DetailActions {
	openDetail: (notificationId: string) => void;
	closeDetail: () => void;
	handleOpenInlineDetail: (notification: Notification) => Promise<void>;
	handleCloseInlineDetail: (opts?: { clearFocus?: boolean }) => Promise<void>;
	navigateToNextNotification: () => Promise<void>;
	navigateToPreviousNotification: () => Promise<void>;
	toggleSplitMode: () => Promise<void>;
	handlePaneResize: (deltaX: number) => void;
}
