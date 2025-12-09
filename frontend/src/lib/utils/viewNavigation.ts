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
import type { Tag } from "$lib/api/tags";

export interface ViewNavigationEntry {
	slug: string;
	isGlobalSearch?: boolean;
}

/**
 * Gets all views in order: built-in views, custom views, tag views
 * Custom views are sorted by displayOrder if all views have it, otherwise alphabetically
 * Tag views are sorted by their display order (server returns them sorted)
 */
export function getAllViewsInOrder(
	builtInViews: NotificationView[],
	allViews: NotificationView[],
	tags: Tag[] = []
): ViewNavigationEntry[] {
	const viewList: ViewNavigationEntry[] = [];

	// Add built-in views
	for (const builtIn of builtInViews) {
		viewList.push({ slug: builtIn.slug });
	}

	// Filter out system views to get only user/custom views
	const customViews = allViews.filter((v) => !v.systemView);

	// Sort custom views using the same logic as the sidebar:
	// - If all views have displayOrder (not null), sort by displayOrder
	// - Otherwise, sort alphabetically by name
	// Note: displayOrder can be 0 or any number, null/undefined means no order set
	const allHaveOrder = customViews.every((v) => v.displayOrder != null);

	let sortedCustomViews: NotificationView[];
	if (allHaveOrder) {
		// All views have displayOrder - sort by it (0 will come first)
		sortedCustomViews = [...customViews].sort(
			(a, b) => (a.displayOrder || 0) - (b.displayOrder || 0)
		);
	} else {
		// Not all views have displayOrder - sort alphabetically
		sortedCustomViews = [...customViews].sort((a, b) => a.name.localeCompare(b.name));
	}

	// Add custom views
	for (const view of sortedCustomViews) {
		viewList.push({ slug: view.slug });
	}

	// Add tag views after custom views
	// Tags are already sorted by display order from the server (sorted by display_order, then name)
	for (const tag of tags) {
		viewList.push({ slug: `tag-${tag.slug}` });
	}

	return viewList;
}

/**
 * Finds the current view's index in the ordered list
 */
export function getCurrentViewIndex(allViews: ViewNavigationEntry[], currentSlug: string): number {
	// Find the current view by slug
	return allViews.findIndex((v) => v.slug === currentSlug);
}

/**
 * Finds the next view to navigate to
 * Wraps around to the first view if at the end
 */
export function findNextView(
	allViews: ViewNavigationEntry[],
	currentIndex: number
): ViewNavigationEntry | null {
	if (allViews.length === 0) {
		return null;
	}

	// Navigate to next view if available
	if (currentIndex >= 0 && currentIndex < allViews.length - 1) {
		return allViews[currentIndex + 1];
	}

	// At the last view - wrap around to the first view
	if (currentIndex === allViews.length - 1) {
		return allViews[0] || null;
	}

	return null;
}

/**
 * Finds the previous view to navigate to
 * Returns null if at the beginning or no valid previous view
 */
export function findPreviousView(
	allViews: ViewNavigationEntry[],
	currentIndex: number
): ViewNavigationEntry | null {
	if (allViews.length === 0) {
		return null;
	}

	// Navigate to previous view if available
	if (currentIndex > 0) {
		return allViews[currentIndex - 1];
	}

	// Already at the first view or invalid index
	return null;
}
