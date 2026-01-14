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
import { tick } from "svelte";
import type { Notification } from "$lib/api/types";
import type { BulkActions } from "../interfaces/bulkActions";
import {
	bulkMarkNotificationsRead,
	bulkMarkNotificationsUnread,
	bulkArchiveNotifications,
	bulkUnarchiveNotifications,
	bulkMuteNotifications,
	bulkUnmuteNotifications,
	bulkSnoozeNotifications,
	bulkUnsnoozeNotifications,
	bulkStarNotifications,
	bulkUnstarNotifications,
	bulkUnfilterNotifications,
	bulkAssignTag as apiBulkAssignTag,
	bulkRemoveTag as apiBulkRemoveTag,
} from "$lib/api/notifications";
import { toastStore } from "$lib/stores/toastStore";
import { undoStore } from "$lib/stores/undoStore";
import {
	createUndoableAction,
	toRecentActionNotification,
	type UndoableActionType,
} from "$lib/undo/types";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { SelectionStore } from "../../stores/selectionStore";
import type { QueryStore } from "../../stores/queryStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { ControllerOptions } from "../interfaces/common";
import type { SharedHelpers } from "./sharedHelpers";

interface StoreCollection {
	notificationStore: NotificationStore;
	paginationStore: PaginationStore;
	selectionStore: SelectionStore;
	queryStore: QueryStore;
	uiStore: UIStore;
}

/**
 * Bulk Action Controller
 * Manages bulk operations on notifications
 */
