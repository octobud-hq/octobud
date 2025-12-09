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
import type { NotificationThreadItem, NotificationTimelineResponse } from "$lib/api/types";
import { fetchNotificationTimeline } from "$lib/api/notifications";

export interface CommentsPagination {
	total: number;
	page: number;
	perPage: number;
	hasMore: boolean;
}

export interface CommentsControllerStores {
	items: Writable<NotificationThreadItem[]>;
	isLoading: Writable<boolean>;
	error: Writable<string | null>;
	pagination: Writable<CommentsPagination>;
}

export interface CommentsControllerActions {
	loadComments: (githubId: string, perPage: number) => Promise<void>;
	loadMoreComments: (githubId: string, perPage: number) => Promise<void>;
	reset: () => void;
}

export interface CommentsController {
	stores: CommentsControllerStores;
	actions: CommentsControllerActions;
}

/**
 * Creates a controller for managing comment/review loading and pagination
 */
export function createCommentsController(): CommentsController {
	const items = writable<NotificationThreadItem[]>([]);
	const isLoading = writable<boolean>(false);
	const error = writable<string | null>(null);
	const pagination = writable<CommentsPagination>({
		total: 0,
		page: 0,
		perPage: 5,
		hasMore: false,
	});

	let currentGithubId: string | null = null;

	async function loadComments(githubId: string, perPage: number = 5): Promise<void> {
		currentGithubId = githubId;
		isLoading.set(true);
		error.set(null);

		try {
			const response: NotificationTimelineResponse = await fetchNotificationTimeline(
				githubId,
				perPage,
				1
			);

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
			console.error("Failed to load comments:", err);
			error.set(err instanceof Error ? err.message : "Failed to load comments");
		} finally {
			isLoading.set(false);
		}
	}

	async function loadMoreComments(githubId: string, perPage: number = 5): Promise<void> {
		const currentPagination = get(pagination);
		const nextPage = currentPagination.page + 1;

		isLoading.set(true);
		error.set(null);

		try {
			const response: NotificationTimelineResponse = await fetchNotificationTimeline(
				githubId,
				perPage,
				nextPage
			);

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
			console.error("Failed to load more comments:", err);
			error.set(err instanceof Error ? err.message : "Failed to load more comments");
		} finally {
			isLoading.set(false);
		}
	}

	function reset(): void {
		items.set([]);
		isLoading.set(false);
		error.set(null);
		pagination.set({
			total: 0,
			page: 0,
			perPage: 5,
			hasMore: false,
		});
		currentGithubId = null;
	}

	return {
		stores: {
			items,
			isLoading,
			error,
			pagination,
		},
		actions: {
			loadComments,
			loadMoreComments,
			reset,
		},
	};
}
