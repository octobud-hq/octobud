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
import type { KeyboardShortcutActions } from "../interfaces/keyboardShortcutActions";
import type { NotificationActions } from "../interfaces/notificationActions";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { UIStore } from "../../stores/uiStateStore";

/**
 * Keyboard Shortcut Action Controller
 * Thin wrappers that get focused notification and call corresponding notification action
 */
export function createKeyboardShortcutActionController(
	keyboardStore: KeyboardStore,
	uiStore: UIStore,
	notificationActions: Pick<
		NotificationActions,
		| "markRead"
		| "archive"
		| "mute"
		| "unmute"
		| "snooze"
		| "unsnooze"
		| "star"
		| "unstar"
		| "unfilter"
	>
): KeyboardShortcutActions {
	function getFocusedNotification() {
		const focused = get(keyboardStore.focusedNotification);
		return focused;
	}

	function archiveFocusedNotification(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		void notificationActions.archive(focused);
		return true;
	}

	function markFocusedNotificationRead(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		void notificationActions.markRead(focused);
		return true;
	}

	function muteFocusedNotification(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		const isCurrentlyMuted = !!focused.muted;
		if (isCurrentlyMuted) {
			void notificationActions.unmute(focused);
		} else {
			void notificationActions.mute(focused);
		}
		return true;
	}

	function starFocusedNotification(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		const isCurrentlyStarred = !!focused.starred;
		if (isCurrentlyStarred) {
			void notificationActions.unstar(focused);
		} else {
			void notificationActions.star(focused);
		}
		return true;
	}

	function unfilterFocusedNotification(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		if (!focused.filtered) {
			return false;
		}
		void notificationActions.unfilter(focused);
		return true;
	}

	function snoozeFocusedNotification(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		const notificationId = focused.githubId ?? focused.id;
		const currentState = get(uiStore.snoozeDropdownState);

		// Toggle: if already open for this notification with keyboard-shortcut source, close it
		if (
			currentState?.notificationId === notificationId &&
			currentState?.source === "keyboard-shortcut"
		) {
			uiStore.closeSnoozeDropdown();
		} else {
			uiStore.openSnoozeDropdown(notificationId, "keyboard-shortcut");
		}
		return true;
	}

	function tagFocusedNotification(): boolean {
		const focused = getFocusedNotification();
		if (!focused) {
			return false;
		}
		const notificationId = focused.githubId ?? focused.id;
		const currentState = get(uiStore.tagDropdownState);

		// Toggle: if already open for this notification with keyboard-shortcut source, close it
		if (
			currentState?.notificationId === notificationId &&
			currentState?.source === "keyboard-shortcut"
		) {
			uiStore.closeTagDropdown();
		} else {
			uiStore.openTagDropdown(notificationId, "keyboard-shortcut");
		}
		return true;
	}

	return {
		archiveFocusedNotification,
		markFocusedNotificationRead,
		muteFocusedNotification,
		starFocusedNotification,
		snoozeFocusedNotification,
		tagFocusedNotification,
		unfilterFocusedNotification,
	};
}
