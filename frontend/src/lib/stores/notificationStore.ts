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
import type { Notification, NotificationPage } from "$lib/api/types";
import { applyNotificationUpdateToPage, updatePageItem } from "$lib/state/notificationsPage";

/**
 * Notification Store
 * Single source of truth for notification data with normalized storage
 */
export function createNotificationStore(initialPage: NotificationPage) {
	// Normalized storage - all notifications by ID
	const notificationsById = writable<Map<string, Notification>>(new Map());

	// Current page data (references normalized store)
	const pageData = writable<NotificationPage>(initialPage);

	// Initialize normalized store with initial page data
	const initialMap = new Map<string, Notification>();
	for (const item of initialPage.items) {
		const key = item.githubId ?? item.id;
		if (key) {
			initialMap.set(key, item);
		}
	}
	notificationsById.set(initialMap);

	/**
	 * Update a notification in both normalized store and pageData
	 * Preserves optimistic read status to prevent refresh from overriding it
	 */
	function updateNotification(updated: Notification): void {
		const key = updated.githubId ?? updated.id;
		if (!key) return;

		// Check if we have an existing notification with optimistic read status
		const currentMap = get(notificationsById);
		const existing = currentMap.get(key);

		// If the existing notification is optimistically marked as read (but the updated one isn't),
		// preserve the optimistic read status. This prevents refresh from overriding it.
		let finalNotification = updated;
		if (existing && existing.isRead && !updated.isRead) {
			finalNotification = { ...updated, isRead: true };
		}

		// Update normalized store
		notificationsById.update((map) => {
			map.set(key, finalNotification);
			return map;
		});

		// Update pageData if notification is on current page
		pageData.update((data) => applyNotificationUpdateToPage(data, finalNotification));
	}

	/**
	 * Set new page data and update normalized store
	 */
	function setPageData(newPageData: NotificationPage): void {
		// Update normalized store with all items from new page
		notificationsById.update((map) => {
			for (const item of newPageData.items) {
				const key = item.githubId ?? item.id;
				if (key) {
					map.set(key, item);
				}
			}
			return map;
		});

		// Update pageData
		pageData.set(newPageData);
	}

	/**
	 * Get notification from normalized store
	 */
	function getNotification(id: string): Notification | null {
		const map = get(notificationsById);
		return map.get(id) ?? null;
	}

	/**
	 * Remove a notification optimistically from both normalized store and pageData
	 * Returns the removed notification for potential rollback, or null if not found
	 * @param id - The notification key (githubId or id)
	 */
	function removeNotification(id: string): Notification | null {
		const currentMap = get(notificationsById);
		const notification = currentMap.get(id);

		if (!notification) {
			return null;
		}

		// Remove from normalized store
		notificationsById.update((map) => {
			const newMap = new Map(map);
			newMap.delete(id);
			return newMap;
		});

		// Remove from pageData by filtering out the matching item
		// Match by both githubId and id to handle all cases
		pageData.update((data) => {
			const nextItems: Notification[] = [];
			let removed = false;

			for (const item of data.items) {
				const itemKey = item.githubId ?? item.id;
				// Match by key (githubId or id) or by explicit id match
				if (itemKey === id || item.id === notification.id) {
					removed = true;
					// Skip this item (don't add to nextItems)
				} else {
					nextItems.push(item);
				}
			}

			if (!removed) {
				// Item not in current page, no change needed
				return data;
			}

			return {
				...data,
				items: nextItems,
				total: Math.max(0, data.total - 1),
			};
		});

		return notification;
	}

	/**
	 * Restore a notification that was optimistically removed
	 * Used for error rollback
	 */
	function restoreNotification(notification: Notification): void {
		const key = notification.githubId ?? notification.id;
		if (!key) return;

		// Restore to normalized store
		notificationsById.update((map) => {
			const newMap = new Map(map);
			newMap.set(key, notification);
			return newMap;
		});

		// Restore to pageData - need to restore the item and increment total
		pageData.update((data) => {
			// Check if notification is already in items
			const existsInItems = data.items.some((item) => (item.githubId ?? item.id) === key);

			if (existsInItems) {
				// Already exists, just update it
				return applyNotificationUpdateToPage(data, notification);
			}

			// Not in items, need to restore it
			// We need to figure out where it should go - for simplicity, append and increment total
			// In practice, the refresh will handle the correct ordering
			const newItems = [...data.items, notification];
			return {
				...data,
				items: newItems,
				total: data.total + 1,
			};
		});
	}

	return {
		// Expose stores
		notificationsById: notificationsById as Writable<Map<string, Notification>>,
		pageData: pageData as Writable<NotificationPage>,

		// Actions
		updateNotification,
		setPageData,
		getNotification,
		removeNotification,
		restoreNotification,
	};
}

export type NotificationStore = ReturnType<typeof createNotificationStore>;
