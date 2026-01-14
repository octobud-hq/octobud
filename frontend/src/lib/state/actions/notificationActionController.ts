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

import { get } from "svelte/store";
import type { Notification } from "$lib/api/types";
import type { NotificationActions } from "../interfaces/notificationActions";
import {
	markNotificationRead,
	markNotificationUnread,
	archiveNotification,
	unarchiveNotification,
	muteNotification,
	unmuteNotification,
	snoozeNotification,
	unsnoozeNotification,
	starNotification,
	unstarNotification,
	unfilterNotification,
} from "$lib/api/notifications";
import { assignTagToNotification, removeTagFromNotification } from "$lib/api/tags";
import { toastStore } from "$lib/stores/toastStore";
import { undoStore } from "$lib/stores/undoStore";
import {
	createUndoableAction,
	toRecentActionNotification,
	type UndoableActionType,
} from "$lib/undo/types";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { SelectionStore } from "../../stores/selectionStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { ControllerOptions } from "../interfaces/common";
import type { SharedHelpers } from "./sharedHelpers";
import type { OptimisticUpdateHelpers } from "./optimisticUpdates";

interface StoreCollection {
	notificationStore: NotificationStore;
	paginationStore: PaginationStore;
	detailStore: DetailStore;
	selectionStore: SelectionStore;
	keyboardStore: KeyboardStore;
}

/**
 * Notification Action Controller
 * Manages individual notification actions with optimistic updates
 */
