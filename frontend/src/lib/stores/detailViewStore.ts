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

import { writable, derived } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import type { Notification, NotificationDetail } from "$lib/api/types";
import type { NotificationStore } from "./notificationStore";

/**
 * Detail View Store
 * Simplified: detailNotificationId is the single source of truth
 * Everything else is derived from it + notificationStore
 */
export function createDetailViewStore(notificationStore: NotificationStore) {
	// SINGLE SOURCE OF TRUTH: Which notification is being viewed
	const detailNotificationId = writable<string | null>(null);

	// UI state flags (not derivable)
	const currentDetail = writable<NotificationDetail | null>(null);
	const detailLoading = writable<boolean>(false);
	const detailShowingStaleData = writable<boolean>(false);
	const detailIsRefreshing = writable<boolean>(false);

	// DERIVED: Is detail open? (true if detailNotificationId is set)
	const detailOpen = derived(
		detailNotificationId,
		($detailNotificationId) => $detailNotificationId !== null
	);

	// DERIVED: The notification object from normalized store
	const currentDetailNotification = derived(
		[detailNotificationId, notificationStore.notificationsById],
		([$detailNotificationId, $notificationsById]) => {
			if (!$detailNotificationId) return null;
			return $notificationsById.get($detailNotificationId) ?? null;
		}
	);

	// DERIVED: Index of the notification in current page (for navigation/focus)
	const detailNotificationIndex = derived(
		[detailNotificationId, notificationStore.pageData],
		([$detailNotificationId, $pageData]) => {
			if (!$detailNotificationId) return null;
			const index = $pageData.items.findIndex(
				(n) => (n.githubId ?? n.id) === $detailNotificationId
			);
			return index !== -1 ? index : null;
		}
	);

	return {
		// Writable stores (source of truth)
		detailNotificationId: detailNotificationId as Writable<string | null>,

		// UI state stores (not derivable)
		currentDetail: currentDetail as Writable<NotificationDetail | null>,
		detailLoading: detailLoading as Writable<boolean>,
		detailShowingStaleData: detailShowingStaleData as Writable<boolean>,
		detailIsRefreshing: detailIsRefreshing as Writable<boolean>,

		// Derived stores (automatically stay in sync)
		detailOpen: detailOpen as Readable<boolean>,
		currentDetailNotification: currentDetailNotification as Readable<Notification | null>,
		detailNotificationIndex: detailNotificationIndex as Readable<number | null>,

		// Actions
		openDetail: (id: string) => {
			detailNotificationId.set(id);
		},
		closeDetail: () => {
			detailNotificationId.set(null);
			// Clear detail data when closing
			currentDetail.set(null);
			detailLoading.set(false);
			detailShowingStaleData.set(false);
			detailIsRefreshing.set(false);
		},
		setDetail: (detail: NotificationDetail | null) => {
			currentDetail.set(detail);
		},
		setLoading: (loading: boolean) => {
			detailLoading.set(loading);
		},
		setShowingStaleData: (showing: boolean) => {
			detailShowingStaleData.set(showing);
		},
		setRefreshing: (refreshing: boolean) => {
			detailIsRefreshing.set(refreshing);
		},
	};
}

export type DetailStore = ReturnType<typeof createDetailViewStore>;
