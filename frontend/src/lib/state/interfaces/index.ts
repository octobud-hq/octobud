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
 * Action Interfaces
 * Modular interfaces for notification page controller actions
 *
 * These interfaces are split by domain to allow components and controllers
 * to import only the interfaces they need.
 */

export type { ViewActions } from "./viewActions";
export type { QueryActions } from "./queryActions";
export type { PaginationActions } from "./paginationActions";
export type { SelectionActions } from "./selectionActions";
export type { KeyboardActions } from "./keyboardActions";
export type { DetailActions } from "./detailActions";
export type { NotificationActions } from "./notificationActions";
export type { KeyboardShortcutActions } from "./keyboardShortcutActions";
export type { BulkActions } from "./bulkActions";
export type { NavigationState, NavigateOptions, ControllerOptions } from "./common";

// Import all interfaces to ensure they're available for the composed interface
import type { ViewActions } from "./viewActions";
import type { QueryActions } from "./queryActions";
import type { PaginationActions } from "./paginationActions";
import type { SelectionActions } from "./selectionActions";
import type { KeyboardActions } from "./keyboardActions";
import type { DetailActions } from "./detailActions";
import type { NotificationActions } from "./notificationActions";
import type { KeyboardShortcutActions } from "./keyboardShortcutActions";
import type { BulkActions } from "./bulkActions";

/**
 * NotificationPageControllerActions
 * Composed interface containing all action types
 */
export interface NotificationPageControllerActions
	extends
		ViewActions,
		QueryActions,
		PaginationActions,
		SelectionActions,
		KeyboardActions,
		DetailActions,
		NotificationActions,
		KeyboardShortcutActions,
		BulkActions {
	// UI state actions (exposed directly from store)
	toggleSidebar: () => void;
	openSnoozeDropdown: (notificationId: string, source: string) => void;
	closeSnoozeDropdown: () => void;
	openCustomSnoozeDialog: (notificationId: string) => void;
	closeCustomSnoozeDialog: () => void;
	openTagDropdown: (notificationId: string, source: string) => void;
	closeTagDropdown: () => void;
	// Utility/helper actions
	destroy: () => void;
}