export function createBulkActionController(
	stores: StoreCollection,
	options: ControllerOptions,
	sharedHelpers: SharedHelpers
): BulkActions {
	const { notificationStore, paginationStore, selectionStore, queryStore, uiStore } = stores;

	async function handleBulkAction(params: {
		actionName:
			| "markRead"
			| "markUnread"
			| "archive"
			| "unarchive"
			| "mute"
			| "unmute"
			| "snooze"
			| "unsnooze"
			| "star"
			| "unstar"
			| "unfilter"
			| "assignTag"
			| "removeTag";
		performAction: (ids: string[], query?: string) => Promise<number>;
		successToast: (count: number) => string;
		errorToast: string;
		/** Configuration for making this action undoable (only for ID-based operations) */
		undoConfig?: {
			/** The action type for undo purposes */
			actionType: UndoableActionType;
			/** Function to perform the inverse operation */
			createPerformUndo: (ids: string[], notifications: Notification[]) => () => Promise<void>;
			/** Optional metadata (e.g., tagId, snoozedUntil) */
			metadata?: {
				tagId?: string;
				tagName?: string;
				snoozedUntil?: string | null;
			};
		};
	}): Promise<void> {
		const { actionName, performAction, successToast, errorToast, undoConfig } = params;

		// Determine operation mode and parameters
		const mode = get(selectionStore.selectAllMode);
		const currentSelectedIds = get(selectionStore.selectedIds);
		const currentQuery = get(queryStore.quickQuery);
		const total = get(notificationStore.pageData).total;

		const useQuery = mode === "all";
		const ids = useQuery ? [] : Array.from(currentSelectedIds);
		const count = useQuery ? total : ids.length;

		if (count === 0) return;

		// Check for confirmation if needed:
		// - Always confirm for query-based operations (they are NOT undoable)
		// - Confirm for ID-based operations exceeding the threshold
		const BULK_CONFIRMATION_THRESHOLD = 30;
		const needsConfirmation = useQuery || count > BULK_CONFIRMATION_THRESHOLD;
		if (needsConfirmation && options.requestBulkConfirmation) {
			const actionNameForConfirmation =
				actionName === "assignTag"
					? "assign tag"
					: actionName === "removeTag"
						? "remove tag"
						: actionName;
			const confirmed = await options.requestBulkConfirmation(
				actionNameForConfirmation,
				count,
				useQuery
			);
			if (!confirmed) return;
		}

		uiStore.bulkUpdating.set(true);

		// Capture affected notifications BEFORE the action (for undo history display)
		// Only do this for ID-based operations (not query-based)
		let affectedNotifications: Notification[] = [];
		if (!useQuery && undoConfig) {
			const pageData = get(notificationStore.pageData);
			affectedNotifications = pageData.items.filter((item) => {
				const key = item.githubId ?? item.id;
				return key && ids.includes(key);
			});
		}

		try {
			// Check if action will dismiss using first item's hints
			let willDismiss = false;
			const pageData = get(notificationStore.pageData);
			const items = pageData.items;
			const firstItem =
				!useQuery && ids.length > 0
					? items.find((item) => {
							const key = item.githubId ?? item.id;
							return key === ids[0];
						})
					: items[0];

			if (firstItem) {
				// Simply check if the action is in the dismissedOn list
				const dismissedOn = firstItem.actionHints?.dismissedOn ?? [];
				willDismiss = dismissedOn.includes(actionName);
			}

			// Perform the bulk action
			const actualCount = await performAction(ids, useQuery ? currentQuery : undefined);

			await options.onRefreshViewCounts?.();

			// Sync query to URL before refreshing to preserve it
			await sharedHelpers.syncQueryToUrl();

			// Reload the page
			paginationStore.setLoading(true);
			await options.onRefresh?.();
			paginationStore.setLoading(false);
			await tick();

			// Handle empty page navigation
			const refreshedPageData = get(notificationStore.pageData);
			const refreshedItems = refreshedPageData.items;
			const refreshedPage = get(paginationStore.page);

			if (refreshedItems.length === 0 && refreshedPage > 1) {
				const previousPage = refreshedPage - 1;
				paginationStore.setPage(previousPage);
				await sharedHelpers.syncQueryToUrl();
				await options.onRefresh?.();
			}

			// Clear selection only if action dismissed items
			if (willDismiss) {
				selectionStore.clearSelection();
			}

			// Show success toast with undo support for ID-based operations
			const toastMessage = successToast(actualCount || count);

			if (!useQuery && undoConfig && affectedNotifications.length > 0) {
				// Create undoable action and push to undo store (shows toast with undo button)
				// Use toRecentActionNotification to store only essential fields for lighter storage
				const undoableAction = createUndoableAction({
					type: undoConfig.actionType,
					notifications: affectedNotifications.map(toRecentActionNotification),
					metadata: undoConfig.metadata,
					performUndo: undoConfig.createPerformUndo(ids, affectedNotifications),
				});
				// Use setTimeout to ensure the UI has updated before showing the toast.
				// This prevents visual jank from the toast appearing before the list refreshes.
				setTimeout(() => {
					undoStore.pushAction(undoableAction);
				}, 0);
			} else {
				// Query-based operations or no undo config - show regular toast
				// setTimeout used for consistency with undo flow
				setTimeout(() => {
					toastStore.success(toastMessage);
				}, 0);
			}
		} catch (err) {
			console.error(errorToast, err);
			toastStore.error(errorToast);
		} finally {
			uiStore.bulkUpdating.set(false);
		}
	}

	async function bulkMarkRead(): Promise<void> {
		await handleBulkAction({
			actionName: "markRead",
			performAction: (ids, query) => bulkMarkNotificationsRead(ids, query),
			successToast: (count) => `Marked ${count} notification${count === 1 ? "" : "s"} as read`,
			errorToast: "Failed to mark notifications as read",
			undoConfig: {
				actionType: "markRead",
				createPerformUndo: (ids) => async () => {
					await bulkMarkNotificationsUnread(ids);
				},
			},
		});
	}

	async function bulkMarkUnread(): Promise<void> {
		await handleBulkAction({
			actionName: "markUnread",
			performAction: (ids, query) => bulkMarkNotificationsUnread(ids, query),
			successToast: (count) => `Marked ${count} notification${count === 1 ? "" : "s"} as unread`,
			errorToast: "Failed to mark notifications as unread",
			undoConfig: {
				actionType: "markUnread",
				createPerformUndo: (ids) => async () => {
					await bulkMarkNotificationsRead(ids);
				},
			},
		});
	}

	async function bulkArchive(): Promise<void> {
		await handleBulkAction({
			actionName: "archive",
			performAction: (ids, query) => bulkArchiveNotifications(ids, query),
			successToast: (count) => `Archived ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to archive notifications",
			undoConfig: {
				actionType: "archive",
				createPerformUndo: (ids) => async () => {
					await bulkUnarchiveNotifications(ids);
				},
			},
		});
	}

	async function bulkUnarchive(): Promise<void> {
		await handleBulkAction({
			actionName: "unarchive",
			performAction: (ids, query) => bulkUnarchiveNotifications(ids, query),
			successToast: (count) => `Unarchived ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to unarchive notifications",
			undoConfig: {
				actionType: "unarchive",
				createPerformUndo: (ids) => async () => {
					await bulkArchiveNotifications(ids);
				},
			},
		});
	}

	async function bulkMute(): Promise<void> {
		await handleBulkAction({
			actionName: "mute",
			performAction: (ids, query) => bulkMuteNotifications(ids, query),
			successToast: (count) => `Muted ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to mute notifications",
			undoConfig: {
				actionType: "mute",
				createPerformUndo: (ids) => async () => {
					await bulkUnmuteNotifications(ids);
				},
			},
		});
	}

	async function bulkUnmute(): Promise<void> {
		await handleBulkAction({
			actionName: "unmute",
			performAction: (ids, query) => bulkUnmuteNotifications(ids, query),
			successToast: (count) => `Unmuted ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to unmute notifications",
			undoConfig: {
				actionType: "unmute",
				createPerformUndo: (ids) => async () => {
					await bulkMuteNotifications(ids);
				},
			},
		});
	}

	async function bulkSnooze(snoozedUntil: string): Promise<void> {
		await handleBulkAction({
			actionName: "snooze",
			performAction: (ids, query) => bulkSnoozeNotifications(ids, snoozedUntil, query),
			successToast: (count) => `Snoozed ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to snooze notifications",
			undoConfig: {
				actionType: "snooze",
				metadata: { snoozedUntil },
				createPerformUndo: (ids) => async () => {
					await bulkUnsnoozeNotifications(ids);
				},
			},
		});
	}

	async function bulkUnsnooze(): Promise<void> {
		await handleBulkAction({
			actionName: "unsnooze",
			performAction: (ids, query) => bulkUnsnoozeNotifications(ids, query),
			successToast: (count) => `Unsnoozed ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to unsnooze notifications",
			// Note: unsnooze can't be easily undone as we don't know the original snoozedUntil values
			// So we don't provide undoConfig for bulk unsnooze
		});
	}

	async function bulkStar(): Promise<void> {
		await handleBulkAction({
			actionName: "star",
			performAction: (ids, query) => bulkStarNotifications(ids, query),
			successToast: (count) => `Starred ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to star notifications",
			undoConfig: {
				actionType: "star",
				createPerformUndo: (ids) => async () => {
					await bulkUnstarNotifications(ids);
				},
			},
		});
	}

	async function bulkUnstar(): Promise<void> {
		await handleBulkAction({
			actionName: "unstar",
			performAction: (ids, query) => bulkUnstarNotifications(ids, query),
			successToast: (count) => `Unstarred ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to unstar notifications",
			undoConfig: {
				actionType: "unstar",
				createPerformUndo: (ids) => async () => {
					await bulkStarNotifications(ids);
				},
			},
		});
	}

	async function bulkUnfilter(): Promise<void> {
		await handleBulkAction({
			actionName: "unfilter",
			performAction: (ids, query) => bulkUnfilterNotifications(ids, query),
			successToast: (count) => `Moved ${count} notification${count === 1 ? "" : "s"} to inbox`,
			errorToast: "Failed to move notifications to inbox",
			// Note: unfilter can't be easily undone as we don't know the original filter state
			// So we don't provide undoConfig for bulk unfilter
		});
	}

	async function bulkAssignTag(tagId: string, tagName?: string): Promise<void> {
		await handleBulkAction({
			actionName: "assignTag",
			performAction: (ids, query) => apiBulkAssignTag(ids, tagId, query),
			successToast: (count) => `Applied tag to ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to apply tag to notifications",
			undoConfig: {
				actionType: "assignTag",
				metadata: { tagId, tagName },
				createPerformUndo: (ids) => async () => {
					await apiBulkRemoveTag(ids, tagId);
				},
			},
		});
	}

	async function bulkRemoveTag(tagId: string, tagName?: string): Promise<void> {
		await handleBulkAction({
			actionName: "removeTag",
			performAction: (ids, query) => apiBulkRemoveTag(ids, tagId, query),
			successToast: (count) => `Removed tag from ${count} notification${count === 1 ? "" : "s"}`,
			errorToast: "Failed to remove tag from notifications",
			undoConfig: {
				actionType: "removeTag",
				metadata: { tagId, tagName },
				createPerformUndo: (ids) => async () => {
					await apiBulkAssignTag(ids, tagId);
				},
			},
		});
	}

	async function markAllRead(): Promise<void> {
		selectionStore.selectAllAcrossPages();
		await bulkMarkRead();
	}

	async function markAllUnread(): Promise<void> {
		selectionStore.selectAllAcrossPages();
		await bulkMarkUnread();
	}

	async function markAllArchive(): Promise<void> {
		selectionStore.selectAllAcrossPages();
		await bulkArchive();
	}

	return {
		bulkMarkRead,
		bulkMarkUnread,
		bulkArchive,
		bulkUnarchive,
		bulkMute,
		bulkUnmute,
		bulkSnooze,
		bulkUnsnooze,
		bulkStar,
		bulkUnstar,
		bulkUnfilter,
		bulkAssignTag,
		bulkRemoveTag,
		markAllRead,
		markAllUnread,
		markAllArchive,
	};
}
