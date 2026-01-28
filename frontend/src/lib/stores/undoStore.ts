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

import { writable, derived, get } from "svelte/store";
import type {
	UndoableAction,
	UndoableActionMetadata,
	UndoableActionType,
	RecentActionNotification,
} from "$lib/undo/types";
import { getInverseActionType } from "$lib/undo/types";
import { toastStore } from "./toastStore";
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
	bulkAssignTag,
	bulkRemoveTag,
} from "$lib/api/notifications";
import { assignTagToNotification, removeTagFromNotification } from "$lib/api/tags";

const MAX_HISTORY_SIZE = 20;
const UNDO_TOAST_DURATION = 7000; // 7 seconds for undo toasts
const STORAGE_KEY = "octobud_undo_history";

/**
 * Serializable version of UndoableAction for localStorage
 */
interface SerializedUndoableAction {
	id: string;
	type: UndoableActionType;
	description: string;
	notifications: RecentActionNotification[];
	timestamp: number;
	metadata?: UndoableActionMetadata;
	/** Notification keys for reconstructing undo */
	notificationKeys: string[];
}

interface UndoStoreState {
	/** History of actions (most recent first) */
	history: UndoableAction[];
	/** Currently active toast action (the one whose toast is visible) */
	activeToastAction: UndoableAction | null;
	/** Whether an undo operation is currently in progress */
	isUndoing: boolean;
}

/**
 * Execute the inverse operation for an action type
 * Uses INVERSE_ACTION_MAP to determine what operation to perform
 */
async function executeInverseOperation(
	type: UndoableActionType,
	notificationKeys: string[],
	metadata?: UndoableActionMetadata
): Promise<void> {
	const inverseType = getInverseActionType(type);
	const isBulk = notificationKeys.length > 1;
	const key = notificationKeys[0];

	// Handle operations that require metadata
	if (inverseType === "snooze") {
		if (!metadata?.snoozedUntil) {
			throw new Error("Cannot undo unsnooze: original snooze time not available");
		}
		if (isBulk) {
			await bulkSnoozeNotifications(notificationKeys, metadata.snoozedUntil);
		} else {
			await snoozeNotification(key, metadata.snoozedUntil);
		}
		return;
	}

	if (inverseType === "assignTag") {
		if (!metadata?.tagId) {
			throw new Error("Cannot undo tag removal: tag ID not available");
		}
		if (isBulk) {
			await bulkAssignTag(notificationKeys, metadata.tagId);
		} else {
			await assignTagToNotification(key, metadata.tagId);
		}
		return;
	}

	if (inverseType === "removeTag") {
		if (!metadata?.tagId) {
			throw new Error("Cannot undo tag assignment: tag ID not available");
		}
		if (isBulk) {
			await bulkRemoveTag(notificationKeys, metadata.tagId);
		} else {
			await removeTagFromNotification(key, metadata.tagId);
		}
		return;
	}

	// Handle simple inverse operations (no metadata required)
	if (isBulk) {
		switch (inverseType) {
			case "unarchive":
				await bulkUnarchiveNotifications(notificationKeys);
				break;
			case "archive":
				await bulkArchiveNotifications(notificationKeys);
				break;
			case "unmute":
				await bulkUnmuteNotifications(notificationKeys);
				break;
			case "mute":
				await bulkMuteNotifications(notificationKeys);
				break;
			case "unstar":
				await bulkUnstarNotifications(notificationKeys);
				break;
			case "star":
				await bulkStarNotifications(notificationKeys);
				break;
			case "markUnread":
				await bulkMarkNotificationsUnread(notificationKeys);
				break;
			case "markRead":
				await bulkMarkNotificationsRead(notificationKeys);
				break;
			case "unsnooze":
				await bulkUnsnoozeNotifications(notificationKeys);
				break;
			default:
				throw new Error(`Unknown inverse action type: ${inverseType}`);
		}
	} else {
		switch (inverseType) {
			case "unarchive":
				await unarchiveNotification(key);
				break;
			case "archive":
				await archiveNotification(key);
				break;
			case "unmute":
				await unmuteNotification(key);
				break;
			case "mute":
				await muteNotification(key);
				break;
			case "unstar":
				await unstarNotification(key);
				break;
			case "star":
				await starNotification(key);
				break;
			case "markUnread":
				await markNotificationUnread(key);
				break;
			case "markRead":
				await markNotificationRead(key);
				break;
			case "unsnooze":
				await unsnoozeNotification(key);
				break;
			default:
				throw new Error(`Unknown inverse action type: ${inverseType}`);
		}
	}
}

