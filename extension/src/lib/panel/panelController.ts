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

import { derived, get, writable } from "svelte/store";
import type { Notification, NotificationView, Tag } from "$lib/api/types";
import { ApiError, ApiErrorCode, ApiUnreachableError, isNetworkError } from "$lib/api/fetch";
import { fetchViews } from "$lib/api/views";
import { fetchTags } from "$lib/api/tags";
import {
	archiveNotification,
	bulkAssignTag,
	bulkRemoveTag,
	fetchNotifications,
	markNotificationRead,
	markNotificationUnread,
	muteNotification,
	snoozeNotification,
	starNotification,
	unarchiveNotification,
	unfilterNotification,
	unmuteNotification,
	unsnoozeNotification,
	unstarNotification,
} from "$lib/api/notifications";
import { resolveNotificationHtmlUrl } from "$lib/utils/githubUrls";
import type {
	DropdownSource,
	DropdownState,
	NotificationPageController,
	PanelError,
} from "$lib/state/types";
import { storageGet, storageSet } from "$lib/ext/browser";
import { isBackendReachable } from "$lib/ext/backendUrl";

const SELECTED_SLUG_KEY = "octobud:selectedViewSlug";
const PAGE_SIZE = 30;

/** Turns any thrown value into the three cases the panel renders differently. */
function toPanelError(error: unknown): PanelError {
	if (error instanceof ApiUnreachableError || isNetworkError(error)) {
		return { kind: "unreachable", message: "Can't reach Octobud." };
	}
	if (error instanceof ApiError && error.code === ApiErrorCode.NotConnected) {
		return { kind: "not_connected", message: error.message };
	}
	return {
		kind: "request",
		message: error instanceof Error ? error.message : "Something went wrong.",
	};
}

function keyOf(notification: Notification): string {
	return notification.githubId ?? notification.id;
}

