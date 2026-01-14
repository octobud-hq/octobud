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

import { describe, it, expect } from "vitest";
import {
	INVERSE_ACTION_MAP,
	getInverseActionType,
	toRecentActionNotification,
	createActionDescription,
	generateUndoActionId,
	createUndoableAction,
	type UndoableActionType,
} from "./types";
import type { Notification } from "$lib/api/types";

describe("Undo Types", () => {
	describe("INVERSE_ACTION_MAP", () => {
		it("maps each action to its inverse", () => {
			expect(INVERSE_ACTION_MAP.archive).toBe("unarchive");
			expect(INVERSE_ACTION_MAP.unarchive).toBe("archive");
			expect(INVERSE_ACTION_MAP.mute).toBe("unmute");
			expect(INVERSE_ACTION_MAP.unmute).toBe("mute");
			expect(INVERSE_ACTION_MAP.star).toBe("unstar");
			expect(INVERSE_ACTION_MAP.unstar).toBe("star");
			expect(INVERSE_ACTION_MAP.markRead).toBe("markUnread");
			expect(INVERSE_ACTION_MAP.markUnread).toBe("markRead");
			expect(INVERSE_ACTION_MAP.snooze).toBe("unsnooze");
			expect(INVERSE_ACTION_MAP.unsnooze).toBe("snooze");
			expect(INVERSE_ACTION_MAP.assignTag).toBe("removeTag");
			expect(INVERSE_ACTION_MAP.removeTag).toBe("assignTag");
		});

		it("is symmetric (inverse of inverse equals original)", () => {
			const actionTypes: UndoableActionType[] = [
				"archive",
				"unarchive",
				"mute",
				"unmute",
				"star",
				"unstar",
				"markRead",
				"markUnread",
				"snooze",
				"unsnooze",
				"assignTag",
				"removeTag",
			];

			for (const action of actionTypes) {
				const inverse = INVERSE_ACTION_MAP[action];
				const doubleInverse = INVERSE_ACTION_MAP[inverse];
				expect(doubleInverse).toBe(action);
			}
		});
	});

	describe("getInverseActionType", () => {
		it("returns the inverse action type", () => {
			expect(getInverseActionType("archive")).toBe("unarchive");
			expect(getInverseActionType("markRead")).toBe("markUnread");
			expect(getInverseActionType("assignTag")).toBe("removeTag");
		});
	});

	describe("toRecentActionNotification", () => {
		const createMockNotification = (overrides: Partial<Notification> = {}): Notification => ({
			id: "1",
			githubId: "gh-123",
			repoFullName: "owner/repo",
			reason: "review_requested",
			subjectTitle: "Test PR Title",
			subjectType: "PullRequest",
			subjectUrl: "https://github.com/owner/repo/pull/1",
			updatedAt: "2024-01-15T10:00:00Z",
			isRead: false,
			archived: false,
			muted: false,
			starred: false,
			filtered: false,
			snoozedUntil: null,
			labels: [],
			viewIds: [],
			tags: [],
			subjectState: "open",
			subjectMerged: false,
			subjectRaw: "PullRequest",
			...overrides,
		});

		it("extracts only essential fields from Notification", () => {
			const notification = createMockNotification();
			const recent = toRecentActionNotification(notification);

			expect(recent.id).toBe("1");
			expect(recent.githubId).toBe("gh-123");
			expect(recent.subjectTitle).toBe("Test PR Title");
			expect(recent.repoFullName).toBe("owner/repo");
			expect(recent.subjectType).toBe("PullRequest");
			expect(recent.isRead).toBe(false);
			expect(recent.subjectState).toBe("open");
			expect(recent.subjectMerged).toBe(false);
		});

		it("excludes non-essential fields", () => {
			const notification = createMockNotification({
				archived: true,
				muted: true,
				starred: true,
				labels: ["bug", "enhancement"],
				viewIds: ["view-1", "view-2"],
			});
			const recent = toRecentActionNotification(notification);

			// These fields should not exist on RecentActionNotification
			expect((recent as any).archived).toBeUndefined();
			expect((recent as any).muted).toBeUndefined();
			expect((recent as any).starred).toBeUndefined();
			expect((recent as any).labels).toBeUndefined();
			expect((recent as any).viewIds).toBeUndefined();
		});
	});

	describe("createActionDescription", () => {
		it("returns base description for single notification", () => {
			expect(createActionDescription("archive", 1)).toBe("Archived");
			expect(createActionDescription("markRead", 1)).toBe("Marked as read");
			expect(createActionDescription("star", 1)).toBe("Starred");
		});

		it("includes count for multiple notifications", () => {
			expect(createActionDescription("archive", 3)).toBe("Archived 3 notifications");
			expect(createActionDescription("mute", 10)).toBe("Muted 10 notifications");
		});

		it("includes tag name for tag actions", () => {
			expect(createActionDescription("assignTag", 1, { tagName: "urgent" })).toBe(
				'Added tag "urgent"'
			);
			expect(createActionDescription("removeTag", 1, { tagName: "bug" })).toBe('Removed tag "bug"');
		});

		it("includes tag name and count for bulk tag actions", () => {
			expect(createActionDescription("assignTag", 5, { tagName: "review" })).toBe(
				'Added tag "review" to 5 notifications'
			);
		});
	});

	describe("generateUndoActionId", () => {
		it("generates unique IDs", () => {
			const id1 = generateUndoActionId();
			const id2 = generateUndoActionId();

			expect(id1).not.toBe(id2);
		});

		it("generates IDs with undo prefix", () => {
			const id = generateUndoActionId();

			expect(id.startsWith("undo-")).toBe(true);
		});
	});

	describe("createUndoableAction", () => {
		it("creates an UndoableAction with all required fields", () => {
			const mockPerformUndo = async () => {};
			const notifications = [
				{
					id: "1",
					githubId: "gh-1",
					subjectTitle: "Test",
					repoFullName: "owner/repo",
					subjectType: "PullRequest",
					isRead: false,
				},
			];

			const action = createUndoableAction({
				type: "archive",
				notifications,
				performUndo: mockPerformUndo,
			});

			expect(action.id).toBeDefined();
			expect(action.id.startsWith("undo-")).toBe(true);
			expect(action.type).toBe("archive");
			expect(action.description).toBe("Archived");
			expect(action.notifications).toBe(notifications);
			expect(action.timestamp).toBeLessThanOrEqual(Date.now());
			expect(action.performUndo).toBe(mockPerformUndo);
		});

		it("includes metadata when provided", () => {
			const action = createUndoableAction({
				type: "assignTag",
				notifications: [
					{
						id: "1",
						githubId: "gh-1",
						subjectTitle: "Test",
						repoFullName: "owner/repo",
						subjectType: "Issue",
						isRead: false,
					},
				],
				metadata: { tagId: "tag-123", tagName: "urgent" },
				performUndo: async () => {},
			});

			expect(action.metadata).toEqual({ tagId: "tag-123", tagName: "urgent" });
			expect(action.description).toBe('Added tag "urgent"');
		});
	});
});