/**
 * Create the performUndo function for an action based on its type and metadata.
 * Note: Refresh is handled centrally by the undo store after performUndo completes,
 * so we don't need to call it here.
 */
function createPerformUndo(
	type: UndoableActionType,
	notificationKeys: string[],
	metadata?: UndoableActionMetadata
): () => Promise<void> {
	return async () => {
		await executeInverseOperation(type, notificationKeys, metadata);
	};
}

/**
 * Serialize an UndoableAction for storage
 */
function serializeAction(action: UndoableAction): SerializedUndoableAction {
	return {
		id: action.id,
		type: action.type,
		description: action.description,
		notifications: action.notifications,
		timestamp: action.timestamp,
		metadata: action.metadata,
		notificationKeys: action.notifications.map((n) => n.githubId ?? n.id),
	};
}

/**
 * Deserialize a stored action back to UndoableAction
 */
function deserializeAction(serialized: SerializedUndoableAction): UndoableAction {
	return {
		id: serialized.id,
		type: serialized.type,
		description: serialized.description,
		notifications: serialized.notifications,
		timestamp: serialized.timestamp,
		metadata: serialized.metadata,
		performUndo: createPerformUndo(
			serialized.type,
			serialized.notificationKeys,
			serialized.metadata
		),
	};
}

/**
 * Load history from localStorage
 */
function loadFromStorage(): UndoableAction[] {
	if (typeof localStorage === "undefined") return [];

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (!stored) return [];

		const serialized: SerializedUndoableAction[] = JSON.parse(stored);
		return serialized.map((s) => deserializeAction(s));
	} catch (error) {
		console.error("Failed to load undo history from storage:", error);
		return [];
	}
}

/**
 * Save history to localStorage
 */
function saveToStorage(history: UndoableAction[]): void {
	if (typeof localStorage === "undefined") return;

	try {
		const serialized = history.map(serializeAction);
		localStorage.setItem(STORAGE_KEY, JSON.stringify(serialized));
	} catch (error) {
		console.error("Failed to save undo history to storage:", error);
	}
}

