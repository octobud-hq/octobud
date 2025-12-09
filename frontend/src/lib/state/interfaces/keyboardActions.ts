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
 * Keyboard Actions
 * Actions related to keyboard focus and navigation
 */
export interface KeyboardActions {
	focusNotificationAt: (index: number, options?: { scroll?: boolean }) => boolean;
	setFocusIndex: (index: number) => boolean;
	getFocusedNotification: () => import("$lib/api/types").Notification | null;
	focusNextNotification: () => Promise<boolean>;
	focusPreviousNotification: () => Promise<boolean>;
	focusFirstNotification: () => Promise<boolean>;
	focusLastNotification: () => Promise<boolean>;
	resetKeyboardFocus: () => boolean;
	clearFocusIfNeeded: () => void;
	executePendingFocusAction: () => void;
	consumePendingFocus: () => void;
}
