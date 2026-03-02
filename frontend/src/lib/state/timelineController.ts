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
import {
	fetchNotificationTimeline,
	fetchReviewComments,
	fetchPRCommits,
	type ReviewCommentsByReviewId,
	type PRCommitAuthorsBySHA,
} from "$lib/api/notifications";

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
	loadReviewComments: (githubId: string) => Promise<void>;
	loadPRCommits: (githubId: string) => Promise<void>;
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

	// Review comments fetched in parallel — merged into items when both are ready
	let reviewCommentsData: ReviewCommentsByReviewId | null = null;

	// PR commit authors fetched in parallel — merged into committed items when both are ready
	let commitAuthorsData: PRCommitAuthorsBySHA | null = null;

	/**
	 * Merge cached review comments into the items store.
	 * Idempotent — safe to call after either timeline items or review comments arrive.
	 */
	function mergeReviewComments(): void {
		if (!reviewCommentsData) return;
		const currentItems = get(items);
		if (currentItems.length === 0) return;

		const data = reviewCommentsData;
		let changed = false;

		const merged = currentItems.map((item) => {
			if (item.type !== "reviewed") return item;
			const reviewId = item.id != null ? String(item.id) : null;
			if (!reviewId) return item;
			const comments = data[reviewId];
			if (!comments || comments.length === 0) return item;
			// Skip if already merged
			if (item.reviewComments && item.reviewComments.length > 0) return item;
			changed = true;
			return { ...item, reviewComments: comments, reviewCommentCount: comments.length };
		});

		if (changed) {
			items.set(merged);
		}
	}

	/**
	 * Merge cached PR commit authors into committed timeline items.
	 * Idempotent — safe to call after either timeline items or commit authors arrive.
	 */
	function mergeCommitAuthors(): void {
		if (!commitAuthorsData) return;
		const currentItems = get(items);
		if (currentItems.length === 0) return;

		const data = commitAuthorsData;
		let changed = false;

		const merged = currentItems.map((item) => {
			if (item.type !== "committed" || !item.sha) return item;
			const author = data[item.sha];
			if (!author) return item;
			// Skip if already enriched
			if (item.author?.avatarUrl) return item;
			changed = true;
			return {
				...item,
				author: { login: author.login, avatarUrl: author.avatarUrl },
			};
		});

		if (changed) {
			items.set(merged);
		}
	}

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
			mergeReviewComments();
			mergeCommitAuthors();
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
			mergeReviewComments();
			mergeCommitAuthors();
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
			mergeReviewComments();
			mergeCommitAuthors();

			pagination.set({
				total: response.total,
				page: response.page,
				perPage: response.perPage,
				// If the page returned no items, there's nothing more to load
				hasMore: response.items.length > 0 && response.hasMore,
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
	 *
	 * Uses a guard to prevent concurrent calls (the direct handler and the reactive
	 * effectiveSortDate block can both trigger this for the same update).
	 */
	let refreshInProgress = false;

	async function refreshTimeline(githubId: string, perPage: number = 10): Promise<void> {
		if (refreshInProgress) return;

		const currentItems = get(items);
		if (currentItems.length === 0) return; // Don't refresh before initial load

		refreshInProgress = true;
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
			// Re-read the store (not the stale capture) so concurrent callers
			// that slipped past the guard still see the latest items.
			// Uses a composite key because some event types (committed, merged)
			// have null IDs — deduplicating by id alone would treat them all as the same item.
			const itemKey = (item: NotificationTimelineItem) =>
				item.id != null ? String(item.id) : `${item.type}-${item.timestamp || ""}`;
			const latestItems = get(items);
			const existingKeys = new Set(latestItems.map(itemKey));
			const newItems = reversedNewItems.filter((item) => !existingKeys.has(itemKey(item)));

			if (newItems.length > 0) {
				// Append new items to the end (they're the newest)
				items.update((current) => [...current, ...newItems]);

				// Re-fetch review comments and commit authors if they were previously
				// loaded (i.e. this is a PR). New activity may include new reviews/commits
				// that weren't in the initial cache.
				if (reviewCommentsData) {
					loadReviewComments(githubId, true);
				} else {
					mergeReviewComments();
				}
				if (commitAuthorsData) {
					loadPRCommits(githubId, true);
				} else {
					mergeCommitAuthors();
				}

				// Update total count but preserve page/hasMore (which track the oldest loaded page)
				pagination.update((p) => ({ ...p, total: response.total }));
			}
		} catch (err) {
			// Silent failure for background refresh
			if (currentGithubId === githubId) {
				console.error("[TimelineController] Failed to refresh timeline:", err);
			}
		} finally {
			refreshInProgress = false;
		}
	}

	/**
	 * Fetch review comments for a PR in the background.
	 * Designed to be fired in parallel with timeline loading — when results arrive,
	 * they are merged into existing items (or cached for when items arrive).
	 */
	async function loadReviewComments(githubId: string, force: boolean = false): Promise<void> {
		// Skip if already fetched (unless forced)
		if (reviewCommentsData && !force) {
			mergeReviewComments();
			return;
		}

		try {
			const data = await fetchReviewComments(githubId);

			// Abort if githubId changed while loading
			if (currentGithubId !== githubId) return;

			reviewCommentsData = data;
			mergeReviewComments();
		} catch (err) {
			// Non-critical — log and continue
			if (currentGithubId === githubId) {
				console.error("[TimelineController] Failed to load review comments:", err);
			}
		}
	}

	/**
	 * Fetch PR commit authors in the background.
	 * Designed to be fired in parallel with timeline loading — when results arrive,
	 * committed events are enriched with GitHub login + avatar.
	 */
	async function loadPRCommits(githubId: string, force: boolean = false): Promise<void> {
		// Skip if already fetched (unless forced)
		if (commitAuthorsData && !force) {
			mergeCommitAuthors();
			return;
		}

		try {
			const data = await fetchPRCommits(githubId);

			// Abort if githubId changed while loading
			if (currentGithubId !== githubId) return;

			commitAuthorsData = data;
			mergeCommitAuthors();
		} catch (err) {
			// Non-critical — log and continue
			if (currentGithubId === githubId) {
				console.error("[TimelineController] Failed to load PR commits:", err);
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
		refreshInProgress = false;
		reviewCommentsData = null;
		commitAuthorsData = null;
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
			loadReviewComments,
			loadPRCommits,
			reset,
		},
	};
}
