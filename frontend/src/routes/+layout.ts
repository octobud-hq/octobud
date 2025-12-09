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

import type { LayoutLoad } from "./$types";
import { fetchViews } from "$lib/api/views";
import { fetchTags } from "$lib/api/tags";
import { browser } from "$app/environment";
import { ApiUnreachableError, isNetworkError } from "$lib/api/fetch";

// Disable SSR for static SPA build - all rendering happens client-side
export const ssr = false;
// Disable prerendering - pages are rendered at runtime
export const prerender = false;

export const load: LayoutLoad = async ({ fetch, url }) => {
	// Don't fetch data on setup page
	if (url.pathname === "/setup") {
		return { views: [], tags: [], apiError: null };
	}

	// In browser, try to fetch data
	if (browser) {
		try {
			const [views, tags] = await Promise.all([fetchViews(fetch), fetchTags(fetch)]);
			return { views, tags, apiError: null };
		} catch (error) {
			// Check if this is a network error (API unreachable)
			if (error instanceof ApiUnreachableError || isNetworkError(error)) {
				console.error("API unreachable:", error);
				return { views: [], tags: [], apiError: "Unable to reach the API server" };
			}
			// If fetch fails, return empty data
			console.error("Failed to load data:", error);
			return { views: [], tags: [], apiError: null };
		}
	}

	// Return empty data if not in browser
	return { views: [], tags: [], apiError: null };
};
