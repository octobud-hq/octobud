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

import { writable, derived } from "svelte/store";
import type { Writable, Readable } from "svelte/store";
import type { NotificationView } from "$lib/api/types";
import { BUILT_IN_VIEWS, normalizeViewId, normalizeViewSlug } from "$lib/state/types";

/**
 * View Store
 * Manages view selection and view list
 */
export function createViewStore(initialViews: NotificationView[], initialViewId: string) {
	const views = writable<NotificationView[]>(initialViews);
	const selectedViewId = writable<string>(initialViewId);

	// Built-in view definitions (derived from BUILT_IN_VIEWS)
	const builtInStarredView: NotificationView = {
		id: BUILT_IN_VIEWS.starred.id,
		slug: BUILT_IN_VIEWS.starred.slug,
		name: BUILT_IN_VIEWS.starred.name,
		description: "",
		icon: BUILT_IN_VIEWS.starred.icon,
		isDefault: false,
		query: BUILT_IN_VIEWS.starred.query,
		unreadCount: 0,
	};

	const builtInArchiveView: NotificationView = {
		id: BUILT_IN_VIEWS.archive.id,
		slug: BUILT_IN_VIEWS.archive.slug,
		name: BUILT_IN_VIEWS.archive.name,
		description: "",
		icon: BUILT_IN_VIEWS.archive.icon,
		isDefault: false,
		query: BUILT_IN_VIEWS.archive.query,
		unreadCount: 0,
	};

	const builtInSnoozedView: NotificationView = {
		id: BUILT_IN_VIEWS.snoozed.id,
		slug: BUILT_IN_VIEWS.snoozed.slug,
		name: BUILT_IN_VIEWS.snoozed.name,
		description: "",
		icon: BUILT_IN_VIEWS.snoozed.icon,
		isDefault: false,
		query: BUILT_IN_VIEWS.snoozed.query,
		unreadCount: 0,
	};

	const builtInEverythingView: NotificationView = {
		id: BUILT_IN_VIEWS.everything.id,
		slug: BUILT_IN_VIEWS.everything.slug,
		name: BUILT_IN_VIEWS.everything.name,
		description: "",
		icon: BUILT_IN_VIEWS.everything.icon,
		isDefault: false,
		query: BUILT_IN_VIEWS.everything.query,
		unreadCount: 0,
	};

	// Derived stores
	const hasExplicitDefault = derived(views, ($views) => $views.some((view) => view.isDefault));

	const builtInInboxView = derived(hasExplicitDefault, ($hasExplicitDefault) => ({
		id: BUILT_IN_VIEWS.inbox.id,
		slug: BUILT_IN_VIEWS.inbox.slug,
		name: BUILT_IN_VIEWS.inbox.name,
		description: "",
		icon: BUILT_IN_VIEWS.inbox.icon,
		isDefault: !$hasExplicitDefault,
		query: BUILT_IN_VIEWS.inbox.query,
		unreadCount: 0,
	}));

	const builtInViewList = derived(builtInInboxView, ($builtInInboxView) => [
		$builtInInboxView,
		builtInStarredView,
		builtInSnoozedView,
		builtInArchiveView,
		builtInEverythingView,
	]);

	const defaultView = derived(views, ($views) => {
		const explicitDefault = $views.find((view) => view.isDefault);
		return explicitDefault ?? null;
	});

	const defaultViewId = derived(
		[hasExplicitDefault, defaultView],
		([$hasExplicitDefault, $defaultView]) =>
			normalizeViewId(
				$hasExplicitDefault && $defaultView ? $defaultView.id : BUILT_IN_VIEWS.inbox.id
			)
	);

	const defaultViewSlug = derived(
		[hasExplicitDefault, defaultView],
		([$hasExplicitDefault, $defaultView]) =>
			normalizeViewSlug(
				$hasExplicitDefault && $defaultView ? $defaultView.slug : BUILT_IN_VIEWS.inbox.slug
			)
	);

	const defaultViewDisplayName = derived(
		[hasExplicitDefault, defaultView],
		([$hasExplicitDefault, $defaultView]) =>
			$hasExplicitDefault && $defaultView
				? ($defaultView.name ?? "Default view")
				: BUILT_IN_VIEWS.inbox.name
	);

	const selectedView = derived([selectedViewId, views], ([$selectedViewId, $views]) => {
		// Check if selected view is a built-in view
		const isBuiltIn = Object.values(BUILT_IN_VIEWS).some(
			(view) => view.id === $selectedViewId || view.slug === $selectedViewId
		);
		if (isBuiltIn) {
			return null;
		}

		return $views.find((view) => normalizeViewId(view.id) === $selectedViewId) ?? null;
	});

	const selectedViewSlug = derived([selectedViewId, views], ([$selectedViewId, $views]) => {
		// Map built-in view IDs to their slugs using BUILT_IN_VIEWS
		for (const view of Object.values(BUILT_IN_VIEWS)) {
			if (view.id === $selectedViewId || view.slug === $selectedViewId) {
				return view.slug;
			}
		}

		// For custom views, look up the slug from the views array
		const customView = $views.find((v) => v.id === $selectedViewId);
		return customView?.slug ?? $selectedViewId;
	});

	const isBuiltInArchiveView = derived(
		selectedViewSlug,
		($selectedViewSlug) => $selectedViewSlug === BUILT_IN_VIEWS.archive.slug
	);

	const isBuiltInSnoozedView = derived(
		selectedViewSlug,
		($selectedViewSlug) => $selectedViewSlug === BUILT_IN_VIEWS.snoozed.slug
	);

	const isBuiltInEverythingView = derived(
		selectedViewSlug,
		($selectedViewSlug) => $selectedViewSlug === BUILT_IN_VIEWS.everything.slug
	);

	return {
		// Expose stores
		views: views as Writable<NotificationView[]>,
		selectedViewId: selectedViewId as Writable<string>,

		// Derived stores
		builtInViewList: builtInViewList as Readable<NotificationView[]>,
		defaultViewId: defaultViewId as Readable<string>,
		defaultViewSlug: defaultViewSlug as Readable<string>,
		defaultViewDisplayName: defaultViewDisplayName as Readable<string>,
		selectedView: selectedView as Readable<NotificationView | null>,
		selectedViewSlug: selectedViewSlug as Readable<string>,
		isBuiltInArchiveView: isBuiltInArchiveView as Readable<boolean>,
		isBuiltInSnoozedView: isBuiltInSnoozedView as Readable<boolean>,
		isBuiltInEverythingView: isBuiltInEverythingView as Readable<boolean>,

		// Actions
		setViews: (newViews: NotificationView[]) => {
			views.set(newViews);
		},
		selectView: (id: string) => {
			selectedViewId.set(id);
		},
	};
}

export type ViewStore = ReturnType<typeof createViewStore>;
