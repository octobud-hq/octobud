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
 * Keyboard Shortcut Actions
 * Actions that are triggered by keyboard shortcuts for focused notifications
 */
export interface KeyboardShortcutActions {
	archiveFocusedNotification: () => boolean;
	markFocusedNotificationRead: () => boolean;
	muteFocusedNotification: () => boolean;
	starFocusedNotification: () => boolean;
	snoozeFocusedNotification: () => boolean;
	tagFocusedNotification: () => boolean;
	unfilterFocusedNotification: () => boolean;
}
