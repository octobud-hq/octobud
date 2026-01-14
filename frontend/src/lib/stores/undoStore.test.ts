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

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { get } from "svelte/store";

// Mock the API modules before importing undoStore
vi.mock("$lib/api/notifications", () => ({
	markNotificationRead: vi.fn(),
	markNotificationUnread: vi.fn(),
	archiveNotification: vi.fn(),
	unarchiveNotification: vi.fn(),
	muteNotification: vi.fn(),
	unmuteNotification: vi.fn(),
	snoozeNotification: vi.fn(),
	unsnoozeNotification: vi.fn(),
	starNotification: vi.fn(),
	unstarNotification: vi.fn(),
	bulkMarkNotificationsRead: vi.fn(),
	bulkMarkNotificationsUnread: vi.fn(),
	bulkArchiveNotifications: vi.fn(),
	bulkUnarchiveNotifications: vi.fn(),
	bulkMuteNotifications: vi.fn(),
	bulkUnmuteNotifications: vi.fn(),
	bulkSnoozeNotifications: vi.fn(),
	bulkUnsnoozeNotifications: vi.fn(),
	bulkStarNotifications: vi.fn(),
	bulkUnstarNotifications: vi.fn(),
	bulkAssignTag: vi.fn(),
	bulkRemoveTag: vi.fn(),
}));

vi.mock("$lib/api/tags", () => ({
	assignTagToNotification: vi.fn(),
	removeTagFromNotification: vi.fn(),
}));

// Mock localStorage
const localStorageMock = (() => {
	let store: Record<string, string> = {};
	return {
		getItem: (key: string) => store[key] || null,
		setItem: (key: string, value: string) => {
			store[key] = value;
		},
		removeItem: (key: string) => {
			delete store[key];
		},
		clear: () => {
			store = {};
		},
	};
})();
Object.defineProperty(globalThis, "localStorage", { value: localStorageMock });

// Mock toastStore
vi.mock("./toastStore", () => ({
	toastStore: {
		showWithUndo: vi.fn(() => "toast-id-123"),
		dismiss: vi.fn(),
		morphToUndone: vi.fn(),
		success: vi.fn(),
		error: vi.fn(),
		info: vi.fn(),
	},
}));

// Now import the module under test
import { undoStore } from "./undoStore";
import { toastStore } from "./toastStore";
import type { UndoableAction, RecentActionNotification } from "$lib/undo/types";

