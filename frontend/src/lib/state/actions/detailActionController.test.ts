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
import { writable, get, derived } from "svelte/store";
import { createDetailActionController } from "./detailActionController";
import { createSharedHelpers } from "./sharedHelpers";
import { createDebounceManager } from "./debounceManager";
import type { DetailStore } from "../../stores/detailViewStore";
import type { KeyboardStore } from "../../stores/keyboardNavigationStore";
import type { NotificationStore } from "../../stores/notificationStore";
import type { PaginationStore } from "../../stores/paginationStore";
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

describe("DetailActionController", () => {
	let detailStore: DetailStore;
	let keyboardStore: KeyboardStore;
	let notificationStore: NotificationStore;
	let paginationStore: PaginationStore;
	let uiStore: UIStore;
	let queryStore: QueryStore;
	let options: ControllerOptions;
	let sharedHelpers: ReturnType<typeof createSharedHelpers>;
	let markReadAction: (notification: any) => Promise<void>;
	let controller: ReturnType<typeof createDetailActionController>;

	beforeEach(() => {
		// Mock window
		global.window = {
			location: {
				href: "http://localhost:3000/views/inbox",
			},
		} as any;

		const detailNotificationId = writable<string | null>(null);
		detailStore = {
			detailNotificationId,
			detailNotificationIndex: writable(null),
			detailOpen: derived(detailNotificationId, ($id) => $id !== null),
			openDetail: vi.fn((id: string) => {
				detailNotificationId.set(id);
			}),
			closeDetail: vi.fn(() => {
				detailNotificationId.set(null);
			}),
		} as any;

		keyboardStore = {
			keyboardFocusIndex: writable(5),
			setFocusIndex: vi.fn((index: number) => {
				keyboardStore.keyboardFocusIndex.set(index);
			}),
			focusAt: vi.fn((index: number) => {
				keyboardStore.keyboardFocusIndex.set(index);
				return true;
			}),
			resetFocus: vi.fn(() => {
				keyboardStore.keyboardFocusIndex.set(null);
				return true;
			}),
		} as any;

		const mockNotification = createMockNotification();
		const notificationsMap = new Map();
		notificationsMap.set("gh-1", mockNotification);
		notificationStore = {
			pageData: writable({
				items: [mockNotification],
				total: 1,
				page: 1,
				pageSize: 50,
			}),
			notificationsById: writable(notificationsMap),
			updateNotification: vi.fn(),
		} as any;

		paginationStore = {
			page: writable(1),
			total: writable(1),
			pageSize: writable(50),
		} as any;

		uiStore = {
			sidebarCollapsed: writable(false),
			splitModeEnabled: writable(false),
			savedListScrollPosition: writable(0),
		} as any;

		queryStore = {
			quickQuery: writable(""),
			viewQuery: writable(""),
		} as any;

		options = {
			navigateToUrl: vi.fn(async (url: string, opts?: any) => {}),
		};

		const debounceManager = createDebounceManager();
		sharedHelpers = createSharedHelpers(
			notificationStore,
			paginationStore,
			queryStore,
			options,
			debounceManager
		);

		markReadAction = vi.fn(async (notification: any) => {});

		controller = createDetailActionController(
			detailStore,
			keyboardStore,
			notificationStore,
			paginationStore,
			uiStore,
			options,
			sharedHelpers,
			markReadAction
		);
	});

	describe("openDetail", () => {
		it("opens detail with notification ID", () => {
			controller.openDetail("gh-1");
			expect(detailStore.openDetail).toHaveBeenCalledWith("gh-1");
			expect(get(detailStore.detailNotificationId)).toBe("gh-1");
		});
	});

	describe("closeDetail", () => {
		it("closes detail", () => {
			detailStore.detailNotificationId.set("gh-1");
			controller.closeDetail();
			expect(detailStore.closeDetail).toHaveBeenCalled();
			expect(get(detailStore.detailNotificationId)).toBeNull();
		});
	});

	describe("handleOpenInlineDetail", () => {
		it("opens detail and sets focus index", async () => {
			const notification = createMockNotification();
			await controller.handleOpenInlineDetail(notification);

			expect(detailStore.openDetail).toHaveBeenCalledWith("gh-1");
			expect(keyboardStore.setFocusIndex).toHaveBeenCalledWith(0);
		});

		it("does not mark notification as read in handleOpenInlineDetail (now handled when detail loads)", async () => {
			const notification = createMockNotification({ isRead: false });
			const splitMode = (uiStore as any).splitModeEnabled;
			splitMode.set(false);

			// Ensure notification is in the notificationsById map
			const notificationsMap = get(notificationStore.notificationsById);
			notificationsMap.set("gh-1", notification);
			notificationStore.notificationsById.set(notificationsMap);

			await controller.handleOpenInlineDetail(notification);

			// Mark-as-read is now handled when detail loads, not in handleOpenInlineDetail
			expect(markReadAction).not.toHaveBeenCalled();
		});

		it("does not mark as read in split mode", async () => {
			const notification = createMockNotification({ isRead: false });
			const splitMode = (uiStore as any).splitModeEnabled;
			splitMode.set(true);
			await controller.handleOpenInlineDetail(notification);

			expect(markReadAction).not.toHaveBeenCalled();
		});

		it("does not mark as read if already read", async () => {
			const notification = createMockNotification({ isRead: true });
			const splitMode = (uiStore as any).splitModeEnabled;
			splitMode.set(false);
			await controller.handleOpenInlineDetail(notification);

			expect(markReadAction).not.toHaveBeenCalled();
		});
	});

	describe("handleCloseInlineDetail", () => {
		it("closes detail", async () => {
			detailStore.detailNotificationId.set("gh-1");
			await controller.handleCloseInlineDetail();

			expect(detailStore.closeDetail).toHaveBeenCalled();
		});

		it("resets focus when clearFocus is true", async () => {
			detailStore.detailNotificationId.set("gh-1");
			await controller.handleCloseInlineDetail({ clearFocus: true });

			expect(keyboardStore.resetFocus).toHaveBeenCalled();
		});
	});
});
