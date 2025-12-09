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

import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import { fetchNotifications } from "$lib/api/notifications";
import { fetchTags } from "$lib/api/tags";
import { ApiUnreachableError, isNetworkError } from "$lib/api/fetch";
import type { NotificationViewFilter, ViewFilterOperator } from "$lib/api/types";

const BUILT_IN_VIEW_SLUG = "inbox";
const EVERYTHING_VIEW_SLUG = "everything";
const DONE_VIEW_SLUG = "done";
const ARCHIVE_VIEW_SLUG = "archive";
const SNOOZED_VIEW_SLUG = "snoozed";
const VALID_OPERATORS: ViewFilterOperator[] = [
	"equals",
	"does not equal",
	"contains",
	"does not contain",
];
const VALID_OPERATOR_SET = new Set<ViewFilterOperator>(VALID_OPERATORS);

const normalizeOperator = (value: unknown): ViewFilterOperator | null => {
	if (typeof value !== "string") return null;
	const normalized = value.trim().toLowerCase();
	return VALID_OPERATOR_SET.has(normalized as ViewFilterOperator)
		? (normalized as ViewFilterOperator)
		: null;
};

const parseFilters = (raw: string | null): NotificationViewFilter[] => {
	if (!raw) {
		return [];
	}

	try {
		const decoded = JSON.parse(raw) as unknown;
		if (!Array.isArray(decoded)) {
			return [];
		}

		return decoded
			.map((item) => {
				if (
					!item ||
					typeof item !== "object" ||
					typeof (item as Record<string, unknown>).field !== "string" ||
					typeof (item as Record<string, unknown>).value !== "string"
				) {
					return null;
				}

				const operator = normalizeOperator((item as Record<string, unknown>).operator);
				if (!operator) {
					return null;
				}

				return {
					field: (item as Record<string, string>).field.trim(),
					operator,
					value: (item as Record<string, string>).value.trim(),
				};
			})
			.filter(
				(item): item is NotificationViewFilter =>
					!!item && item.field.length > 0 && item.operator.length > 0
			);
	} catch {
		return [];
	}
};

const parsePageNumber = (raw: string | null): number => {
	if (!raw) return 1;
	const value = Number.parseInt(raw, 10);
	if (Number.isNaN(value) || value < 1) {
		return 1;
	}
	return value;
};

