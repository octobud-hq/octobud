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

export type ViewMatch = {
	view: NotificationView;
	source: "built-in" | "saved" | "tag" | "settings";
};

export type ViewMatchEntry = ViewMatch & {
	index: number;
};

/**
 * Generates a unique key for a view
 */
export function viewKey(view: NotificationView): string {
	return view.slug || view.id;
}

/**
 * Checks if a view matches the given query string
 */
export function matchesView(view: NotificationView, query: string): boolean {
	if (!query) {
		return true;
	}
	const haystack = `${view.name} ${view.slug ?? ""} ${view.description ?? ""}`.toLowerCase();
	return haystack.includes(query);
}

/**
 * Filters and returns matching views with their metadata
 */
export function getMatchingViews(
	builtInViews: NotificationView[],
	savedViews: NotificationView[],
	query: string,
	maxResults: number = 12
): ViewMatch[] {
	const matches: ViewMatch[] = [];
	const seen = new Set<string>();
	const normalizedQuery = query.trim().toLowerCase();

	const add = (view: NotificationView, source: ViewMatch["source"]) => {
		const key = viewKey(view);
		if (seen.has(key)) {
			return;
		}
		if (matchesView(view, normalizedQuery)) {
			matches.push({ view, source });
			seen.add(key);
		}
	};

	// Add built-in views first
	for (const view of builtInViews ?? []) {
		add(view, "built-in");
	}

	// Then add saved views
	for (const view of savedViews ?? []) {
		add(view, "saved");
	}

	return matches.slice(0, maxResults);
}

/**
 * Adds index property to view matches for selection management
 */
export function addIndicesToMatches(matches: ViewMatch[]): ViewMatchEntry[] {
	return matches.map((match, index) => ({
		...match,
		index,
	}));
}

/**
 * Separates view matches by type
 */
export function separateViewMatches(entries: ViewMatchEntry[]): {
	builtIn: ViewMatchEntry[];
	saved: ViewMatchEntry[];
	tag: ViewMatchEntry[];
	settings: ViewMatchEntry[];
} {
	return {
		builtIn: entries.filter((match) => match.source === "built-in"),
		saved: entries.filter((match) => match.source === "saved"),
		tag: entries.filter((match) => match.source === "tag"),
		settings: entries.filter((match) => match.source === "settings"),
	};
}
