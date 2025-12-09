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
import { createViewStore } from "./viewStore";
import type { NotificationView } from "$lib/api/types";
import { BUILT_IN_VIEWS } from "$lib/state/types";

describe("ViewStore", () => {
	const createMockView = (overrides: Partial<NotificationView> = {}): NotificationView => ({
		id: "1",
		slug: "test-view",
		name: "Test View",
		description: "A test view",
		icon: "inbox",
		isDefault: false,
		query: "type:PullRequest",
		unreadCount: 0,
		...overrides,
	});

	describe("initialization", () => {
		it("initializes with provided views and view id", () => {
			const views = [createMockView()];
			const store = createViewStore(views, "view-1");

			expect(get(store.views)).toEqual(views);
			expect(get(store.selectedViewId)).toBe("view-1");
		});

		it("initializes with empty views", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);

			expect(get(store.views)).toEqual([]);
		});
	});

	describe("built-in views", () => {
		it("includes all built-in views", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);
			const builtInViews = get(store.builtInViewList);

			expect(builtInViews).toHaveLength(5);
			expect(builtInViews.map((v) => v.id)).toEqual([
				BUILT_IN_VIEWS.inbox.id,
				BUILT_IN_VIEWS.starred.id,
				BUILT_IN_VIEWS.snoozed.id,
				BUILT_IN_VIEWS.archive.id,
				BUILT_IN_VIEWS.everything.id,
			]);
		});

		it("inbox is default when no custom views have isDefault", () => {
			const views = [createMockView({ isDefault: false })];
			const store = createViewStore(views, BUILT_IN_VIEWS.inbox.id);
			const builtInViews = get(store.builtInViewList);
			const inbox = builtInViews.find((v) => v.id === BUILT_IN_VIEWS.inbox.id);

			expect(inbox?.isDefault).toBe(true);
		});

		it("inbox is not default when a custom view has isDefault", () => {
			const views = [createMockView({ isDefault: true })];
			const store = createViewStore(views, BUILT_IN_VIEWS.inbox.id);
			const builtInViews = get(store.builtInViewList);
			const inbox = builtInViews.find((v) => v.id === BUILT_IN_VIEWS.inbox.id);

			expect(inbox?.isDefault).toBe(false);
		});
	});

	describe("default view", () => {
		it("returns inbox as default when no custom default exists", () => {
			const views = [createMockView({ isDefault: false })];
			const store = createViewStore(views, BUILT_IN_VIEWS.inbox.id);

			expect(get(store.defaultViewId)).toBe(BUILT_IN_VIEWS.inbox.id);
			expect(get(store.defaultViewSlug)).toBe(BUILT_IN_VIEWS.inbox.slug);
			expect(get(store.defaultViewDisplayName)).toBe(BUILT_IN_VIEWS.inbox.name);
		});

		it("returns custom view as default when isDefault is true", () => {
			const customDefault = createMockView({
				id: "custom-1",
				slug: "custom-default",
				name: "My Default",
				isDefault: true,
			});
			const store = createViewStore([customDefault], BUILT_IN_VIEWS.inbox.id);

			expect(get(store.defaultViewId)).toBe("custom-1");
			expect(get(store.defaultViewSlug)).toBe("custom-default");
			expect(get(store.defaultViewDisplayName)).toBe("My Default");
		});
	});

	describe("selectedView", () => {
		it("returns null for built-in views", () => {
			const views = [createMockView({ id: "custom-1" })];
			const store = createViewStore(views, BUILT_IN_VIEWS.inbox.id);

			expect(get(store.selectedView)).toBeNull();
		});

		it("returns the custom view when selected", () => {
			const customView = createMockView({ id: "custom-1", name: "Custom" });
			const views = [customView];
			const store = createViewStore(views, "custom-1");

			expect(get(store.selectedView)).toEqual(customView);
		});

		it("returns null for non-existent custom view", () => {
			const store = createViewStore([], "non-existent");

			expect(get(store.selectedView)).toBeNull();
		});
	});

	describe("selectedViewSlug", () => {
		it("maps inbox ID to slug", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);
			expect(get(store.selectedViewSlug)).toBe(BUILT_IN_VIEWS.inbox.slug);
		});

		it("maps starred ID to slug", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.starred.id);
			expect(get(store.selectedViewSlug)).toBe(BUILT_IN_VIEWS.starred.slug);
		});

		it("maps archive ID to slug", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.archive.id);
			expect(get(store.selectedViewSlug)).toBe(BUILT_IN_VIEWS.archive.slug);
		});

		it("maps snoozed ID to slug", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.snoozed.id);
			expect(get(store.selectedViewSlug)).toBe(BUILT_IN_VIEWS.snoozed.slug);
		});

		it("maps everything ID to slug", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.everything.id);
			expect(get(store.selectedViewSlug)).toBe(BUILT_IN_VIEWS.everything.slug);
		});

		it("returns custom view slug", () => {
			const customView = createMockView({ id: "custom-1", slug: "my-custom-view" });
			const store = createViewStore([customView], "custom-1");

			expect(get(store.selectedViewSlug)).toBe("my-custom-view");
		});
	});

	describe("built-in view flags", () => {
		it("isBuiltInArchiveView is true for archive", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.archive.id);
			expect(get(store.isBuiltInArchiveView)).toBe(true);
			expect(get(store.isBuiltInSnoozedView)).toBe(false);
			expect(get(store.isBuiltInEverythingView)).toBe(false);
		});

		it("isBuiltInSnoozedView is true for snoozed", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.snoozed.id);
			expect(get(store.isBuiltInArchiveView)).toBe(false);
			expect(get(store.isBuiltInSnoozedView)).toBe(true);
			expect(get(store.isBuiltInEverythingView)).toBe(false);
		});

		it("isBuiltInEverythingView is true for everything", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.everything.id);
			expect(get(store.isBuiltInArchiveView)).toBe(false);
			expect(get(store.isBuiltInSnoozedView)).toBe(false);
			expect(get(store.isBuiltInEverythingView)).toBe(true);
		});

		it("all flags are false for inbox", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);
			expect(get(store.isBuiltInArchiveView)).toBe(false);
			expect(get(store.isBuiltInSnoozedView)).toBe(false);
			expect(get(store.isBuiltInEverythingView)).toBe(false);
		});
	});

	describe("actions", () => {
		it("setViews updates the views", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);

			const newViews = [createMockView({ id: "new-1" })];
			store.setViews(newViews);

			expect(get(store.views)).toEqual(newViews);
		});

		it("selectView updates selectedViewId", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);

			store.selectView(BUILT_IN_VIEWS.archive.id);

			expect(get(store.selectedViewId)).toBe(BUILT_IN_VIEWS.archive.id);
		});
	});

	describe("reactive updates", () => {
		it("default view updates when views change", () => {
			const store = createViewStore([], BUILT_IN_VIEWS.inbox.id);

			expect(get(store.defaultViewId)).toBe(BUILT_IN_VIEWS.inbox.id);

			// Add a custom default view
			const customDefault = createMockView({
				id: "custom-1",
				slug: "custom",
				isDefault: true,
			});
			store.setViews([customDefault]);

			expect(get(store.defaultViewId)).toBe("custom-1");
		});

		it("selectedView updates when views change", () => {
			const customView = createMockView({ id: "custom-1", name: "Original" });
			const store = createViewStore([customView], "custom-1");

			expect(get(store.selectedView)?.name).toBe("Original");

			// Update the view
			const updatedView = { ...customView, name: "Updated" };
			store.setViews([updatedView]);

			expect(get(store.selectedView)?.name).toBe("Updated");
		});
	});
});
