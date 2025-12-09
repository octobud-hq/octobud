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

import { describe, it, expect, vi, beforeEach } from "vitest";
import { writable, get } from "svelte/store";
import { createNotificationActionController } from "./notificationActionController";
import { createOptimisticUpdateHelpers } from "./optimisticUpdates";
import { createSharedHelpers } from "./sharedHelpers";
import { createDebounceManager } from "./debounceManager";
import {
	markNotificationRead,
	markNotificationUnread,
	archiveNotification,
	unarchiveNotification,
} from "$lib/api/notifications";
import { toastStore } from "$lib/stores/toastStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
import type { DetailStore } from "../../stores/detailViewStore";
import type { SelectionStore } from "../../stores/selectionStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { UIStore } from "../../stores/uiStateStore";
import type { QueryStore } from "../../stores/queryStore";
import type { ControllerOptions } from "../interfaces/common";
import type { Notification } from "$lib/api/types";

// Helper to create mock notifications
const createMockNotification = (overrides: Partial<Notification> = {}): Notification => ({
	id: "1",
	githubId: "gh-1",
	repoFullName: "owner/repo",
	reason: "review_requested",
	subjectTitle: "Test Notification",
	subjectType: "pull_request",
	updatedAt: "2024-01-15T10:00:00Z",
	isRead: false,
	archived: false,
	muted: false,
	starred: false,
	labels: [],
	viewIds: [],
	...overrides,
});

// Mock API functions
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
	unfilterNotification: vi.fn(),
}));

// Mock toast store
vi.mock("$lib/stores/toastStore", () => ({
	toastStore: {
		info: vi.fn(),
		error: vi.fn(),
	},
}));

