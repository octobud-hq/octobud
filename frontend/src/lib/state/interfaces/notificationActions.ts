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

import type { Notification } from "$lib/api/types";

/**
 * Notification Actions
 * Actions related to individual notification operations
 */
export interface NotificationActions {
	markRead: (notification: Notification) => Promise<void>;
	/**
	 * Lightweight mark-read for auto-mark-read scenarios (split mode timer,
	 * single mode detail open, new-activity-viewed).
	 *
	 * Unlike the full `markRead` pipeline, this does NOT replace the notification
	 * with the server response. This avoids a race condition where the server
	 * response carries a stale `timelineLastSeenAt` (because the
	 * `updateTimelineLastSeen` DB write may still be in-flight), which would
	 * revert the optimistic update and cause the "New Activity" button to
	 * reappear.
	 *
	 * What it does:
	 * 1. Optimistic UI update (isRead: true)
	 * 2. Refresh sidebar/view unread counts
	 * 3. Fire-and-forget API call with pending-mark-read guard
	 * 4. Revert on error
	 */
	softMarkRead: (notification: Notification) => void;
	archive: (notification: Notification) => Promise<void>;
	mute: (notification: Notification) => Promise<void>;
	unmute: (notification: Notification) => Promise<void>;
	snooze: (notification: Notification, until: string) => Promise<void>;
	unsnooze: (notification: Notification) => Promise<void>;
	star: (notification: Notification) => Promise<void>;
	unstar: (notification: Notification) => Promise<void>;
	unfilter: (notification: Notification) => Promise<void>;
	assignTag: (githubId: string, tagId: string, tagName?: string) => Promise<void>;
	removeTag: (githubId: string, tagId: string, tagName?: string) => Promise<void>;
}