function createUndoStore() {
	// We'll set up refresh callback later via setRefreshCallback
	let refreshCallback: (() => Promise<void>) | undefined;

	const { subscribe, update, set } = writable<UndoStoreState>({
		history: [],
		activeToastAction: null,
		isUndoing: false,
	});

	let toastTimeoutId: ReturnType<typeof setTimeout> | null = null;
	let activeToastId: string | null = null;
	let initialized = false;

	/**
	 * Initialize the store by loading from localStorage
	 */
	function initialize() {
		if (initialized) return;
		initialized = true;

		const history = loadFromStorage();
		if (history.length > 0) {
			update((state) => ({
				...state,
				history,
			}));
		}
	}

	/**
	 * Set the refresh callback for undo operations.
	 * This is called centrally after any undo operation completes,
	 * so we don't need to inject it into individual performUndo functions.
	 */
	function setRefreshCallback(callback: () => Promise<void>) {
		refreshCallback = callback;
	}

	/**
	 * Clear the current toast timeout
	 */
	function clearToastTimeout() {
		if (toastTimeoutId !== null) {
			clearTimeout(toastTimeoutId);
			toastTimeoutId = null;
		}
	}

	/**
	 * Get notification keys from an action for comparison
	 */
	function getNotificationKeys(action: UndoableAction): string[] {
		return action.notifications.map((n) => n.githubId ?? n.id).sort();
	}

	/**
	 * Check if two actions have the same notification set
	 */
	function haveSameNotifications(action1: UndoableAction, action2: UndoableAction): boolean {
		const keys1 = getNotificationKeys(action1);
		const keys2 = getNotificationKeys(action2);
		if (keys1.length !== keys2.length) return false;
		return keys1.every((key, i) => key === keys2[i]);
	}

	/**
	 * Check if an action is the inverse of another (for cancellation)
	 */
	function isInverseAction(newAction: UndoableAction, existingAction: UndoableAction): boolean {
		const inverseType = getInverseActionType(newAction.type);
		return existingAction.type === inverseType && haveSameNotifications(newAction, existingAction);
	}

	/**
	 * Clear the active toast action when toast expires.
	 * The action is already in history (saved immediately on push), so we just
	 * need to clear the active reference.
	 */
	function clearActiveToast() {
		update((state) => ({
			...state,
			activeToastAction: null,
		}));
		activeToastId = null;
	}

	/**
	 * Push a new undoable action. Shows an undo toast and sets it as active.
	 *
	 * Actions are saved to history immediately (not when toast expires), ensuring
	 * they persist across page refreshes even while the toast is visible.
	 *
	 * Implements inverse cancellation: if the inverse action exists in history,
	 * it's removed (e.g., Star then Unstar cancels out).
	 */
	function pushAction(action: UndoableAction) {
		const currentState = get({ subscribe });

		// If there's already an active toast, dismiss it first
		if (currentState.activeToastAction) {
			if (activeToastId) {
				toastStore.dismiss(activeToastId);
			}
			clearToastTimeout();
		}

		// Remove any inverse actions from history (inverse cancellation)
		// This handles both the previous activeToastAction (which is in history) and older items
		update((state) => {
			const newHistory = state.history.filter((h) => !isInverseAction(action, h));
			if (newHistory.length !== state.history.length) {
				saveToStorage(newHistory);
			}
			return { ...state, history: newHistory, activeToastAction: null };
		});

		// Add the new action to history immediately and set as active toast
		// This ensures the action persists even if the page is refreshed while toast is visible
		update((state) => {
			const newHistory = [action, ...state.history].slice(0, MAX_HISTORY_SIZE);
			saveToStorage(newHistory);
			return {
				...state,
				history: newHistory,
				activeToastAction: action,
			};
		});

		// Show toast with undo support
		activeToastId = toastStore.showWithUndo(
			action.description,
			"success",
			UNDO_TOAST_DURATION,
			() => {
				void undoActiveToast();
			}
		);

		// Set timeout to clear active toast reference when toast expires
		// (action remains in history)
		toastTimeoutId = setTimeout(() => {
			clearActiveToast();
		}, UNDO_TOAST_DURATION);
	}

	/**
	 * Undo the currently active toast action
	 */
	async function undoActiveToast(): Promise<boolean> {
		const state = get({ subscribe });
		if (!state.activeToastAction || state.isUndoing) {
			return false;
		}

		const action = state.activeToastAction;
		const currentToastId = activeToastId;

		// Clear the timeout but keep the toast visible for morphing
		clearToastTimeout();

		// Mark as undoing, clear active action, and remove from history
		// (action is now stored in history immediately on push)
		update((s) => {
			const newHistory = s.history.filter((a) => a.id !== action.id);
			saveToStorage(newHistory);
			return {
				...s,
				history: newHistory,
				activeToastAction: null,
				isUndoing: true,
			};
		});

		try {
			await action.performUndo();

			// Centralized refresh after undo - this ensures UI is updated regardless of
			// whether the action was freshly created or deserialized from localStorage
			await refreshCallback?.();

			// Morph the toast to show "Undone" state
			if (currentToastId) {
				toastStore.morphToUndone(currentToastId);
			}
			activeToastId = null;
			return true;
		} catch (error) {
			console.error("Failed to undo action:", error);
			// Dismiss the toast on error
			if (currentToastId) {
				toastStore.dismiss(currentToastId);
				activeToastId = null;
			}
			handleUndoError(error, action);
			return false;
		} finally {
			update((s) => ({
				...s,
				isUndoing: false,
			}));
		}
	}

	/**
	 * Undo a specific action from history by ID
	 */
	async function undoFromHistory(actionId: string): Promise<boolean> {
		const state = get({ subscribe });
		const action = state.history.find((a) => a.id === actionId);

		if (!action || state.isUndoing) {
			return false;
		}

		// Mark as undoing
		update((s) => ({
			...s,
			isUndoing: true,
		}));

		try {
			await action.performUndo();

			// Centralized refresh after undo - this ensures UI is updated regardless of
			// whether the action was freshly created or deserialized from localStorage
			await refreshCallback?.();

			// Remove from history on success
			update((s) => {
				const newHistory = s.history.filter((a) => a.id !== actionId);
				saveToStorage(newHistory);
				return {
					...s,
					history: newHistory,
				};
			});

			// Show a brief success toast
			toastStore.success("Undone", 1500);
			return true;
		} catch (error) {
			console.error("Failed to undo action:", error);
			handleUndoError(error, action);
			return false;
		} finally {
			update((s) => ({
				...s,
				isUndoing: false,
			}));
		}
	}

	/**
	 * Handle errors during undo operations
	 */
	function handleUndoError(error: unknown, action: UndoableAction) {
		const errorMessage = error instanceof Error ? error.message : "Unknown error";

		// Check for specific error types
		if (errorMessage.includes("404") || errorMessage.includes("not found")) {
			if (action.type === "assignTag" || action.type === "removeTag") {
				toastStore.error("Cannot undo: Tag no longer exists");
			} else {
				toastStore.error("Cannot undo: Notification no longer exists");
			}
			// Remove from history since it can't be undone
			update((s) => {
				const newHistory = s.history.filter((a) => a.id !== action.id);
				saveToStorage(newHistory);
				return {
					...s,
					history: newHistory,
				};
			});
		} else if (errorMessage.includes("403") || errorMessage.includes("permission")) {
			toastStore.error("Cannot undo: You no longer have access");
			// Remove from history
			update((s) => {
				const newHistory = s.history.filter((a) => a.id !== action.id);
				saveToStorage(newHistory);
				return {
					...s,
					history: newHistory,
				};
			});
		} else {
			// Network or other error - keep in history for retry
			toastStore.error("Cannot undo: Unable to connect to server");
		}
	}

	/**
	 * Remove an action from history by ID
	 */
	function removeFromHistory(actionId: string) {
		update((state) => {
			const newHistory = state.history.filter((a) => a.id !== actionId);
			saveToStorage(newHistory);
			return {
				...state,
				history: newHistory,
			};
		});
	}

	/**
	 * Clear all history
	 */
	function clearHistory() {
		update((state) => ({
			...state,
			history: [],
		}));
		saveToStorage([]);
	}

	/**
	 * Reset the entire store state (for testing purposes)
	 */
	function _reset() {
		clearToastTimeout();
		if (activeToastId) {
			toastStore.dismiss(activeToastId);
			activeToastId = null;
		}
		set({
			history: [],
			activeToastAction: null,
			isUndoing: false,
		});
		saveToStorage([]);
		// Reset initialized flag so initialize() can be called again
		initialized = false;
	}

	/**
	 * Check if undo via keyboard shortcut is available (only when toast is visible)
	 */
	function canUndoViaKeyboard(): boolean {
		const state = get({ subscribe });
		return state.activeToastAction !== null && !state.isUndoing;
	}

	// Derived stores
	const history = derived({ subscribe }, ($state) => $state.history);
	const activeToastAction = derived({ subscribe }, ($state) => $state.activeToastAction);
	const isUndoing = derived({ subscribe }, ($state) => $state.isUndoing);
	const historyCount = derived({ subscribe }, ($state) => $state.history.length);
	const hasHistory = derived({ subscribe }, ($state) => $state.history.length > 0);

	return {
		subscribe,
		// Actions
		initialize,
		setRefreshCallback,
		pushAction,
		undoActiveToast,
		undoFromHistory,
		removeFromHistory,
		clearHistory,
		canUndoViaKeyboard,
		// Derived stores
		history,
		activeToastAction,
		isUndoing,
		historyCount,
		hasHistory,
		// Testing only
		_reset,
	};
}

export const undoStore = createUndoStore();
