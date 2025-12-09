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
import { get } from "svelte/store";
import { createDetailViewStore } from "./detailViewStore";
import { createNotificationStore } from "./notificationStore";
import type { Notification, NotificationPage, NotificationDetail } from "$lib/api/types";

describe("DetailViewStore", () => {
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

	const createMockDetail = (id: string): NotificationDetail => ({
		notification: createMockNotification(id),
		subject: null,
	});

	const createStores = (notifications: Notification[] = []) => {
		const notificationStore = createNotificationStore(createMockPageData(notifications));
		const detailStore = createDetailViewStore(notificationStore);
		return { notificationStore, detailStore };
	};

	describe("initialization", () => {
		it("initializes with null detail notification id", () => {
			const { detailStore } = createStores();

			expect(get(detailStore.detailNotificationId)).toBeNull();
		});

		it("initializes with detail closed", () => {
			const { detailStore } = createStores();

			expect(get(detailStore.detailOpen)).toBe(false);
		});

		it("initializes with null current detail", () => {
			const { detailStore } = createStores();

			expect(get(detailStore.currentDetail)).toBeNull();
		});

		it("initializes loading/refreshing states as false", () => {
			const { detailStore } = createStores();

			expect(get(detailStore.detailLoading)).toBe(false);
			expect(get(detailStore.detailIsRefreshing)).toBe(false);
			expect(get(detailStore.detailShowingStaleData)).toBe(false);
		});
	});

	describe("openDetail", () => {
		it("sets detailNotificationId", () => {
			const { detailStore } = createStores();

			detailStore.openDetail("gh-1");

			expect(get(detailStore.detailNotificationId)).toBe("gh-1");
		});

		it("sets detailOpen to true", () => {
			const { detailStore } = createStores();

			detailStore.openDetail("gh-1");

			expect(get(detailStore.detailOpen)).toBe(true);
		});
	});

	describe("closeDetail", () => {
		it("sets detailNotificationId to null", () => {
			const { detailStore } = createStores();

			detailStore.openDetail("gh-1");
			detailStore.closeDetail();

			expect(get(detailStore.detailNotificationId)).toBeNull();
		});

		it("sets detailOpen to false", () => {
			const { detailStore } = createStores();

			detailStore.openDetail("gh-1");
			detailStore.closeDetail();

			expect(get(detailStore.detailOpen)).toBe(false);
		});

		it("clears currentDetail", () => {
			const { detailStore } = createStores();

			detailStore.openDetail("gh-1");
			detailStore.setDetail(createMockDetail("1"));
			detailStore.closeDetail();

			expect(get(detailStore.currentDetail)).toBeNull();
		});

		it("resets loading and refreshing states", () => {
			const { detailStore } = createStores();

			detailStore.openDetail("gh-1");
			detailStore.setLoading(true);
			detailStore.setRefreshing(true);
			detailStore.setShowingStaleData(true);
			detailStore.closeDetail();

			expect(get(detailStore.detailLoading)).toBe(false);
			expect(get(detailStore.detailIsRefreshing)).toBe(false);
			expect(get(detailStore.detailShowingStaleData)).toBe(false);
		});
	});

	describe("setDetail", () => {
		it("sets currentDetail", () => {
			const { detailStore } = createStores();
			const detail = createMockDetail("1");

			detailStore.setDetail(detail);

			expect(get(detailStore.currentDetail)).toEqual(detail);
		});

		it("can set null", () => {
			const { detailStore } = createStores();

			detailStore.setDetail(createMockDetail("1"));
			detailStore.setDetail(null);

			expect(get(detailStore.currentDetail)).toBeNull();
		});
	});

	describe("loading states", () => {
		it("setLoading updates detailLoading", () => {
			const { detailStore } = createStores();

			detailStore.setLoading(true);
			expect(get(detailStore.detailLoading)).toBe(true);

			detailStore.setLoading(false);
			expect(get(detailStore.detailLoading)).toBe(false);
		});

		it("setRefreshing updates detailIsRefreshing", () => {
			const { detailStore } = createStores();

			detailStore.setRefreshing(true);
			expect(get(detailStore.detailIsRefreshing)).toBe(true);

			detailStore.setRefreshing(false);
			expect(get(detailStore.detailIsRefreshing)).toBe(false);
		});

		it("setShowingStaleData updates detailShowingStaleData", () => {
			const { detailStore } = createStores();

			detailStore.setShowingStaleData(true);
			expect(get(detailStore.detailShowingStaleData)).toBe(true);

			detailStore.setShowingStaleData(false);
			expect(get(detailStore.detailShowingStaleData)).toBe(false);
		});
	});

	describe("currentDetailNotification", () => {
		it("returns null when no detail is open", () => {
			const notifications = [createMockNotification("1")];
			const { detailStore } = createStores(notifications);

			expect(get(detailStore.currentDetailNotification)).toBeNull();
		});

		it("returns notification from store", () => {
			const notification = createMockNotification("1");
			const { detailStore } = createStores([notification]);

			detailStore.openDetail("gh-1");

			const result = get(detailStore.currentDetailNotification);
			expect(result?.id).toBe("1");
			expect(result?.githubId).toBe("gh-1");
		});

		it("returns null when notification not found", () => {
			const { detailStore } = createStores([]);

			detailStore.openDetail("non-existent");

			expect(get(detailStore.currentDetailNotification)).toBeNull();
		});

		it("updates when notification is updated", () => {
			const notification = createMockNotification("1");
			const { notificationStore, detailStore } = createStores([notification]);

			detailStore.openDetail("gh-1");
			expect(get(detailStore.currentDetailNotification)?.isRead).toBe(false);

			// Update the notification
			notificationStore.updateNotification({ ...notification, isRead: true });

			expect(get(detailStore.currentDetailNotification)?.isRead).toBe(true);
		});
	});

	describe("detailNotificationIndex", () => {
		it("returns null when no detail is open", () => {
			const notifications = [createMockNotification("1")];
			const { detailStore } = createStores(notifications);

			expect(get(detailStore.detailNotificationIndex)).toBeNull();
		});

		it("returns correct index", () => {
			const notifications = [
				createMockNotification("1"),
				createMockNotification("2"),
				createMockNotification("3"),
			];
			const { detailStore } = createStores(notifications);

			detailStore.openDetail("gh-2");

			expect(get(detailStore.detailNotificationIndex)).toBe(1);
		});

		it("returns null when notification not in page", () => {
			const notifications = [createMockNotification("1")];
			const { detailStore } = createStores(notifications);

			detailStore.openDetail("non-existent");

			expect(get(detailStore.detailNotificationIndex)).toBeNull();
		});

		it("updates when page data changes", () => {
			const notifications = [createMockNotification("1"), createMockNotification("2")];
			const { notificationStore, detailStore } = createStores(notifications);

			detailStore.openDetail("gh-2");
			expect(get(detailStore.detailNotificationIndex)).toBe(1);

			// Remove first notification, so gh-2 moves to index 0
			notificationStore.removeNotification("gh-1");

			expect(get(detailStore.detailNotificationIndex)).toBe(0);
		});
	});
});
