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

import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";
import { createKeyboardNavigationStore } from "./keyboardNavigationStore";
import { createNotificationStore } from "./notificationStore";
import type { Notification, NotificationPage } from "$lib/api/types";

describe("KeyboardNavigationStore", () => {
	const createMockNotification = (id: string): Notification => ({
		id,
		githubId: `gh-${id}`,
		repoFullName: "owner/repo",
		reason: "review_requested",
		subjectTitle: `Test ${id}`,
		subjectType: "PullRequest",
		subjectUrl: `https://github.com/owner/repo/pull/${id}`,
		updatedAt: "2024-01-15T10:00:00Z",
		isRead: false,
		archived: false,
		muted: false,
		starred: false,
		filtered: false,
		snoozedUntil: null,
		tags: [],
		labels: [],
		viewIds: [],
	});

	const createMockPageData = (items: Notification[] = []): NotificationPage => ({
		items,
		total: items.length,
		page: 1,
		pageSize: 25,
	});

	const createStores = (notifications: Notification[] = []) => {
		const notificationStore = createNotificationStore(createMockPageData(notifications));
		const keyboardStore = createKeyboardNavigationStore(notificationStore);
		return { notificationStore, keyboardStore };
	};

	describe("initialization", () => {
		it("initializes with null focus index", () => {
			const { keyboardStore } = createStores();

			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});

		it("initializes with empty row components", () => {
			const { keyboardStore } = createStores();

			expect(get(keyboardStore.notificationRowComponents)).toEqual([]);
		});

		it("initializes with null pending focus action", () => {
			const { keyboardStore } = createStores();

			expect(get(keyboardStore.pendingFocusAction)).toBeNull();
		});
	});

	describe("focusAt", () => {
		it("sets focus to specified index", () => {
			const notifications = [
				createMockNotification("1"),
				createMockNotification("2"),
				createMockNotification("3"),
			];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(1);

			expect(get(keyboardStore.keyboardFocusIndex)).toBe(1);
		});

		it("clamps index to valid range (too high)", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(10);

			expect(get(keyboardStore.keyboardFocusIndex)).toBe(1); // Last valid index
		});

		it("clamps index to valid range (negative)", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(-5);

			expect(get(keyboardStore.keyboardFocusIndex)).toBe(0);
		});

		it("returns false and sets null when list is empty", () => {
			const { keyboardStore } = createStores([]);

			const result = keyboardStore.focusAt(0);

			expect(result).toBe(false);
			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});

		it("returns true when focus is set successfully", () => {
			const notifications = [createMockNotification("1")];
			const { keyboardStore } = createStores(notifications);

			const result = keyboardStore.focusAt(0);

			expect(result).toBe(true);
		});
	});

	describe("resetFocus", () => {
		it("sets focus index to null", () => {
			const notifications = [createMockNotification("1")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(0);
			keyboardStore.resetFocus();

			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});

		it("returns true", () => {
			const { keyboardStore } = createStores([]);

			const result = keyboardStore.resetFocus();

			expect(result).toBe(true);
		});

		it("calls blur on focused component if available", () => {
			const notifications = [createMockNotification("1")];
			const { keyboardStore } = createStores(notifications);

			// Mock a row component
			const mockBlur = vi.fn();
			keyboardStore.notificationRowComponents.set([{ blur: mockBlur } as any]);

			keyboardStore.focusAt(0);
			keyboardStore.resetFocus();

			expect(mockBlur).toHaveBeenCalled();
		});
	});

	describe("clearFocusIfNeeded", () => {
		it("clears focus when index is out of bounds (too high)", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore, notificationStore } = createStores(notifications);

			keyboardStore.focusAt(1);

			// Remove items so focus is out of bounds
			notificationStore.setPageData(createMockPageData([createMockNotification("1")]));

			keyboardStore.clearFocusIfNeeded();

			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});

		it("does not clear focus when index is valid", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(0);
			keyboardStore.clearFocusIfNeeded();

			expect(get(keyboardStore.keyboardFocusIndex)).toBe(0);
		});

		it("does nothing when focus is null", () => {
			const { keyboardStore } = createStores([]);

			keyboardStore.clearFocusIfNeeded();

			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});
	});

	describe("setPendingFocusAction", () => {
		it("sets pending focus action to first", () => {
			const { keyboardStore } = createStores([]);

			keyboardStore.setPendingFocusAction("first");

			expect(get(keyboardStore.pendingFocusAction)).toBe("first");
		});

		it("sets pending focus action to last", () => {
			const { keyboardStore } = createStores([]);

			keyboardStore.setPendingFocusAction("last");

			expect(get(keyboardStore.pendingFocusAction)).toBe("last");
		});

		it("clears pending focus action", () => {
			const { keyboardStore } = createStores([]);

			keyboardStore.setPendingFocusAction("first");
			keyboardStore.setPendingFocusAction(null);

			expect(get(keyboardStore.pendingFocusAction)).toBeNull();
		});
	});

	describe("executePendingFocusAction", () => {
		it("focuses first item when action is first", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.setPendingFocusAction("first");
			keyboardStore.executePendingFocusAction();

			expect(get(keyboardStore.keyboardFocusIndex)).toBe(0);
			expect(get(keyboardStore.pendingFocusAction)).toBeNull();
		});

		it("focuses last item when action is last", () => {
			const notifications = [
				createMockNotification("1"),
				createMockNotification("2"),
				createMockNotification("3"),
			];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.setPendingFocusAction("last");
			keyboardStore.executePendingFocusAction();

			expect(get(keyboardStore.keyboardFocusIndex)).toBe(2);
			expect(get(keyboardStore.pendingFocusAction)).toBeNull();
		});

		it("does nothing when no pending action", () => {
			const notifications = [createMockNotification("1")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.executePendingFocusAction();

			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});

		it("does nothing when list is empty", () => {
			const { keyboardStore } = createStores([]);

			keyboardStore.setPendingFocusAction("first");
			keyboardStore.executePendingFocusAction();

			expect(get(keyboardStore.keyboardFocusIndex)).toBeNull();
		});
	});

	describe("focusedNotification derived store", () => {
		it("returns null when no focus", () => {
			const notifications = [createMockNotification("1")];
			const { keyboardStore } = createStores(notifications);

			expect(get(keyboardStore.focusedNotification)).toBeNull();
		});

		it("returns focused notification", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(1);

			expect(get(keyboardStore.focusedNotification)?.id).toBe("2");
		});

		it("updates when focus changes", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { keyboardStore } = createStores(notifications);

			keyboardStore.focusAt(0);
			expect(get(keyboardStore.focusedNotification)?.id).toBe("1");

			keyboardStore.focusAt(1);
			expect(get(keyboardStore.focusedNotification)?.id).toBe("2");
		});

		it("returns null when index is out of range", () => {
			const notifications = [createMockNotification("1")];
			const { keyboardStore, notificationStore } = createStores(notifications);

			keyboardStore.focusAt(0);

			// Clear the notifications
			notificationStore.setPageData(createMockPageData([]));

			expect(get(keyboardStore.focusedNotification)).toBeNull();
		});
	});
});
