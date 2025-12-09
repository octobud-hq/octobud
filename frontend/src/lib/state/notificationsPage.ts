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

import type { Notification, NotificationPage } from "$lib/api/types";

export function updatePageItem(
	page: NotificationPage,
	id: string,
	mutate: (notification: Notification) => Notification | null
): NotificationPage {
	const nextItems: Notification[] = [];
	let changed = false;
	let removed = false;

	for (const item of page.items) {
		if (item.id === id) {
			const result = mutate(item);
			if (result) {
				nextItems.push(result);
			} else {
				removed = true;
			}
			changed = true;
		} else {
			nextItems.push(item);
		}
	}

	if (!changed) {
		return page;
	}

	return {
		...page,
		items: nextItems,
		total: removed ? Math.max(0, page.total - 1) : page.total,
	};
}

export function applyNotificationUpdateToPage(
	page: NotificationPage,
	updated: Notification
): NotificationPage {
	return updatePageItem(page, updated.id, () => updated);
}

export function applyNotificationUpdatesToPage(
	page: NotificationPage,
	updates: Notification[]
): NotificationPage {
	return updates.reduce((next, current) => applyNotificationUpdateToPage(next, current), page);
}

export function selectKey(notification: Notification): string | null {
	return notification.githubId ?? notification.id ?? null;
}
