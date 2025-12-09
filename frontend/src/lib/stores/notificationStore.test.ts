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

import { describe, it, expect, beforeEach } from "vitest";
import { get } from "svelte/store";
import { createNotificationStore } from "./notificationStore";
import type { Notification, NotificationPage } from "$lib/api/types";

describe("NotificationStore", () => {
	const createMockNotification = (overrides: Partial<Notification> = {}): Notification => ({
		id: "1",
		githubId: "gh-1",
		repoFullName: "owner/repo",
		reason: "review_requested",
		subjectTitle: "Test PR",
		subjectType: "PullRequest",
		subjectUrl: "https://github.com/owner/repo/pull/1",
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
		...overrides,
	});

	const createMockPageData = (
		items: Notification[] = [],
		overrides: Partial<NotificationPage> = {}
	): NotificationPage => ({
		items,
		total: items.length,
		page: 1,
		pageSize: 25,
		...overrides,
	});

	describe("initialization", () => {
		it("initializes with empty page data", () => {
			const store = createNotificationStore(createMockPageData());

			expect(get(store.pageData).items).toEqual([]);
			expect(get(store.pageData).total).toBe(0);
			expect(get(store.notificationsById).size).toBe(0);
		});

		it("initializes with provided notifications", () => {
			const notifications = [
				createMockNotification({ id: "1", githubId: "gh-1" }),
				createMockNotification({ id: "2", githubId: "gh-2" }),
			];
			const store = createNotificationStore(createMockPageData(notifications, { total: 2 }));

			expect(get(store.pageData).items).toHaveLength(2);
			expect(get(store.notificationsById).size).toBe(2);
			expect(get(store.notificationsById).get("gh-1")).toBeDefined();
			expect(get(store.notificationsById).get("gh-2")).toBeDefined();
		});

		it("uses id as key when githubId is not available", () => {
			const notification = createMockNotification({ id: "1", githubId: undefined });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			expect(get(store.notificationsById).get("1")).toBeDefined();
		});
	});

	describe("getNotification", () => {
		it("returns notification by id", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			expect(store.getNotification("gh-1")).toEqual(notification);
		});

		it("returns null for non-existent id", () => {
			const store = createNotificationStore(createMockPageData());

			expect(store.getNotification("non-existent")).toBeNull();
		});
	});

	describe("updateNotification", () => {
		it("updates notification in both stores", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1", isRead: false });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			const updated = { ...notification, isRead: true };
			store.updateNotification(updated);

			expect(get(store.notificationsById).get("gh-1")?.isRead).toBe(true);
			expect(get(store.pageData).items[0].isRead).toBe(true);
		});

		it("handles notification not in normalized store gracefully", () => {
			const store = createNotificationStore(createMockPageData());

			const newNotification = createMockNotification({ id: "1", githubId: "gh-1" });
			store.updateNotification(newNotification);

			expect(get(store.notificationsById).get("gh-1")).toEqual(newNotification);
		});

		it("preserves optimistic read status when refreshing", () => {
			// Start with an unread notification
			const notification = createMockNotification({ id: "1", githubId: "gh-1", isRead: false });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			// Optimistically mark as read
			const optimisticUpdate = { ...notification, isRead: true };
			store.updateNotification(optimisticUpdate);
			expect(get(store.notificationsById).get("gh-1")?.isRead).toBe(true);

			// Simulate refresh returning stale data (isRead: false)
			const refreshedNotification = {
				...notification,
				subjectTitle: "Updated Title",
				isRead: false,
			};
			store.updateNotification(refreshedNotification);

			// Should preserve the optimistic read status
			expect(get(store.notificationsById).get("gh-1")?.isRead).toBe(true);
			expect(get(store.notificationsById).get("gh-1")?.subjectTitle).toBe("Updated Title");
		});

		it("does not preserve read status if notification was not previously read", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1", isRead: false });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			// Refresh with unread status
			const refreshedNotification = {
				...notification,
				subjectTitle: "Updated Title",
				isRead: false,
			};
			store.updateNotification(refreshedNotification);

			// Should remain unread
			expect(get(store.notificationsById).get("gh-1")?.isRead).toBe(false);
		});
	});

	describe("setPageData", () => {
		it("replaces page data and updates normalized store", () => {
			const initial = createMockNotification({ id: "1", githubId: "gh-1" });
			const store = createNotificationStore(createMockPageData([initial], { total: 1 }));

			const newNotifications = [
				createMockNotification({ id: "2", githubId: "gh-2" }),
				createMockNotification({ id: "3", githubId: "gh-3" }),
			];
			store.setPageData(createMockPageData(newNotifications, { total: 2, page: 2 }));

			expect(get(store.pageData).items).toHaveLength(2);
			expect(get(store.pageData).page).toBe(2);
			// Original notification still in normalized store
			expect(get(store.notificationsById).get("gh-1")).toBeDefined();
			// New notifications added
			expect(get(store.notificationsById).get("gh-2")).toBeDefined();
			expect(get(store.notificationsById).get("gh-3")).toBeDefined();
		});
	});

	describe("removeNotification", () => {
		it("removes notification from both stores", () => {
			const notifications = [
				createMockNotification({ id: "1", githubId: "gh-1" }),
				createMockNotification({ id: "2", githubId: "gh-2" }),
			];
			const store = createNotificationStore(createMockPageData(notifications, { total: 2 }));

			const removed = store.removeNotification("gh-1");

			expect(removed?.githubId).toBe("gh-1");
			expect(get(store.notificationsById).has("gh-1")).toBe(false);
			expect(get(store.pageData).items).toHaveLength(1);
			expect(get(store.pageData).items[0].githubId).toBe("gh-2");
			expect(get(store.pageData).total).toBe(1);
		});

		it("returns null for non-existent notification", () => {
			const store = createNotificationStore(createMockPageData());

			const removed = store.removeNotification("non-existent");

			expect(removed).toBeNull();
		});

		it("decrements total when removing", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });
			const store = createNotificationStore(createMockPageData([notification], { total: 5 }));

			store.removeNotification("gh-1");

			expect(get(store.pageData).total).toBe(4);
		});

		it("does not go below zero for total", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });
			const store = createNotificationStore(createMockPageData([notification], { total: 0 }));

			store.removeNotification("gh-1");

			expect(get(store.pageData).total).toBe(0);
		});
	});

	describe("restoreNotification", () => {
		it("restores a removed notification", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			const removed = store.removeNotification("gh-1");
			expect(get(store.pageData).items).toHaveLength(0);

			store.restoreNotification(removed!);

			expect(get(store.notificationsById).get("gh-1")).toBeDefined();
			expect(get(store.pageData).items).toHaveLength(1);
			expect(get(store.pageData).total).toBe(1);
		});

		it("does not duplicate if already exists", () => {
			const notification = createMockNotification({ id: "1", githubId: "gh-1" });
			const store = createNotificationStore(createMockPageData([notification], { total: 1 }));

			store.restoreNotification(notification);

			expect(get(store.pageData).items).toHaveLength(1);
		});
	});
});