export function createPanelController(): NotificationPageController {
	const views = writable<NotificationView[]>([]);
	const tags = writable<Tag[]>([]);
	const selectedSlug = writable<string>("");
	const items = writable<Notification[]>([]);
	const total = writable(0);
	const page = writable(1);
	const loading = writable(false);
	const loaded = writable(false);
	const error = writable<PanelError | null>(null);
	const snoozeDropdownState = writable<DropdownState | null>(null);
	const tagDropdownState = writable<DropdownState | null>(null);
	const customSnoozeNotificationId = writable<string | null>(null);

	const totalPages = derived([total], ([$total]) => Math.max(1, Math.ceil($total / PAGE_SIZE)));

	function queryForSlug(slug: string): string {
		return get(views).find((view) => view.slug === slug)?.query ?? "";
	}

	async function loadPage(targetPage: number): Promise<void> {
		const slug = get(selectedSlug);
		if (!slug) return;

		loading.set(true);
		try {
			const result = await fetchNotifications({
				page: targetPage,
				pageSize: PAGE_SIZE,
				filters: { query: queryForSlug(slug) },
			});
			items.set(result.items);
			total.set(result.total);
			page.set(result.page);
			error.set(null);
		} catch (caught) {
			error.set(toPanelError(caught));
			items.set([]);
			total.set(0);
		} finally {
			loading.set(false);
			loaded.set(true);
		}
	}

	async function loadViews(): Promise<void> {
		views.set(await fetchViews());
	}

	/**
	 * Re-reads the list and the view list after a successful action. An action
	 * can move an item out of the current view (archiving from the inbox) and
	 * always changes an unread count, so a refetch is what keeps the panel
	 * honest. It is a local SQLite query, so the round trip is cheap.
	 */
	async function resync(): Promise<void> {
		await Promise.all([loadViews(), loadPage(get(page))]);
	}

	/** Applies a field patch to one row in place, returning a rollback function. */
	function patchLocally(notification: Notification, patch: Partial<Notification>): () => void {
		const key = keyOf(notification);
		const previous = get(items);
		items.set(previous.map((item) => (keyOf(item) === key ? { ...item, ...patch } : item)));
		return () => items.set(previous);
	}

	/**
	 * Optimistically patches the row, calls the API, then resyncs. On failure the
	 * row snaps back and the reason surfaces, rather than the panel quietly
	 * showing state the backend never accepted.
	 */
	async function act(
		notification: Notification,
		patch: Partial<Notification>,
		request: () => Promise<unknown>
	): Promise<void> {
		const rollback = patchLocally(notification, patch);
		try {
			await request();
		} catch (caught) {
			rollback();
			error.set(toPanelError(caught));
			return;
		}
		await resync();
	}

	async function selectView(slug: string): Promise<void> {
		selectedSlug.set(slug);
		await storageSet(SELECTED_SLUG_KEY, slug);
		await loadPage(1);
	}

	/** Last used view, else the backend's default, else the first one. */
	async function resolveInitialSlug(): Promise<string> {
		const available = get(views);
		const stored = await storageGet<string>(SELECTED_SLUG_KEY);
		if (stored && available.some((view) => view.slug === stored)) {
			return stored;
		}
		return available.find((view) => view.isDefault)?.slug ?? available[0]?.slug ?? "inbox";
	}

	return {
		stores: {
			views,
			tags,
			selectedSlug,
			items,
			total,
			page,
			totalPages,
			loading,
			loaded,
			error,
			snoozeDropdownState,
			tagDropdownState,
			customSnoozeNotificationId,
		},

		actions: {
			async initialize() {
				loading.set(true);
				if (!(await isBackendReachable())) {
					error.set({ kind: "unreachable", message: "Can't reach Octobud." });
					loading.set(false);
					return;
				}
				try {
					await loadViews();
					// Tags only drive the tag dropdown; failing to load them should not
					// blank the panel.
					tags.set(await fetchTags().catch(() => []));
					error.set(null);
				} catch (caught) {
					error.set(toPanelError(caught));
					loading.set(false);
					return;
				}
				loading.set(false);
				selectedSlug.set(await resolveInitialSlug());
				await loadPage(1);
			},

			async refresh() {
				await resync();
			},

			selectView,

			async goToPage(target: number) {
				const clamped = Math.min(Math.max(1, target), get(totalPages));
				await loadPage(clamped);
			},

			async openInGitHub(notification) {
				const url = resolveNotificationHtmlUrl(notification);
				if (url) {
					window.open(url, "_blank", "noopener,noreferrer");
				}
				if (!notification.isRead) {
					await act(notification, { isRead: true }, () =>
						markNotificationRead(keyOf(notification))
					);
				}
			},

			async markRead(notification) {
				const next = !notification.isRead;
				await act(notification, { isRead: next }, () =>
					next
						? markNotificationRead(keyOf(notification))
						: markNotificationUnread(keyOf(notification))
				);
			},

			async archive(notification) {
				const next = !notification.archived;
				await act(notification, { archived: next }, () =>
					next
						? archiveNotification(keyOf(notification))
						: unarchiveNotification(keyOf(notification))
				);
			},

			async mute(notification) {
				await act(notification, { muted: true }, () => muteNotification(keyOf(notification)));
			},

			async unmute(notification) {
				await act(notification, { muted: false }, () => unmuteNotification(keyOf(notification)));
			},

			async star(notification) {
				await act(notification, { starred: true }, () => starNotification(keyOf(notification)));
			},

			async unstar(notification) {
				await act(notification, { starred: false }, () => unstarNotification(keyOf(notification)));
			},

			async unfilter(notification) {
				await act(notification, { filtered: false }, () =>
					unfilterNotification(keyOf(notification))
				);
			},

			async snooze(notification, until) {
				await act(notification, { snoozedUntil: until }, () =>
					snoozeNotification(keyOf(notification), until)
				);
			},

			async unsnooze(notification) {
				await act(notification, { snoozedUntil: undefined }, () =>
					unsnoozeNotification(keyOf(notification))
				);
			},

			async assignTag(githubId, tagId) {
				const tag = get(tags).find((candidate) => candidate.id === tagId);
				const notification = get(items).find((item) => keyOf(item) === githubId);
				if (!tag || !notification) return;

				await act(notification, { tags: [...(notification.tags ?? []), tag] }, () =>
					bulkAssignTag([githubId], tagId)
				);
			},

			async removeTag(githubId, tagId) {
				const notification = get(items).find((item) => keyOf(item) === githubId);
				if (!notification) return;

				await act(
					notification,
					{ tags: (notification.tags ?? []).filter((candidate) => candidate.id !== tagId) },
					() => bulkRemoveTag([githubId], tagId)
				);
			},

			openSnoozeDropdown(notificationId: string, source: DropdownSource) {
				tagDropdownState.set(null);
				snoozeDropdownState.set({ notificationId, source });
			},

			closeSnoozeDropdown() {
				snoozeDropdownState.set(null);
			},

			openTagDropdown(notificationId: string, source: DropdownSource) {
				snoozeDropdownState.set(null);
				tagDropdownState.set({ notificationId, source });
			},

			closeTagDropdown() {
				tagDropdownState.set(null);
			},

			openCustomSnoozeDialog(notificationId: string) {
				snoozeDropdownState.set(null);
				customSnoozeNotificationId.set(notificationId);
			},

			closeCustomSnoozeDialog() {
				customSnoozeNotificationId.set(null);
			},
		},
	};
}
