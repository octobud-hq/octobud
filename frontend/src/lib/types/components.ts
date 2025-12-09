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
 * Component type interfaces for component refs.
 * These interfaces define the public API of components that are accessed via `bind:this`.
 */

/**
 * NotificationToolbar component interface
 * Used in NotificationViewHeader
 */
export interface NotificationToolbarComponent {
	focusSearchInput(): boolean;
}

/**
 * BulkSelectionBar component interface
 * Used in NotificationViewHeader
 */
export interface BulkSelectionBarComponent {
	openTagDropdown(): void;
	closeTagDropdown(): void;
	isTagDropdownOpenState(): boolean;
	openSnoozeDropdown(): void;
	closeSnoozeDropdown(): void;
	isSnoozeDropdownOpenState(): boolean;
}

/**
 * NotificationListSection component interface
 * Used in NotificationSingleMode and NotificationSplitMode
 */
export interface NotificationListSectionComponent {
	getScrollPosition(): number;
	setScrollPosition(scrollTop: number): void;
	saveScrollPosition(): number;
	restoreScrollPosition(): void;
	resetScrollPosition(): void;
}

/**
 * UnifiedSearchInput component interface
 * Used in NotificationToolbar
 */
export interface UnifiedSearchInputComponent {
	focus(): void;
}

/**
 * SnoozeDropdown component interface
 * Used in NotificationRow, DetailActionBar, and BulkSelectionBar
 */
export interface SnoozeDropdownComponent {
	setButtonElement(element: HTMLButtonElement | null): void;
}

/**
 * TagDropdown component interface
 * Used in NotificationRow, DetailActionBar, and BulkSelectionBar
 */
export interface TagDropdownComponent {
	setButtonElement(element: HTMLButtonElement | null): void;
}

/**
 * BulkTagDropdown component interface
 * Used in BulkSelectionBar
 */
export interface BulkTagDropdownComponent {
	setButtonElement(element: HTMLButtonElement | null): void;
}

/**
 * NotificationViewHeader component interface
 * Used in NotificationView
 */
export interface NotificationViewHeaderComponent {
	focusSearchInput(): boolean;
	openBulkTagDropdown(): boolean;
	closeBulkTagDropdown(): boolean;
	getBulkTagDropdownOpen(): boolean;
	openBulkSnoozeDropdown(): boolean;
	closeBulkSnoozeDropdown(): boolean;
	getBulkSnoozeDropdownOpen(): boolean;
}

/**
 * NotificationSingleMode component interface
 * Used in NotificationView
 */
export interface NotificationSingleModeComponent {
	getListSectionComponent(): NotificationListSectionComponent | null;
}

/**
 * NotificationSplitMode component interface
 * Used in NotificationView
 */
export interface NotificationSplitModeComponent {
	getListSectionComponent(): NotificationListSectionComponent | null;
}

/**
 * CommandPalette component interface
 * Used in PageHeader
 */
export interface CommandPaletteComponent {
	// Add methods here as they are discovered/needed
	[key: string]: unknown;
}