export const load: PageLoad = async ({ fetch, params, url, parent }) => {
	const { views, tags, apiError } = await parent();

	// If parent layout had an API error, propagate it
	if (apiError) {
		return {
			views: [],
			tags: [],
			selectedViewSlug: params.slug ?? BUILT_IN_VIEW_SLUG,
			selectedViewId: params.slug ?? BUILT_IN_VIEW_SLUG,
			// baseFilters removed - views now use query strings
			initialPage: { items: [], total: 0, page: 1, pageSize: 50 },
			initialSearchTerm: "",
			initialQuickFilters: [],
			initialPageNumber: 1,
			initialQuery: "",
			viewQuery: "",
			apiError,
		};
	}

	const slug = params.slug ?? BUILT_IN_VIEW_SLUG;

	// Check if this is a tag view (slug starts with "tag-")
	const isTagView = slug.startsWith("tag-");
	let tagSlug: string | null = null;
	let tag = null;

	if (isTagView) {
		// Extract tag slug from "tag-{slug}" format
		tagSlug = slug.substring(4); // Remove "tag-" prefix

		// Fetch tags to find the tag by slug
		try {
			const tags = await fetchTags(fetch);
			tag = tags.find((t) => t.slug === tagSlug);

			if (!tag) {
				// Tag not found, redirect to inbox
				throw redirect(302, "/views/inbox");
			}
		} catch (error) {
			console.error("Failed to fetch tags:", error);
			throw redirect(302, "/views/inbox");
		}
	}

	// Define all system view slugs
	const systemViewSlugs = [
		BUILT_IN_VIEW_SLUG,
		EVERYTHING_VIEW_SLUG,
		DONE_VIEW_SLUG,
		ARCHIVE_VIEW_SLUG,
		SNOOZED_VIEW_SLUG,
	];
	const isSystemView = systemViewSlugs.includes(slug);

	// Find the view (custom or system)
	const selectedView = isSystemView ? null : (views.find((view) => view.slug === slug) ?? null);

	// Also check for system views in the views array (backend now returns them with queries)
	const systemView = views.find((view) => view.slug === slug && view.systemView);

	if (!selectedView && !systemView && !isSystemView && !isTagView) {
		throw redirect(302, "/views/inbox");
	}

	// Views now use query strings instead of filters array
	// baseFilters is no longer needed - the view's query is used directly

	const quickFilters = parseFilters(url.searchParams.get("filters"));
	const page = parsePageNumber(url.searchParams.get("page"));

	// Build the query string from the view's query
	let viewQuery = "";

	// Handle tag views - explicit query with in:anywhere -is:muted
	if (isTagView && tag) {
		viewQuery = `tags:${tag.slug} in:anywhere -is:muted`;
	}
	// Get the view's base query (system views and custom views both have explicit queries now)
	else if (selectedView?.query) {
		viewQuery = selectedView.query;
	} else if (systemView?.query) {
		viewQuery = systemView.query;
	} else if (isSystemView) {
		// Handle frontend-only system views (archive, snoozed)
		if (slug === ARCHIVE_VIEW_SLUG) {
			viewQuery = "archived:true -muted:true";
		} else if (slug === SNOOZED_VIEW_SLUG) {
			viewQuery = "snoozed:true -muted:true";
		}
	}

	// Get query from URL parameter
	const queryParam = url.searchParams.get("query") ?? "";

	// If query param exists, use it; otherwise use the view's query
	const currentQuery = queryParam || viewQuery;

	// Combine with search term if present (deprecated, for backward compatibility)
	const searchTerm = url.searchParams.get("search") ?? "";
	const combinedQuery = [currentQuery, searchTerm].filter(Boolean).join(" ");

	try {
		const notificationsPage = await fetchNotifications(
			{
				page,
				filters: {
					// Use combined query - no more viewSlug, searchTerm, or triageStatuses
					// Always pass query (even if empty) to use new query engine with inbox defaults
					query: combinedQuery,
					filters: quickFilters,
				},
			},
			fetch
		);

		return {
			views,
			tags,
			selectedViewSlug: slug,
			selectedViewId: selectedView?.id ?? slug,
			// baseFilters removed
			initialPage: notificationsPage,
			initialSearchTerm: searchTerm,
			initialQuickFilters: quickFilters,
			initialPageNumber: page,
			initialQuery: currentQuery,
			viewQuery: viewQuery,
			apiError: null,
			tag, // Pass tag if this is a tag view
		};
	} catch (error) {
		console.error("Failed to fetch notifications:", error);

		// Check if this is a network error (API unreachable)
		// These should show full-screen error, not inline
		if (error instanceof ApiUnreachableError || isNetworkError(error)) {
			return {
				views,
				tags,
				selectedViewSlug: slug,
				selectedViewId: selectedView?.id ?? slug,
				initialPage: { items: [], total: 0, page: 1, pageSize: 50 },
				initialSearchTerm: searchTerm,
				initialQuickFilters: quickFilters,
				initialPageNumber: page,
				initialQuery: currentQuery,
				viewQuery: viewQuery,
				apiError: "Unable to reach the API server",
				apiErrorIsInline: false, // Show full-screen error for network issues
				tag,
			};
		}

		// Extract error message from the error
		const errorMessage = error instanceof Error ? error.message : "Failed to load notifications";

		// Query/validation errors should be shown inline (not blocking the whole view)
		return {
			views,
			tags,
			selectedViewSlug: slug,
			selectedViewId: selectedView?.id ?? slug,
			// baseFilters removed
			initialPage: { items: [], total: 0, page: 1, pageSize: 50 },
			initialSearchTerm: searchTerm,
			initialQuickFilters: quickFilters,
			initialPageNumber: page,
			initialQuery: currentQuery,
			viewQuery: viewQuery,
			apiError: errorMessage,
			apiErrorIsInline: true, // Show inline error for query validation errors
			tag, // Pass tag if this is a tag view
		};
	}
};
