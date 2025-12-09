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
import { createSelectionStore } from "./selectionStore";
import { createNotificationStore } from "./notificationStore";
import type { Notification, NotificationPage } from "$lib/api/types";

describe("SelectionStore", () => {
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
		labels: [],
		viewIds: [],
		tags: [],
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

	const createStores = (notifications: Notification[] = []) => {
		const notificationStore = createNotificationStore(
			createMockPageData(notifications, { total: notifications.length })
		);
		const selectionStore = createSelectionStore(notificationStore);
		return { notificationStore, selectionStore };
	};

	describe("initialization", () => {
		it("initializes with empty selection", () => {
			const { selectionStore } = createStores();

			expect(get(selectionStore.selectedIds).size).toBe(0);
			expect(get(selectionStore.multiselectMode)).toBe(false);
			expect(get(selectionStore.selectAllMode)).toBe("none");
		});
	});

	describe("toggleSelection", () => {
		it("adds id to selection", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleSelection("gh-1");

			expect(get(selectionStore.selectedIds).has("gh-1")).toBe(true);
		});

		it("removes id from selection if already selected", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleSelection("gh-1");
			selectionStore.toggleSelection("gh-1");

			expect(get(selectionStore.selectedIds).has("gh-1")).toBe(false);
		});

		it("can select multiple items", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleSelection("gh-1");
			selectionStore.toggleSelection("gh-2");

			expect(get(selectionStore.selectedIds).size).toBe(2);
		});
	});

	describe("clearSelection", () => {
		it("clears all selected ids", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleSelection("gh-1");
			selectionStore.toggleSelection("gh-2");
			selectionStore.clearSelection();

			expect(get(selectionStore.selectedIds).size).toBe(0);
		});

		it("resets selectAllMode to none", () => {
			const notifications = [createMockNotification({ id: "1", githubId: "gh-1" })];
			const { selectionStore } = createStores(notifications);

			selectionStore.selectAllPage();
			selectionStore.clearSelection();

			expect(get(selectionStore.selectAllMode)).toBe("none");
		});
	});

	describe("selectAllPage", () => {
		it("selects all items on current page", () => {
			const notifications = [
				createMockNotification({ id: "1", githubId: "gh-1" }),
				createMockNotification({ id: "2", githubId: "gh-2" }),
				createMockNotification({ id: "3", githubId: "gh-3" }),
			];
			const { selectionStore } = createStores(notifications);

			selectionStore.selectAllPage();

			expect(get(selectionStore.selectedIds).size).toBe(3);
			expect(get(selectionStore.selectedIds).has("gh-1")).toBe(true);
			expect(get(selectionStore.selectedIds).has("gh-2")).toBe(true);
			expect(get(selectionStore.selectedIds).has("gh-3")).toBe(true);
			expect(get(selectionStore.selectAllMode)).toBe("page");
		});
	});

	describe("selectAllAcrossPages", () => {
		it("sets selectAllMode to all and clears individual selections", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleSelection("gh-1");
			selectionStore.selectAllAcrossPages();

			expect(get(selectionStore.selectedIds).size).toBe(0);
			expect(get(selectionStore.selectAllMode)).toBe("all");
		});
	});

	describe("clearSelectAllMode", () => {
		it("resets selectAllMode and clears selection", () => {
			const { selectionStore } = createStores();

			selectionStore.selectAllAcrossPages();
			selectionStore.clearSelectAllMode();

			expect(get(selectionStore.selectAllMode)).toBe("none");
			expect(get(selectionStore.selectedIds).size).toBe(0);
		});
	});

	describe("toggleMultiselectMode", () => {
		it("toggles multiselect mode on", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleMultiselectMode();

			expect(get(selectionStore.multiselectMode)).toBe(true);
		});

		it("toggles multiselect mode off and resets selectAllMode", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleMultiselectMode(); // on
			selectionStore.selectAllAcrossPages();
			selectionStore.toggleMultiselectMode(); // off

			expect(get(selectionStore.multiselectMode)).toBe(false);
			expect(get(selectionStore.selectAllMode)).toBe("none");
		});
	});

	describe("exitMultiselectMode", () => {
		it("exits multiselect mode and clears all selection state", () => {
			const { selectionStore } = createStores();

			selectionStore.toggleMultiselectMode();
			selectionStore.toggleSelection("gh-1");
			selectionStore.exitMultiselectMode();

			expect(get(selectionStore.multiselectMode)).toBe(false);
			expect(get(selectionStore.selectedIds).size).toBe(0);
			expect(get(selectionStore.selectAllMode)).toBe("none");
		});
	});

	describe("derived stores", () => {
		describe("selectionMap", () => {
			it("reflects individual selections", () => {
				const { selectionStore } = createStores();

				selectionStore.toggleSelection("gh-1");
				selectionStore.toggleSelection("gh-2");

				const map = get(selectionStore.selectionMap);
				expect(map.get("gh-1")).toBe(true);
				expect(map.get("gh-2")).toBe(true);
				expect(map.get("gh-3")).toBeUndefined();
			});

			it("reflects all page items when selectAllMode is all", () => {
				const notifications = [
					createMockNotification({ id: "1", githubId: "gh-1" }),
					createMockNotification({ id: "2", githubId: "gh-2" }),
				];
				const { selectionStore } = createStores(notifications);

				selectionStore.selectAllAcrossPages();

				const map = get(selectionStore.selectionMap);
				expect(map.get("gh-1")).toBe(true);
				expect(map.get("gh-2")).toBe(true);
			});
		});

		describe("selectedCount", () => {
			it("returns count of individual selections", () => {
				const { selectionStore } = createStores();

				selectionStore.toggleSelection("gh-1");
				selectionStore.toggleSelection("gh-2");

				expect(get(selectionStore.selectedCount)).toBe(2);
			});

			it("returns total when selectAllMode is all", () => {
				const notifications = [
					createMockNotification({ id: "1", githubId: "gh-1" }),
					createMockNotification({ id: "2", githubId: "gh-2" }),
				];
				const { notificationStore, selectionStore } = createStores(notifications);

				// Set a higher total to simulate more items across pages
				notificationStore.pageData.update((data) => ({ ...data, total: 100 }));
				selectionStore.selectAllAcrossPages();

				expect(get(selectionStore.selectedCount)).toBe(100);
			});
		});

		describe("individualSelectionDisabled", () => {
			it("is false when selectAllMode is not all", () => {
				const { selectionStore } = createStores();

				expect(get(selectionStore.individualSelectionDisabled)).toBe(false);

				selectionStore.selectAllPage();
				expect(get(selectionStore.individualSelectionDisabled)).toBe(false);
			});

			it("is true when selectAllMode is all", () => {
				const { selectionStore } = createStores();

				selectionStore.selectAllAcrossPages();

				expect(get(selectionStore.individualSelectionDisabled)).toBe(true);
			});
		});

		describe("selectionEnabled", () => {
			it("is false when multiselect mode is off", () => {
				const { selectionStore } = createStores();

				expect(get(selectionStore.selectionEnabled)).toBe(false);
			});

			it("is true when multiselect mode is on", () => {
				const { selectionStore } = createStores();

				selectionStore.toggleMultiselectMode();

				expect(get(selectionStore.selectionEnabled)).toBe(true);
			});
		});
	});
});