describe("NotificationActionController", () => {
	let notificationStore: NotificationStore;
	let paginationStore: PaginationStore;
	let detailStore: DetailStore;
	let selectionStore: SelectionStore;
	let keyboardStore: KeyboardStore;
	let uiStore: UIStore;
	let queryStore: QueryStore;
	let options: ControllerOptions;
	let optimisticUpdateHelpers: ReturnType<typeof createOptimisticUpdateHelpers>;
	let sharedHelpers: ReturnType<typeof createSharedHelpers>;
	let controller: ReturnType<typeof createNotificationActionController>;

	beforeEach(() => {
		vi.clearAllMocks();

		const mockNotification = createMockNotification({
			actionHints: { dismissedOn: ["archive"] },
		});

		notificationStore = {
			pageData: writable({
				items: [mockNotification],
				total: 1,
				page: 1,
				pageSize: 50,
			}),
			notificationsById: writable(new Map([["gh-1", mockNotification]])),
			updateNotification: vi.fn(),
			removeNotification: vi.fn(),
			restoreNotification: vi.fn(),
		} as any;

		paginationStore = {
			page: writable(1),
			isLoading: writable(false),
			total: writable(1),
			pageSize: writable(50),
			setTotal: vi.fn(),
		} as any;

		detailStore = {
			detailNotificationId: writable(null),
			detailOpen: writable(false),
		} as any;

		selectionStore = {
			selectedIds: writable(new Set()),
			toggleSelection: vi.fn(),
		} as any;

		keyboardStore = {
			keyboardFocusIndex: writable(0),
		} as any;

		uiStore = {
			splitModeEnabled: writable(false),
		} as any;

		queryStore = {
			quickQuery: writable(""),
			viewQuery: writable(""),
		} as any;

		options = {
			onRefresh: vi.fn(async () => {}),
			onRefreshViewCounts: vi.fn(async () => {}),
		};

		const debounceManager = createDebounceManager();
		sharedHelpers = createSharedHelpers(
			notificationStore,
			paginationStore,
			queryStore,
			options,
			debounceManager
		);

		optimisticUpdateHelpers = createOptimisticUpdateHelpers(
			{
				notificationStore,
				paginationStore,
				detailStore,
				selectionStore,
				keyboardStore,
				uiStore,
			},
			options,
			sharedHelpers
		);

		controller = createNotificationActionController(
			{
				notificationStore,
				paginationStore,
				detailStore,
				selectionStore,
				keyboardStore,
			},
			options,
			optimisticUpdateHelpers,
			sharedHelpers
		);
	});

	describe("markRead", () => {
		it("marks unread notification as read", async () => {
			const notification = createMockNotification({ isRead: false });
			const updated = { ...notification, isRead: true };
			vi.mocked(markNotificationRead).mockResolvedValue(updated);

			await controller.markRead(notification);

			expect(markNotificationRead).toHaveBeenCalledWith("gh-1");
		});

		it("marks read notification as unread", async () => {
			const notification = createMockNotification({ isRead: true });
			const updated = { ...notification, isRead: false };
			vi.mocked(markNotificationUnread).mockResolvedValue(updated);

			await controller.markRead(notification);

			expect(markNotificationUnread).toHaveBeenCalledWith("gh-1");
		});
	});

	describe("archive", () => {
		it("archives unarchived notification", async () => {
			const notification = createMockNotification({ archived: false });
			const updated = { ...notification, archived: true };
			vi.mocked(archiveNotification).mockResolvedValue(updated);

			await controller.archive(notification);

			expect(archiveNotification).toHaveBeenCalledWith("gh-1");
			expect(toastStore.info).toHaveBeenCalledWith("Archived");
		});

		it("unarchives archived notification", async () => {
			const notification = createMockNotification({ archived: true });
			const updated = { ...notification, archived: false };
			vi.mocked(unarchiveNotification).mockResolvedValue(updated);

			await controller.archive(notification);

			expect(toastStore.info).toHaveBeenCalledWith("Unarchived");
		});
	});

	describe("error handling", () => {
		it("shows error toast when API call fails", async () => {
			const notification = createMockNotification({ isRead: false });
			const error = new Error("Network error");
			vi.mocked(markNotificationRead).mockRejectedValue(error);

			await controller.markRead(notification);

			expect(toastStore.error).toHaveBeenCalledWith("Failed to toggle read status");
			expect(options.onRefresh).toHaveBeenCalled();
		});

		it("refreshes data on error to recover", async () => {
			const notification = createMockNotification({ isRead: false });
			vi.mocked(markNotificationRead).mockRejectedValue(new Error("API error"));

			await controller.markRead(notification);

			expect(options.onRefresh).toHaveBeenCalled();
		});

		it("handles missing notification key gracefully", async () => {
			vi.clearAllMocks();
			const notification = {}; // No githubId or id
			await controller.markRead(notification as any);

			// Should not throw, just return early - no key means no action
			expect(markNotificationRead).not.toHaveBeenCalled();
			expect(markNotificationUnread).not.toHaveBeenCalled();
		});

		it("uses notification from store when available", async () => {
			const pageNotification = createMockNotification({ isRead: false });
			notificationStore.pageData.set({
				items: [pageNotification],
				total: 1,
				page: 1,
				pageSize: 50,
			});
			notificationStore.notificationsById.set(new Map([["gh-1", pageNotification]]));
			const updated = { ...pageNotification, isRead: true };
			vi.mocked(markNotificationRead).mockResolvedValue(updated);

			// Pass a different notification object
			const passedNotification = createMockNotification({ isRead: false });
			await controller.markRead(passedNotification);

			// Should use the one from store
			expect(markNotificationRead).toHaveBeenCalledWith("gh-1");
		});
	});

	describe("action hints", () => {
		it("determines if action will remove notification", async () => {
			const notification = createMockNotification({
				archived: false,
				actionHints: { dismissedOn: ["archive"] },
			});
			const updated = { ...notification, archived: true };
			vi.mocked(archiveNotification).mockResolvedValue(updated);

			await controller.archive(notification);

			// Should handle removal correctly via optimistic updates
			expect(archiveNotification).toHaveBeenCalled();
		});
	});
});
