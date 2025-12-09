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

import { writable } from "svelte/store";
import type { Writable } from "svelte/store";
import type { Notification } from "$lib/api/types";

/**
 * Event Bus
 * Event system for cross-cutting concerns and coordination
 */
export type NotificationEvent =
	| { type: "notification:updated"; notification: Notification }
	| { type: "notification:marked-read"; id: string }
	| { type: "notification:marked-unread"; id: string }
	| { type: "page:changed"; page: number }
	| { type: "detail:opened"; id: string }
	| { type: "detail:closed"; id: string }
	| { type: "sync:completed"; notifications: Notification[] }
	| { type: "action:performed"; action: string; notificationId: string };

/**
 * Create event bus for cross-store coordination
 */
export function createEventBus() {
	const eventBus = writable<NotificationEvent | null>(null);

	return {
		// Expose store
		eventBus: eventBus as Writable<NotificationEvent | null>,

		// Actions
		emit: (event: NotificationEvent) => {
			eventBus.set(event);
			// Reset to null after emitting to allow same event to be emitted again
			// Components should subscribe to the store to react to events
			setTimeout(() => eventBus.set(null), 0);
		},
	};
}

export type EventBus = ReturnType<typeof createEventBus>;
