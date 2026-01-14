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
 * All undoable action types for notifications
 */
export type UndoableActionType =
	| "markRead"
	| "markUnread"
	| "archive"
	| "unarchive"
	| "mute"
	| "unmute"
	| "star"
	| "unstar"
	| "snooze"
	| "unsnooze"
	| "assignTag"
	| "removeTag";

/**
 * Mapping of action types to their inverse operations
 */
export const INVERSE_ACTION_MAP: Record<UndoableActionType, UndoableActionType> = {
	markRead: "markUnread",
	markUnread: "markRead",
	archive: "unarchive",
	unarchive: "archive",
	mute: "unmute",
	unmute: "mute",
	star: "unstar",
	unstar: "star",
	snooze: "unsnooze",
	unsnooze: "snooze",
	assignTag: "removeTag",
	removeTag: "assignTag",
};

/**
 * Get the inverse action type for undo operations
 */
export function getInverseActionType(type: UndoableActionType): UndoableActionType {
	return INVERSE_ACTION_MAP[type];
}

/**
 * Lightweight notification data for storage in undo history.
 * Contains only fields needed for display and undo operations.
 */
export interface RecentActionNotification {
	/** Primary key for undo operations */
	id: string;
	/** GitHub notification ID (preferred for API calls) */
	githubId?: string;
	/** Notification title for display */
	subjectTitle: string;
	/** Repository name for display */
	repoFullName: string;
	/** Type for icon display */
	subjectType: string;
	/** Raw subject data for icon determination */
	subjectRaw?: string;
	/** Read status for icon styling */
	isRead: boolean;
	/** State for icon styling (open, closed, merged) */
	subjectState?: string | null;
	/** Whether PR was merged */
	subjectMerged?: boolean | null;
	/** Reason for notification */
	reason?: string;
}

/**
 * Convert a full Notification to a lightweight RecentActionNotification
 */
export function toRecentActionNotification(notification: Notification): RecentActionNotification {
	return {
		id: notification.id,
		githubId: notification.githubId,
		subjectTitle: notification.subjectTitle,
		repoFullName: notification.repoFullName,
		subjectType: notification.subjectType,
		// subjectRaw is typed as unknown in Notification, cast to string for storage
		subjectRaw: notification.subjectRaw as string | undefined,
		isRead: notification.isRead,
		subjectState: notification.subjectState,
		subjectMerged: notification.subjectMerged,
		reason: notification.reason,
	};
}

/**
 * Human-readable descriptions for action types
 */
export const ACTION_DESCRIPTIONS: Record<UndoableActionType, string> = {
	markRead: "Marked as read",
	markUnread: "Marked as unread",
	archive: "Archived",
	unarchive: "Unarchived",
	mute: "Muted",
	unmute: "Unmuted",
	star: "Starred",
	unstar: "Unstarred",
	snooze: "Snoozed",
	unsnooze: "Unsnoozed",
	assignTag: "Added tag",
	removeTag: "Removed tag",
};

/**
 * Optional metadata for actions that require additional info to undo
 */
export interface UndoableActionMetadata {
	tagId?: string;
	tagName?: string;
	snoozedUntil?: string | null;
}

/**
 * An undoable action that can be stored in the history stack
 */
export interface UndoableAction {
	/** Unique identifier for this action */
	id: string;
	/** The type of action performed */
	type: UndoableActionType;
	/** Human-readable description (e.g., "Archived", "Starred 3 notifications") */
	description: string;
	/** Lightweight notification data for display and undo operations */
	notifications: RecentActionNotification[];
	/** Timestamp when the action was performed */
	timestamp: number;
	/** Additional metadata needed for undo (tagId, snoozedUntil, etc.) */
	metadata?: UndoableActionMetadata;
	/** Function that executes the inverse operation */
	performUndo: () => Promise<void>;
}

/**
 * Parameters for creating an undoable action
 */
export interface CreateUndoableActionParams {
	type: UndoableActionType;
	notifications: RecentActionNotification[];
	metadata?: UndoableActionMetadata;
	performUndo: () => Promise<void>;
}

/**
 * Generate a unique ID for an undoable action
 */
export function generateUndoActionId(): string {
	return `undo-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Create a human-readable description for an action
 */
export function createActionDescription(
	type: UndoableActionType,
	notificationCount: number,
	metadata?: UndoableActionMetadata
): string {
	const baseDescription = ACTION_DESCRIPTIONS[type];

	if (type === "assignTag" && metadata?.tagName) {
		return notificationCount === 1
			? `Added tag "${metadata.tagName}"`
			: `Added tag "${metadata.tagName}" to ${notificationCount} notifications`;
	}

	if (type === "removeTag" && metadata?.tagName) {
		return notificationCount === 1
			? `Removed tag "${metadata.tagName}"`
			: `Removed tag "${metadata.tagName}" from ${notificationCount} notifications`;
	}

	if (notificationCount === 1) {
		return baseDescription;
	}

	return `${baseDescription} ${notificationCount} notifications`;
}

/**
 * Create an UndoableAction from parameters
 */
export function createUndoableAction(params: CreateUndoableActionParams): UndoableAction {
	const { type, notifications, metadata, performUndo } = params;

	return {
		id: generateUndoActionId(),
		type,
		description: createActionDescription(type, notifications.length, metadata),
		notifications,
		timestamp: Date.now(),
		metadata,
		performUndo,
	};
}
