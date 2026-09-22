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
import { VIEWS_DEPENDENCY } from "$lib/constants/loaderDependencies";

// Disable SSR for static SPA build - all rendering happens client-side
export const ssr = false;
// Disable prerendering - pages are rendered at runtime
export const prerender = false;

export const load: LayoutLoad = async ({ fetch, url, depends, untrack }) => {
	// This loader re-runs only when VIEWS_DEPENDENCY is invalidated, never merely because
	// the URL changed. The views and tags responses compute unread counts for every view
	// and tag, which is slow on large databases, and nothing about a navigation needs a
	// fresh copy: the page loader only needs the view's query, and sidebar badges are
	// refreshed in the background after navigation. Reading the pathname untracked keeps
	// the setup-page short-circuit without making the URL a dependency.
	depends(VIEWS_DEPENDENCY);
	if (untrack(() => url.pathname === "/setup")) {
		return { views: [], tags: [], apiError: null };
	}

	// In browser, try to fetch data
	if (browser) {
		try {
			const [views, tags] = await Promise.all([fetchViews(fetch), fetchTags(fetch)]);
			return { views, tags, apiError: null };
		} catch (error) {
			// Check if this is a network error (API unreachable)
			// A failed load must not be cached for the session: read the pathname *tracked*
			// here so the next navigation re-runs this loader and retries the fetch.
			void url.pathname;
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
