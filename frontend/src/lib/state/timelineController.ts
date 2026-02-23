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

import { writable, get } from "svelte/store";
import type { Writable } from "svelte/store";
import type { NotificationTimelineItem, NotificationTimelineResponse } from "$lib/api/types";
import { fetchNotificationTimeline } from "$lib/api/notifications";

export interface TimelinePagination {
	total: number;
	page: number;
	perPage: number;
	hasMore: boolean;
}

export interface TimelineControllerStores {
	items: Writable<NotificationTimelineItem[]>;
	isLoading: Writable<boolean>;
	error: Writable<string | null>;
	pagination: Writable<TimelinePagination>;
	hasAttemptedAutoLoad: Writable<boolean>;
}

export interface TimelineControllerActions {
	autoLoadTimeline: (githubId: string, perPage: number) => Promise<void>;
	loadTimeline: (githubId: string, perPage: number) => Promise<void>;
	loadMoreTimeline: (githubId: string, perPage: number) => Promise<void>;
	refreshTimeline: (githubId: string, perPage?: number) => Promise<void>;
	reset: () => void;
}

export interface TimelineController {
	stores: TimelineControllerStores;
	actions: TimelineControllerActions;
}

/**
 * Creates a controller for managing timeline event loading and pagination
 */
export function createTimelineController(): TimelineController {
	const items = writable<NotificationTimelineItem[]>([]);
	const isLoading = writable<boolean>(false);
	const error = writable<string | null>(null);
	const pagination = writable<TimelinePagination>({
		total: 0,
		page: 0,
		perPage: 10,
		hasMore: false,
	});
	const hasAttemptedAutoLoad = writable<boolean>(false);

	let currentGithubId: string | null = null;

	async function autoLoadTimeline(githubId: string, perPage: number = 10): Promise<void> {
		// Mark that we've attempted auto-load
		hasAttemptedAutoLoad.set(true);

		currentGithubId = githubId;
		isLoading.set(true);
		error.set(null);

		try {
			const response: NotificationTimelineResponse = await fetchNotificationTimeline(
				githubId,
				perPage,
				1
			);

			// Abort if githubId changed while loading (detail was closed/opened)
			if (currentGithubId !== githubId) {
				return;
			}

			// Reverse items to show oldest first (matching GitHub)
			const reversedItems = [...response.items].reverse();

			items.set(reversedItems);
			pagination.set({
				total: response.total,
				page: response.page,
				perPage: response.perPage,
				hasMore: response.hasMore,
			});
		} catch (err) {
			// Only set error if this is still the current notification
			if (currentGithubId === githubId) {
				console.error("[TimelineController] Failed to auto-load timeline:", err);
				error.set(err instanceof Error ? err.message : "Failed to load activity");
			}
		} finally {
			// Only clear loading if this is still the current notification
			if (currentGithubId === githubId) {
				isLoading.set(false);
			}
		}
	}

	async function loadTimeline(githubId: string, perPage: number = 10): Promise<void> {
		hasAttemptedAutoLoad.set(true);
		currentGithubId = githubId;
		isLoading.set(true);
		error.set(null);

		try {
			const response: NotificationTimelineResponse = await fetchNotificationTimeline(
				githubId,
				perPage,
				1
			);

			// Abort if githubId changed while loading (detail was closed/opened)
			if (currentGithubId !== githubId) {
				return;
			}

			// Reverse items to show oldest first (matching GitHub)
			const reversedItems = [...response.items].reverse();

			items.set(reversedItems);
			pagination.set({
				total: response.total,
				page: response.page,
				perPage: response.perPage,
				hasMore: response.hasMore,
			});
		} catch (err) {
			// Only set error if this is still the current notification
			if (currentGithubId === githubId) {
				console.error("Failed to load timeline:", err);
				error.set(err instanceof Error ? err.message : "Failed to load activity");
			}
		} finally {
			// Only clear loading if this is still the current notification
			if (currentGithubId === githubId) {
				isLoading.set(false);
			}
		}
	}

	async function loadMoreTimeline(githubId: string, perPage: number = 10): Promise<void> {
		const currentPagination = get(pagination);
		const nextPage = currentPagination.page + 1;

		currentGithubId = githubId;
		isLoading.set(true);
		error.set(null);

		try {
			const response: NotificationTimelineResponse = await fetchNotificationTimeline(
				githubId,
				perPage,
				nextPage
			);

			// Abort if githubId changed while loading (detail was closed/opened)
			if (currentGithubId !== githubId) {
				return;
			}

			// Reverse new items to show oldest first
			const reversedNewItems = [...response.items].reverse();

			// Prepend older items to the beginning of the list (since we're showing oldest first)
			const currentItems = get(items);
			items.set([...reversedNewItems, ...currentItems]);

			pagination.set({
				total: response.total,
				page: response.page,
				perPage: response.perPage,
				hasMore: response.hasMore,
			});
		} catch (err) {
			// Only set error if this is still the current notification
			if (currentGithubId === githubId) {
				console.error("Failed to load more timeline:", err);
				error.set(err instanceof Error ? err.message : "Failed to load more activity");
			}
		} finally {
			// Only clear loading if this is still the current notification
			if (currentGithubId === githubId) {
				isLoading.set(false);
			}
		}
	}

	/**
	 * Silently refresh the timeline by fetching page 1 and appending any new items.
	 * Used when polling detects the notification was updated while the timeline is open.
	 * Does not show loading state — this is a background refresh.
	 */
	async function refreshTimeline(githubId: string, perPage: number = 10): Promise<void> {
		const currentItems = get(items);
		if (currentItems.length === 0) return; // Don't refresh before initial load

		try {
			const response: NotificationTimelineResponse = await fetchNotificationTimeline(
				githubId,
				perPage,
				1
			);

			// Abort if githubId changed while loading
			if (currentGithubId !== githubId) return;

			// Reverse to match oldest-first display order
			const reversedNewItems = [...response.items].reverse();

			// Deduplicate: only keep items not already in the list.
			// Uses a composite key because some event types (committed, merged)
			// have null IDs — deduplicating by id alone would treat them all as the same item.
			// NOTE: The type-timestamp fallback can collide if two events of the same type
			// share the exact same second (e.g. simultaneous commits). This is acceptable
			// since it only affects the rare case of null-ID events at identical timestamps.
			const itemKey = (item: NotificationTimelineItem) =>
				item.id != null ? String(item.id) : `${item.type}-${item.timestamp || ""}`;
			const existingKeys = new Set(currentItems.map(itemKey));
			const newItems = reversedNewItems.filter((item) => !existingKeys.has(itemKey(item)));

			if (newItems.length > 0) {
				// Append new items to the end (they're the newest)
				items.update((current) => [...current, ...newItems]);

				// Update total count but preserve page/hasMore (which track the oldest loaded page)
				pagination.update((p) => ({ ...p, total: response.total }));
			}
		} catch (err) {
			// Silent failure for background refresh
			if (currentGithubId === githubId) {
				console.error("[TimelineController] Failed to refresh timeline:", err);
			}
		}
	}

	function reset(): void {
		items.set([]);
		isLoading.set(false);
		error.set(null);
		pagination.set({
			total: 0,
			page: 0,
			perPage: 10,
			hasMore: false,
		});
		hasAttemptedAutoLoad.set(false);
		currentGithubId = null;
	}

	return {
		stores: {
			items,
			isLoading,
			error,
			pagination,
			hasAttemptedAutoLoad,
		},
		actions: {
			autoLoadTimeline,
			loadTimeline,
			loadMoreTimeline,
			refreshTimeline,
			reset,
		},
	};
}
