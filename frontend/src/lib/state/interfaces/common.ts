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
 * Common interfaces and types used across action interfaces
 */

export interface NavigationState {
	sidebarCollapsed?: boolean;
	splitModeEnabled?: boolean;
	detailNotificationId?: string | null;
	savedScrollPosition?: number;
	keyboardFocusIndex?: number | null;
}

export interface NavigateOptions {
	state?: NavigationState;
	/** If true, replaces the current history entry instead of creating a new one */
	replace?: boolean;
}

export interface ControllerOptions {
	onRefresh?: () => Promise<void>;
	onAfterRefresh?: () => Promise<void>;
	onRefreshViewCounts?: () => Promise<void>;
	navigateToUrl?: (url: string, options?: NavigateOptions) => Promise<void>;
	requestBulkConfirmation?: (action: string, count: number) => Promise<boolean>;
	getTags?: () => import("$lib/api/tags").Tag[];
}