describe("UndoStore", () => {
	const createMockNotification = (
		overrides: Partial<RecentActionNotification> = {}
	): RecentActionNotification => ({
		id: "1",
		githubId: "gh-1",
		subjectTitle: "Test Notification",
		repoFullName: "owner/repo",
		subjectType: "PullRequest",
		isRead: false,
		...overrides,
	});

	const createMockAction = (overrides: Partial<UndoableAction> = {}): UndoableAction => ({
		id: `undo-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`,
		type: "archive",
		description: "Archived",
		notifications: [createMockNotification()],
		timestamp: Date.now(),
		performUndo: vi.fn().mockResolvedValue(undefined),
		...overrides,
	});

	beforeEach(() => {
		vi.useFakeTimers();
		localStorageMock.clear();
		// Fully reset the store state (clears history, activeToastAction, and timeouts)
		undoStore._reset();
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	describe("initialization", () => {
		it("initializes with empty history", () => {
			expect(get(undoStore.history)).toEqual([]);
			expect(get(undoStore.activeToastAction)).toBeNull();
			expect(get(undoStore.isUndoing)).toBe(false);
		});

		it("initializes with empty history count", () => {
			expect(get(undoStore.historyCount)).toBe(0);
			expect(get(undoStore.hasHistory)).toBe(false);
		});
	});

	describe("pushAction", () => {
		it("sets action as activeToastAction", () => {
			const action = createMockAction();

			undoStore.pushAction(action);

			expect(get(undoStore.activeToastAction)).toEqual(action);
		});

		it("shows toast with undo support", () => {
			const action = createMockAction({ description: "Archived 3 notifications" });

			undoStore.pushAction(action);

			expect(toastStore.showWithUndo).toHaveBeenCalledWith(
				"Archived 3 notifications",
				"success",
				7000,
				expect.any(Function)
			);
		});

		it("stores both actions in history when new action is pushed", () => {
			const action1 = createMockAction({ id: "action-1" });
			const action2 = createMockAction({ id: "action-2" });

			undoStore.pushAction(action1);
			undoStore.pushAction(action2);

			expect(get(undoStore.activeToastAction)?.id).toBe("action-2");
			// Both actions are in history, with newest at index 0
			expect(get(undoStore.history)).toHaveLength(2);
			expect(get(undoStore.history)[0].id).toBe("action-2");
			expect(get(undoStore.history)[1].id).toBe("action-1");
		});

		it("dismisses previous toast when new action is pushed", () => {
			const action1 = createMockAction();
			const action2 = createMockAction();

			undoStore.pushAction(action1);
			undoStore.pushAction(action2);

			expect(toastStore.dismiss).toHaveBeenCalled();
		});
	});

	describe("toast expiration", () => {
		it("action is stored immediately but activeToastAction clears after timeout", () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			// Actions are now stored in history immediately
			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.activeToastAction)?.id).toBe(action.id);

			// Fast-forward past the toast duration
			vi.advanceTimersByTime(7000);

			// activeToastAction clears but action remains in history
			expect(get(undoStore.activeToastAction)).toBeNull();
			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].id).toBe(action.id);
		});
	});

	describe("undoActiveToast", () => {
		it("calls performUndo on the active action", async () => {
			const mockPerformUndo = vi.fn().mockResolvedValue(undefined);
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			await undoStore.undoActiveToast();

			expect(mockPerformUndo).toHaveBeenCalled();
		});

		it("morphs toast to undone state on success", async () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			await undoStore.undoActiveToast();

			expect(toastStore.morphToUndone).toHaveBeenCalledWith("toast-id-123");
		});

		it("clears activeToastAction on success", async () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			await undoStore.undoActiveToast();

			expect(get(undoStore.activeToastAction)).toBeNull();
		});

		it("returns false if no active toast action", async () => {
			const result = await undoStore.undoActiveToast();

			expect(result).toBe(false);
		});

		it("returns false if already undoing", async () => {
			const slowPerformUndo = vi
				.fn()
				.mockImplementation(() => new Promise((resolve) => setTimeout(resolve, 1000)));
			const action = createMockAction({ performUndo: slowPerformUndo });

			undoStore.pushAction(action);

			// Start first undo (don't await)
			const promise1 = undoStore.undoActiveToast();
			// Try second undo immediately
			const result = await undoStore.undoActiveToast();

			expect(result).toBe(false);

			// Cleanup
			vi.advanceTimersByTime(1000);
			await promise1;
		});

		it("shows error toast on failure", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("Network error"));
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			await undoStore.undoActiveToast();

			expect(toastStore.error).toHaveBeenCalled();
		});
	});

	describe("undoFromHistory", () => {
		it("calls performUndo on the specified action", async () => {
			const mockPerformUndo = vi.fn().mockResolvedValue(undefined);
			const action = createMockAction({ performUndo: mockPerformUndo });

			// Add to history by pushing and letting toast expire
			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);

			await undoStore.undoFromHistory(action.id);

			expect(mockPerformUndo).toHaveBeenCalled();
		});

		it("removes action from history on success", async () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(1);

			await undoStore.undoFromHistory(action.id);

			expect(get(undoStore.history)).toHaveLength(0);
		});

		it("shows success toast on success", async () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);
			await undoStore.undoFromHistory(action.id);

			expect(toastStore.success).toHaveBeenCalledWith("Undone", 1500);
		});

		it("returns false if action not found", async () => {
			const result = await undoStore.undoFromHistory("non-existent-id");

			expect(result).toBe(false);
		});
	});

	describe("canUndoViaKeyboard", () => {
		it("returns true when active toast action exists and not undoing", () => {
			const action = createMockAction();

			undoStore.pushAction(action);

			expect(undoStore.canUndoViaKeyboard()).toBe(true);
		});

		it("returns false when no active toast action", () => {
			expect(undoStore.canUndoViaKeyboard()).toBe(false);
		});
	});

	describe("removeFromHistory", () => {
		it("removes specified action from history", () => {
			const action1 = createMockAction({ id: "action-1" });
			const action2 = createMockAction({ id: "action-2" });

			undoStore.pushAction(action1);
			vi.advanceTimersByTime(7000);
			undoStore.pushAction(action2);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(2);

			undoStore.removeFromHistory("action-1");

			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].id).toBe("action-2");
		});
	});

	describe("clearHistory", () => {
		it("clears all history", () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(1);

			undoStore.clearHistory();

			expect(get(undoStore.history)).toHaveLength(0);
		});
	});

	describe("localStorage persistence", () => {
		it("saves to localStorage immediately on push (not waiting for toast expiry)", () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			// Don't advance timers - action should already be persisted

			const stored = localStorageMock.getItem("octobud_undo_history");
			expect(stored).toBeDefined();

			const parsed = JSON.parse(stored!);
			expect(parsed).toHaveLength(1);
			expect(parsed[0].id).toBe(action.id);
		});

		it("action is available in both history and localStorage while toast is visible", () => {
			const action = createMockAction();

			undoStore.pushAction(action);

			// Action should be in history immediately (available for undo)
			expect(get(undoStore.history)).toHaveLength(1);

			// And also persisted to localStorage (survives page refresh)
			const stored = localStorageMock.getItem("octobud_undo_history");
			expect(stored).toBeDefined();
			const parsed = JSON.parse(stored!);
			expect(parsed).toHaveLength(1);
			expect(parsed[0].id).toBe(action.id);
		});

		it("clears localStorage when history is cleared", () => {
			const action = createMockAction();

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);
			undoStore.clearHistory();

			const stored = localStorageMock.getItem("octobud_undo_history");
			expect(JSON.parse(stored!)).toHaveLength(0);
		});

		it("removes action from localStorage when undone via toast", async () => {
			const action = createMockAction();

			undoStore.pushAction(action);

			// Verify it's stored
			let stored = localStorageMock.getItem("octobud_undo_history");
			expect(JSON.parse(stored!)).toHaveLength(1);

			// Undo via toast
			await undoStore.undoActiveToast();

			// Should be removed from localStorage
			stored = localStorageMock.getItem("octobud_undo_history");
			expect(JSON.parse(stored!)).toHaveLength(0);
		});
	});

	describe("MAX_HISTORY_SIZE", () => {
		it("limits history to 20 items", () => {
			// Push 25 actions on different notifications to avoid cancellation
			for (let i = 0; i < 25; i++) {
				const action = createMockAction({
					id: `action-${i}`,
					notifications: [createMockNotification({ id: `notif-${i}`, githubId: `gh-${i}` })],
				});
				undoStore.pushAction(action);
				vi.advanceTimersByTime(7000);
			}

			expect(get(undoStore.history).length).toBeLessThanOrEqual(20);
		});
	});

	describe("inverse cancellation", () => {
		it("removes inverse action from history when pushing new action", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });

			// Star the notification
			const starAction = createMockAction({
				id: "star-1",
				type: "star",
				notifications: [notification],
			});
			undoStore.pushAction(starAction);
			vi.advanceTimersByTime(7000); // Move to history

			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].type).toBe("star");

			// Unstar the same notification - should cancel out the star
			const unstarAction = createMockAction({
				id: "unstar-1",
				type: "unstar",
				notifications: [notification],
			});
			undoStore.pushAction(unstarAction);
			vi.advanceTimersByTime(7000);

			// History should only have the unstar (star was removed)
			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].type).toBe("unstar");
		});

		it("removes inverse action from history when inverse is pushed", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });

			// Star the notification (active toast and stored in history immediately)
			const starAction = createMockAction({
				id: "star-1",
				type: "star",
				notifications: [notification],
			});
			undoStore.pushAction(starAction);

			expect(get(undoStore.activeToastAction)?.type).toBe("star");
			expect(get(undoStore.history)).toHaveLength(1);

			// Unstar while star toast is still active
			const unstarAction = createMockAction({
				id: "unstar-1",
				type: "unstar",
				notifications: [notification],
			});
			undoStore.pushAction(unstarAction);

			// Active toast should now be unstar, star should be removed from history
			// The new action (unstar) is stored in history
			expect(get(undoStore.activeToastAction)?.type).toBe("unstar");
			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].type).toBe("unstar");
		});

		it("rapid toggling results in single history entry", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });

			// Rapidly toggle star/unstar
			for (let i = 0; i < 5; i++) {
				undoStore.pushAction(
					createMockAction({
						id: `star-${i}`,
						type: "star",
						notifications: [notification],
					})
				);
				undoStore.pushAction(
					createMockAction({
						id: `unstar-${i}`,
						type: "unstar",
						notifications: [notification],
					})
				);
			}

			// Final action
			undoStore.pushAction(
				createMockAction({
					id: "star-final",
					type: "star",
					notifications: [notification],
				})
			);
			vi.advanceTimersByTime(7000);

			// Should only have the final star action
			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].type).toBe("star");
		});

		it("does not cancel actions on different notifications", () => {
			const notification1 = createMockNotification({ id: "1", githubId: "gh-1" });
			const notification2 = createMockNotification({ id: "2", githubId: "gh-2" });

			// Star notification 1
			undoStore.pushAction(
				createMockAction({
					id: "star-1",
					type: "star",
					notifications: [notification1],
				})
			);
			vi.advanceTimersByTime(7000);

			// Unstar notification 2 - should NOT cancel star on notification 1
			undoStore.pushAction(
				createMockAction({
					id: "unstar-2",
					type: "unstar",
					notifications: [notification2],
				})
			);
			vi.advanceTimersByTime(7000);

			// Both should be in history
			expect(get(undoStore.history)).toHaveLength(2);
		});

		it("handles archive/unarchive cancellation", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });

			// Archive
			undoStore.pushAction(
				createMockAction({
					id: "archive-1",
					type: "archive",
					notifications: [notification],
				})
			);
			vi.advanceTimersByTime(7000);

			// Unarchive - should cancel
			undoStore.pushAction(
				createMockAction({
					id: "unarchive-1",
					type: "unarchive",
					notifications: [notification],
				})
			);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].type).toBe("unarchive");
		});

		it("handles bulk action cancellation with matching notification sets", () => {
			const notifications = [
				createMockNotification({ id: "1", githubId: "gh-1" }),
				createMockNotification({ id: "2", githubId: "gh-2" }),
				createMockNotification({ id: "3", githubId: "gh-3" }),
			];

			// Bulk archive
			undoStore.pushAction(
				createMockAction({
					id: "archive-bulk",
					type: "archive",
					notifications: notifications,
				})
			);
			vi.advanceTimersByTime(7000);

			// Bulk unarchive same set - should cancel
			undoStore.pushAction(
				createMockAction({
					id: "unarchive-bulk",
					type: "unarchive",
					notifications: notifications,
				})
			);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(1);
			expect(get(undoStore.history)[0].type).toBe("unarchive");
		});

		it("does not cancel bulk actions with different notification sets", () => {
			const notification1 = createMockNotification({ id: "1", githubId: "gh-1" });
			const notification2 = createMockNotification({ id: "2", githubId: "gh-2" });

			// Archive notification 1
			undoStore.pushAction(
				createMockAction({
					id: "archive-1",
					type: "archive",
					notifications: [notification1],
				})
			);
			vi.advanceTimersByTime(7000);

			// Unarchive notifications 1 and 2 - different set, should NOT cancel
			undoStore.pushAction(
				createMockAction({
					id: "unarchive-bulk",
					type: "unarchive",
					notifications: [notification1, notification2],
				})
			);
			vi.advanceTimersByTime(7000);

			// Both should be in history (different notification sets)
			expect(get(undoStore.history)).toHaveLength(2);
		});
	});

	describe("error handling", () => {
		it("shows specific error for 404 responses", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("404 not found"));
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);
			await undoStore.undoFromHistory(action.id);

			expect(toastStore.error).toHaveBeenCalledWith("Cannot undo: Notification no longer exists");
		});

		it("shows specific error for tag-related 404", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("404 not found"));
			const action = createMockAction({
				type: "assignTag",
				performUndo: mockPerformUndo,
			});

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);
			await undoStore.undoFromHistory(action.id);

			expect(toastStore.error).toHaveBeenCalledWith("Cannot undo: Tag no longer exists");
		});

		it("shows specific error for 403 responses", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("403 permission denied"));
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);
			await undoStore.undoFromHistory(action.id);

			expect(toastStore.error).toHaveBeenCalledWith("Cannot undo: You no longer have access");
		});

		it("shows network error for other failures", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("Connection refused"));
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);
			await undoStore.undoFromHistory(action.id);

			expect(toastStore.error).toHaveBeenCalledWith("Cannot undo: Unable to connect to server");
		});

		it("removes action from history on 404/403 errors", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("404 not found"));
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(1);

			await undoStore.undoFromHistory(action.id);

			expect(get(undoStore.history)).toHaveLength(0);
		});

		it("keeps action in history on network errors for retry", async () => {
			const mockPerformUndo = vi.fn().mockRejectedValue(new Error("Network timeout"));
			const action = createMockAction({ performUndo: mockPerformUndo });

			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);

			expect(get(undoStore.history)).toHaveLength(1);

			await undoStore.undoFromHistory(action.id);

			// Action should still be in history for retry
			expect(get(undoStore.history)).toHaveLength(1);
		});
	});

	describe("inverse operation metadata validation", () => {
		it("throws error when undoing unsnooze without snoozedUntil metadata", async () => {
			// Simulate an action that was serialized/deserialized (uses real performUndo)
			// by putting it directly in localStorage and reinitializing
			const serializedHistory = [
				{
					id: "unsnooze-action",
					type: "unsnooze",
					description: "Unsnoozed",
					notifications: [
						{
							id: "notif-1",
							githubId: "gh-1",
							subjectTitle: "Test",
							repoFullName: "owner/repo",
							subjectType: "PullRequest",
							isRead: false,
						},
					],
					timestamp: Date.now(),
					metadata: {}, // Missing snoozedUntil - this should cause the error
					notificationKeys: ["gh-1"],
				},
			];

			undoStore._reset();
			localStorageMock.setItem("octobud_undo_history", JSON.stringify(serializedHistory));
			undoStore.initialize();

			// The undo should fail due to missing metadata
			await undoStore.undoFromHistory("unsnooze-action");

			// Should show error toast (the error is caught and displayed)
			expect(toastStore.error).toHaveBeenCalled();
		});

		it("throws error when undoing removeTag without tagId metadata", async () => {
			const serializedHistory = [
				{
					id: "remove-tag-action",
					type: "removeTag",
					description: "Removed tag",
					notifications: [
						{
							id: "notif-1",
							githubId: "gh-1",
							subjectTitle: "Test",
							repoFullName: "owner/repo",
							subjectType: "Issue",
							isRead: false,
						},
					],
					timestamp: Date.now(),
					metadata: {}, // Missing tagId
					notificationKeys: ["gh-1"],
				},
			];

			undoStore._reset();
			localStorageMock.setItem("octobud_undo_history", JSON.stringify(serializedHistory));
			undoStore.initialize();

			await undoStore.undoFromHistory("remove-tag-action");

			expect(toastStore.error).toHaveBeenCalled();
		});

		it("throws error when undoing assignTag without tagId metadata", async () => {
			const serializedHistory = [
				{
					id: "assign-tag-action",
					type: "assignTag",
					description: "Added tag",
					notifications: [
						{
							id: "notif-1",
							githubId: "gh-1",
							subjectTitle: "Test",
							repoFullName: "owner/repo",
							subjectType: "Issue",
							isRead: false,
						},
					],
					timestamp: Date.now(),
					metadata: {}, // Missing tagId
					notificationKeys: ["gh-1"],
				},
			];

			undoStore._reset();
			localStorageMock.setItem("octobud_undo_history", JSON.stringify(serializedHistory));
			undoStore.initialize();

			await undoStore.undoFromHistory("assign-tag-action");

			expect(toastStore.error).toHaveBeenCalled();
		});
	});

	describe("localStorage serialization", () => {
		it("serializes and deserializes actions correctly (roundtrip)", () => {
			const notification = createMockNotification({
				id: "notif-1",
				githubId: "gh-notif-1",
				subjectTitle: "Test PR",
				repoFullName: "owner/repo",
			});
			const action = createMockAction({
				id: "action-roundtrip",
				type: "archive",
				description: "Archived",
				notifications: [notification],
				metadata: { tagId: "tag-123", tagName: "important" },
			});

			// Push action and let it move to history
			undoStore.pushAction(action);
			vi.advanceTimersByTime(7000);

			// Verify it's in localStorage
			const stored = localStorageMock.getItem("octobud_undo_history");
			expect(stored).toBeDefined();

			const parsed = JSON.parse(stored!);
			expect(parsed).toHaveLength(1);
			expect(parsed[0].id).toBe("action-roundtrip");
			expect(parsed[0].type).toBe("archive");
			expect(parsed[0].notifications[0].subjectTitle).toBe("Test PR");
			expect(parsed[0].metadata.tagId).toBe("tag-123");
			expect(parsed[0].notificationKeys).toEqual(["gh-notif-1"]);
		});

		it("initializes store with history from localStorage", () => {
			// Reset first (this clears localStorage)
			undoStore._reset();

			// Then pre-populate localStorage with serialized action
			const serializedHistory = [
				{
					id: "stored-action-1",
					type: "star",
					description: "Starred",
					notifications: [
						{
							id: "notif-1",
							githubId: "gh-1",
							subjectTitle: "Stored Notification",
							repoFullName: "owner/repo",
							subjectType: "Issue",
							isRead: false,
						},
					],
					timestamp: Date.now() - 60000,
					notificationKeys: ["gh-1"],
				},
			];
			localStorageMock.setItem("octobud_undo_history", JSON.stringify(serializedHistory));

			// Now initialize - it should load from localStorage
			undoStore.initialize();

			// History should be loaded from localStorage
			const history = get(undoStore.history);
			expect(history).toHaveLength(1);
			expect(history[0].id).toBe("stored-action-1");
			expect(history[0].type).toBe("star");
			expect(history[0].notifications[0].subjectTitle).toBe("Stored Notification");
		});

		it("setRefreshCallback rebuilds history with new callback", async () => {
			// Reset first
			undoStore._reset();

			// Pre-populate localStorage
			const serializedHistory = [
				{
					id: "stored-action-1",
					type: "archive",
					description: "Archived",
					notifications: [
						{
							id: "notif-1",
							githubId: "gh-1",
							subjectTitle: "Test",
							repoFullName: "owner/repo",
							subjectType: "PullRequest",
							isRead: false,
						},
					],
					timestamp: Date.now() - 60000,
					notificationKeys: ["gh-1"],
				},
			];
			localStorageMock.setItem("octobud_undo_history", JSON.stringify(serializedHistory));

			// Initialize store
			undoStore.initialize();

			// History should have the action
			expect(get(undoStore.history)).toHaveLength(1);

			// Set up a mock refresh callback - this should rebuild history
			const mockRefresh = vi.fn().mockResolvedValue(undefined);
			undoStore.setRefreshCallback(mockRefresh);

			// History should still have the action
			const history = get(undoStore.history);
			expect(history).toHaveLength(1);

			// The performUndo function should now include the refresh callback
			expect(history[0].performUndo).toBeDefined();
			expect(typeof history[0].performUndo).toBe("function");
		});

		it("handles corrupted localStorage gracefully", () => {
			// Set corrupted data
			localStorageMock.setItem("octobud_undo_history", "not valid json{{{");

			// Reset and reinitialize - should not throw
			undoStore._reset();
			expect(() => undoStore.initialize()).not.toThrow();

			// History should be empty
			expect(get(undoStore.history)).toHaveLength(0);
		});
	});
});