export function createNotificationActionController(
	stores: StoreCollection,
	options: ControllerOptions,
	optimisticUpdateHelpers: OptimisticUpdateHelpers,
	sharedHelpers: SharedHelpers
): NotificationActions {
	const { notificationStore, paginationStore, detailStore, selectionStore, keyboardStore } = stores;

	/**
	 * Unified action handler wrapper
	 * Handles API calls, error handling, and delegates to optimistic update handler
	 */
	async function handleAction(params: {
		notification: Notification;
		actionName:
			| "archive"
			| "unarchive"
			| "mute"
			| "unmute"
			| "snooze"
			| "unsnooze"
			| "markRead"
			| "markUnread"
			| "star"
			| "unstar"
			| "assignTag"
			| "unassignTag"
			| "unfilter";
		performAction: (key: string) => Promise<Notification>;
		shouldRemoveFromSelection?: boolean;
		successToast?: string | null;
		errorToast?: string;
		shouldCloseDetailInSingleMode?: boolean;
		/** Configuration for making this action undoable */
		undoConfig?: {
			/** The action type for undo purposes */
			actionType: UndoableActionType;
			/** Function to perform the inverse operation */
			performUndo: () => Promise<void>;
			/** Optional metadata (e.g., tagId, snoozedUntil) */
			metadata?: {
				tagId?: string;
				tagName?: string;
				snoozedUntil?: string | null;
			};
		};
	}): Promise<void> {
		const {
			notification,
			actionName,
			performAction,
			shouldRemoveFromSelection = false,
			successToast = null,
			errorToast = "Failed to perform action",
			shouldCloseDetailInSingleMode = false,
			undoConfig,
		} = params;

		const key = notification.githubId ?? notification.id;
		if (!key) return;

		try {
			// SINGLE SOURCE OF TRUTH: Always get the notification from the normalized store
			// This ensures we're working with the most up-to-date data, not stale props
			const notificationsById = get(notificationStore.notificationsById);
			const pageData = get(notificationStore.pageData);

			// Try to get the notification from normalized store first
			let notificationFromStore = notificationsById.get(key);

			// Fallback to pageData if not in normalized store
			if (!notificationFromStore) {
				notificationFromStore = pageData.items.find((item) => (item.githubId ?? item.id) === key);
			}

			// Use store notification if available, otherwise fall back to passed notification
			const currentNotification = notificationFromStore ?? notification;

			// Check if notification is in current page
			let currentIndex = pageData.items.findIndex(
				(item) => (item.githubId && item.githubId === key) || item.id === currentNotification.id
			);

			// If not found in current page, check if it's in the detail view
			// (notification might be on a different page but detail is still open)
			if (currentIndex === -1) {
				const detailId = get(detailStore.detailNotificationId);
				if (detailId === key) {
					// Notification is in detail view but not in current page
					// Use -1 to indicate it's not in current page, but we still need to handle detail
					currentIndex = -1;
				}
			}

			// Capture focus state BEFORE any async operations
			const capturedFocusIndex = get(keyboardStore.keyboardFocusIndex);

			// Get action hints from the current notification (single source of truth)
			// Prefer pageData item if it exists (most fresh), otherwise use store notification
			const pageItem = pageData.items.find((item) => (item.githubId ?? item.id) === key);

			// Use page item if it has actionHints, otherwise use currentNotification
			const notificationWithHints = pageItem?.actionHints ? pageItem : currentNotification;
			const dismissedOn = notificationWithHints.actionHints?.dismissedOn ?? [];

			// Simply check if the action is in the dismissedOn list
			const willBeRemoved = dismissedOn.includes(actionName);

			// Perform the actual API call using the key from the current notification
			const updated = await performAction(key);

			// Handle success: show toast with undo support or regular toast
			if (undoConfig && successToast) {
				// Create undoable action and push to undo store (shows toast with undo button)
				// Use toRecentActionNotification to store only essential fields for lighter storage
				const undoableAction = createUndoableAction({
					type: undoConfig.actionType,
					notifications: [toRecentActionNotification(currentNotification)],
					metadata: undoConfig.metadata,
					performUndo: undoConfig.performUndo,
				});
				undoStore.pushAction(undoableAction);
			} else if (successToast) {
				// Show regular toast without undo
				toastStore.info(successToast);
			}

			// Use unified post-action handler (handles optimistic updates, selection removal, and refresh)
			// Note: onRefreshViewCounts is called inside handlePostAction for non-dismissive actions
			// Pass the currentNotification from store (single source of truth) instead of the passed notification
			await optimisticUpdateHelpers.handlePostAction({
				notification: currentNotification,
				currentIndex,
				willBeRemoved,
				updated,
				capturedFocusIndex,
				shouldCloseDetailInSingleMode,
				shouldRemoveFromSelection,
			});
		} catch (err) {
			console.error(errorToast, err);
			// Show error toast
			if (errorToast) {
				toastStore.error(errorToast);
			}
			// Optionally refresh to recover from error
			await options.onRefresh?.();
		}
	}

	async function markRead(notification: Notification): Promise<void> {
		const isCurrentlyRead = notification.isRead;

		await handleAction({
			notification,
			actionName: isCurrentlyRead ? "markUnread" : "markRead",
			performAction: (key) =>
				isCurrentlyRead ? markNotificationUnread(key) : markNotificationRead(key),
			successToast: null,
			errorToast: "Failed to toggle read status",
			shouldCloseDetailInSingleMode: isCurrentlyRead,
		});
	}

	async function archive(notification: Notification): Promise<void> {
		const isCurrentlyArchived = notification.archived;
		const key = notification.githubId ?? notification.id;

		await handleAction({
			notification,
			actionName: isCurrentlyArchived ? "unarchive" : "archive",
			performAction: (k) =>
				isCurrentlyArchived ? unarchiveNotification(k) : archiveNotification(k),
			shouldRemoveFromSelection: true,
			successToast: isCurrentlyArchived ? "Unarchived" : "Archived",
			errorToast: "Failed to toggle archive status",
			undoConfig: key
				? {
						actionType: isCurrentlyArchived ? "unarchive" : "archive",
						performUndo: async () => {
							if (isCurrentlyArchived) {
								await archiveNotification(key);
							} else {
								await unarchiveNotification(key);
							}
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	async function mute(notification: Notification): Promise<void> {
		const key = notification.githubId ?? notification.id;

		await handleAction({
			notification,
			actionName: "mute",
			performAction: (k) => muteNotification(k),
			successToast: "Muted",
			errorToast: "Failed to mute notification",
			undoConfig: key
				? {
						actionType: "mute",
						performUndo: async () => {
							await unmuteNotification(key);
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	async function unmute(notification: Notification): Promise<void> {
		const key = notification.githubId ?? notification.id;

		await handleAction({
			notification,
			actionName: "unmute",
			performAction: (k) => unmuteNotification(k),
			successToast: "Unmuted",
			errorToast: "Failed to unmute notification",
			undoConfig: key
				? {
						actionType: "unmute",
						performUndo: async () => {
							await muteNotification(key);
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	async function snooze(notification: Notification, until: string): Promise<void> {
		const key = notification.githubId ?? notification.id;

		await handleAction({
			notification,
			actionName: "snooze",
			performAction: async (k) => {
				await snoozeNotification(k, until);
				// Snooze API doesn't return updated notification, so return original
				return notification;
			},
			successToast: "Snoozed",
			errorToast: "Failed to snooze notification",
			undoConfig: key
				? {
						actionType: "snooze",
						metadata: { snoozedUntil: until },
						performUndo: async () => {
							await unsnoozeNotification(key);
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	async function unsnooze(notification: Notification): Promise<void> {
		const key = notification.githubId ?? notification.id;
		// Store the current snoozedUntil value for potential redo
		const previousSnoozedUntil = notification.snoozedUntil;

		await handleAction({
			notification,
			actionName: "unsnooze",
			performAction: (k) => unsnoozeNotification(k),
			successToast: "Unsnoozed",
			errorToast: "Failed to unsnooze notification",
			undoConfig:
				key && previousSnoozedUntil
					? {
							actionType: "unsnooze",
							metadata: { snoozedUntil: previousSnoozedUntil },
							performUndo: async () => {
								await snoozeNotification(key, previousSnoozedUntil);
								await options.onRefresh?.();
								await options.onRefreshViewCounts?.();
							},
						}
					: undefined,
		});
	}

	async function star(notification: Notification): Promise<void> {
		const key = notification.githubId ?? notification.id;

		await handleAction({
			notification,
			actionName: "star",
			performAction: (k) => starNotification(k),
			successToast: "Starred",
			errorToast: "Failed to star notification",
			undoConfig: key
				? {
						actionType: "star",
						performUndo: async () => {
							await unstarNotification(key);
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	async function unstar(notification: Notification): Promise<void> {
		const key = notification.githubId ?? notification.id;

		await handleAction({
			notification,
			actionName: "unstar",
			performAction: (k) => unstarNotification(k),
			successToast: "Unstarred",
			errorToast: "Failed to unstar notification",
			undoConfig: key
				? {
						actionType: "unstar",
						performUndo: async () => {
							await starNotification(key);
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	async function unfilter(notification: Notification): Promise<void> {
		await handleAction({
			notification,
			actionName: "unfilter",
			performAction: (key) => unfilterNotification(key),
			successToast: "Moved to inbox",
			errorToast: "Failed to move notification to inbox",
		});
	}

	async function assignTag(githubId: string, tagIdOrName: string, tagName?: string): Promise<void> {
		// Find the notification to operate on
		const currentPageData = get(notificationStore.pageData);
		const currentNotification = currentPageData.items.find(
			(n) => (n.githubId && n.githubId === githubId) || n.id === githubId
		);

		if (!currentNotification) {
			toastStore.error("Notification not found");
			return;
		}

		// Check if it's a UUID (tag ID) or a name
		// UUID format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx where x is hexadecimal
		// We only support UUID tag IDs, so if it looks like a UUID, treat it as an ID
		const isUuidId = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(
			tagIdOrName
		);

		const key = currentNotification.githubId ?? currentNotification.id;

		await handleAction({
			notification: currentNotification,
			actionName: "assignTag",
			performAction: async (k) => {
				if (isUuidId) {
					// It's a UUID tag ID - use direct assignment
					return await assignTagToNotification(k, tagIdOrName);
				} else {
					// It's a tag name - use name-based assignment (creates tag if needed)
					return await import("$lib/api/tags").then((m) =>
						m.assignTagToNotificationByName(k, tagIdOrName)
					);
				}
			},
			successToast: `Added tag${tagName ? ` "${tagName}"` : ""}`,
			errorToast: "Failed to assign tag",
			undoConfig:
				key && isUuidId
					? {
							actionType: "assignTag",
							metadata: { tagId: tagIdOrName, tagName: tagName },
							performUndo: async () => {
								await removeTagFromNotification(key, tagIdOrName);
								await options.onRefresh?.();
								await options.onRefreshViewCounts?.();
							},
						}
					: undefined,
		});
	}

	async function removeTag(githubId: string, tagId: string, tagName?: string): Promise<void> {
		// Find the notification to operate on
		const currentPageData = get(notificationStore.pageData);
		const currentNotification = currentPageData.items.find(
			(n) => (n.githubId && n.githubId === githubId) || n.id === githubId
		);

		if (!currentNotification) {
			toastStore.error("Notification not found");
			return;
		}

		const key = currentNotification.githubId ?? currentNotification.id;

		await handleAction({
			notification: currentNotification,
			actionName: "unassignTag",
			performAction: (k) => removeTagFromNotification(k, tagId),
			successToast: `Removed tag${tagName ? ` "${tagName}"` : ""}`,
			errorToast: "Failed to remove tag",
			undoConfig: key
				? {
						actionType: "removeTag",
						metadata: { tagId, tagName },
						performUndo: async () => {
							await assignTagToNotification(key, tagId);
							await options.onRefresh?.();
							await options.onRefreshViewCounts?.();
						},
					}
				: undefined,
		});
	}

	return {
		markRead,
		archive,
		mute,
		unmute,
		snooze,
		unsnooze,
		star,
		unstar,
		unfilter,
		assignTag,
		removeTag,
	};
}
