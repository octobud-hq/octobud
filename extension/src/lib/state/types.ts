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

import type { Readable, Writable } from "svelte/store";
import type { Notification, NotificationView, Tag } from "$lib/api/types";

/**
 * The panel's counterpart to frontend/src/lib/state/types.ts.
 *
 * `NotificationRow` reads its controller from context under the same key and
 * calls the same action names, so the ported row stays close to the desktop
 * original. The desktop interface is much larger; everything the panel has no
 * surface for — multiselect and bulk mode, split mode, inline detail, keyboard
 * focus indices, undo history — is deliberately absent rather than stubbed, so
 * it is obvious what the panel does and does not do.
 */

export const PANEL_CONTROLLER_KEY = "notificationPageController";

/** Where a dropdown was opened from. Only rows can open one in the panel. */
export type DropdownSource = "notification-row";

export interface DropdownState {
	notificationId: string;
	source: DropdownSource;
}

/** Distinguishes "Octobud isn't running" from "Octobud is running but has no GitHub token". */
export type PanelErrorKind = "unreachable" | "not_connected" | "request";

export interface PanelError {
	kind: PanelErrorKind;
	message: string;
}

export interface PanelControllerStores {
	views: Readable<NotificationView[]>;
	tags: Readable<Tag[]>;
	selectedSlug: Readable<string>;
	items: Readable<Notification[]>;
	total: Readable<number>;
	page: Readable<number>;
	totalPages: Readable<number>;
	loading: Readable<boolean>;
	error: Readable<PanelError | null>;
	snoozeDropdownState: Writable<DropdownState | null>;
	tagDropdownState: Writable<DropdownState | null>;
	customSnoozeNotificationId: Writable<string | null>;
}

export interface PanelControllerActions {
	/** Fetches views, tags and the selected view's first page. */
	initialize(): Promise<void>;
	refresh(): Promise<void>;
	selectView(slug: string): Promise<void>;
	goToPage(page: number): Promise<void>;

	/** Opens the notification's GitHub page in a new tab and marks it read. */
	openInGitHub(notification: Notification): Promise<void>;

	markRead(notification: Notification): Promise<void>;
	archive(notification: Notification): Promise<void>;
	mute(notification: Notification): Promise<void>;
	unmute(notification: Notification): Promise<void>;
	star(notification: Notification): Promise<void>;
	unstar(notification: Notification): Promise<void>;
	unfilter(notification: Notification): Promise<void>;
	snooze(notification: Notification, until: string): Promise<void>;
	unsnooze(notification: Notification): Promise<void>;
	assignTag(githubId: string, tagId: string): Promise<void>;
	removeTag(githubId: string, tagId: string): Promise<void>;

	openSnoozeDropdown(notificationId: string, source: DropdownSource): void;
	closeSnoozeDropdown(): void;
	openTagDropdown(notificationId: string, source: DropdownSource): void;
	closeTagDropdown(): void;
	openCustomSnoozeDialog(notificationId: string): void;
	closeCustomSnoozeDialog(): void;
}

export interface NotificationPageController {
	stores: PanelControllerStores;
	actions: PanelControllerActions;
}
