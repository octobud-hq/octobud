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

import { get, writable } from "svelte/store";
import type { Writable } from "svelte/store";
import { createView, updateView, deleteView, fetchViews } from "$lib/api/views";
import { createRule } from "$lib/api/rules";
import type { NotificationView, NotificationViewDraft } from "$lib/api/types";
import { BUILT_IN_VIEWS, normalizeViewId, normalizeViewSlug } from "$lib/state/types";
import { DEFAULT_VIEW_ICON, cleanIconInput, normalizeViewIcon } from "$lib/utils/viewIcons";
import { toastStore } from "$lib/stores/toastStore";

type NavigateToSlug = (slug: string, shouldInvalidate?: boolean) => Promise<void>;

interface ViewDialogControllerOptions {
	views: Writable<NotificationView[]>;
	setViews: (nextViews: NotificationView[]) => void;
	selectedViewId: Writable<string>;
	invalidateViews: () => Promise<void>;
	navigateToSlug: NavigateToSlug;
	refreshNotifications?: () => Promise<void>;
	quickQuery?: Writable<string>;
}

interface ViewDialogControllerStores {
	open: Writable<boolean>;
	saving: Writable<boolean>;
	editing: Writable<NotificationView | null>;
	confirmDeleteOpen: Writable<boolean>;
	linkedRulesConfirmOpen: Writable<boolean>;
	linkedRuleCount: Writable<number>;
	draft: Writable<NotificationViewDraft>;
	error: Writable<string | null>;
}

interface ViewDialogControllerActions {
	openNewDialog: () => void;
	openNewDialogWithQuery: (query: string) => void; // New: open with pre-filled query
	startEditing: (view: NotificationView) => void;
	startEditingWithQuery: (view: NotificationView, query: string) => void; // New: edit with custom query
	closeDialog: () => void;
	handleSave: (payload: {
		name: string;
		description: string;
		icon: string;
		query: string; // New: query string instead of filters array
		createAutoRule?: boolean;
		ruleConfig?: {
			name: string;
			description?: string;
			skipInbox: boolean;
			markRead: boolean;
			star: boolean;
			archive?: boolean;
			mute?: boolean;
			assignTags?: string[];
			enabled: boolean;
			applyToExisting?: boolean;
		};
	}) => Promise<void>;
	clearError: () => void;
	requestDelete: () => void;
	cancelDelete: () => void;
	confirmDelete: () => Promise<void>;
	confirmLinkedRulesDelete: () => Promise<void>;
	cancelLinkedRulesDelete: () => void;
	reset: () => void;
}

export interface ViewDialogController {
	stores: ViewDialogControllerStores;
	actions: ViewDialogControllerActions;
}

const createEmptyDraft = (): NotificationViewDraft => ({
	name: "",
	description: "",
	icon: DEFAULT_VIEW_ICON,
	query: "in:inbox,filtered", // Pre-populate with in:inbox,filtered for explicit query context (good for skip inbox rules)
});

export function createViewDialogController(
	options: ViewDialogControllerOptions
): ViewDialogController {
	const {
		views,
		setViews,
		selectedViewId,
		invalidateViews,
		navigateToSlug,
		refreshNotifications,
		quickQuery,
	} = options;

	const open = writable(false);
	const saving = writable(false);
	const editing = writable<NotificationView | null>(null);
	const confirmDeleteOpen = writable(false);
	const linkedRulesConfirmOpen = writable(false);
	const linkedRuleCount = writable(0);
	const draft = writable<NotificationViewDraft>(createEmptyDraft());
	const error = writable<string | null>(null);

	function resetDraft() {
		draft.set(createEmptyDraft());
	}

	function resetState() {
		editing.set(null);
		confirmDeleteOpen.set(false);
		linkedRulesConfirmOpen.set(false);
		linkedRuleCount.set(0);
		error.set(null);
		resetDraft();
	}

	function openNewDialog() {
		resetState();
		open.set(true);
	}

	function openNewDialogWithQuery(query: string) {
		resetState();
		draft.set({
			...createEmptyDraft(),
			query: query,
		});
		open.set(true);
	}

	function startEditing(view: NotificationView) {
		editing.set(view);
		draft.set({
			id: view.id,
			name: view.name,
			description: view.description ?? "",
			icon: normalizeViewIcon(view.icon),
			query: view.query || "", // New: use view's query string
		});
		confirmDeleteOpen.set(false);
		open.set(true);
	}

	function startEditingWithQuery(view: NotificationView, query: string) {
		editing.set(view);
		draft.set({
			id: view.id,
			name: view.name,
			description: view.description ?? "",
			icon: normalizeViewIcon(view.icon),
			query: query, // Use the provided query instead of view's original query
		});
		confirmDeleteOpen.set(false);
		open.set(true);
	}

	function closeDialog() {
		open.set(false);
		resetState();
	}

	async function handleSave(payload: {
		name: string;
		description: string;
		icon: string;
		query: string; // New: query string instead of filters array
		createAutoRule?: boolean;
		ruleConfig?: {
			name: string;
			description?: string;
			skipInbox: boolean;
			markRead: boolean;
			star: boolean;
			archive?: boolean;
			mute?: boolean;
			assignTags?: string[];
			enabled: boolean;
			applyToExisting?: boolean;
		};
	}) {
		saving.set(true);
		error.set(null); // Clear any previous errors
		try {
			const request = {
				name: payload.name,
				description: payload.description,
				icon: cleanIconInput(payload.icon) ?? DEFAULT_VIEW_ICON,
				query: payload.query, // New: send query string
			};

			const currentEditing = get(editing);
			let savedView: NotificationView;
			const isUpdate = !!currentEditing;

			if (currentEditing) {
				savedView = await updateView(currentEditing.id, request);
			} else {
				savedView = await createView(request);

				// Create auto-rule if requested (only for new views)
				if (payload.createAutoRule && payload.ruleConfig) {
					try {
						await createRule({
							name: payload.ruleConfig.name,
							description: payload.ruleConfig.description,
							viewId: savedView.id,
							// Don't include query - viewId provides the query
							actions: {
								skipInbox: payload.ruleConfig.skipInbox,
								markRead: payload.ruleConfig.markRead,
								star: payload.ruleConfig.star,
								archive: payload.ruleConfig.archive,
								mute: payload.ruleConfig.mute,
								assignTags: payload.ruleConfig.assignTags,
							},
							enabled: payload.ruleConfig.enabled,
							applyToExisting: payload.ruleConfig.applyToExisting,
						});
					} catch (ruleError) {
						// Don't fail view creation if rule creation fails
						toastStore.warning("View created, but auto-rule creation failed");
					}
				}
			}

			let refreshedViews: NotificationView[] | null = null;
			try {
				const updatedViews = await fetchViews();
				const ensuredViews = updatedViews.some((view) => view.id === savedView.id)
					? updatedViews
					: [...updatedViews, savedView];
				refreshedViews = ensuredViews;
				setViews(ensuredViews);
				await invalidateViews();
			} catch (error) {
				if (currentEditing) {
					views.update((existing) =>
						existing.map((item) => (item.id === savedView.id ? savedView : item))
					);
				} else {
					views.update((existing) => [...existing, savedView]);
				}
				await invalidateViews();
			}

			const targetView = refreshedViews?.find((view) => view.id === savedView.id) ?? savedView;

			// Reset draft on success (before closing dialog)
			resetDraft();
			open.set(false);
			resetState();

			const targetSlug = normalizeViewSlug(targetView.slug);
			await navigateToSlug(
				targetSlug.length > 0 ? targetSlug : BUILT_IN_VIEWS.inbox.slug,
				true // Invalidate to refresh views from API
			);

			// Refresh notifications after updating a view to reflect any query changes
			if (isUpdate && refreshNotifications) {
				// Update quickQuery to match the saved view's query before refreshing
				// This ensures the search input displays the correct query and notifications fetch with it
				if (quickQuery) {
					quickQuery.set(targetView.query || "");
				}
				await refreshNotifications();
			}

			// Show success toast after navigation
			setTimeout(() => {
				if (isUpdate) {
					toastStore.success("View updated successfully");
				} else {
					if (payload.createAutoRule) {
						toastStore.success("View and auto-rule created successfully");
					} else {
						toastStore.success("View created successfully");
					}
				}
			}, 0);
		} catch (err: any) {
			// Set error message - show all errors in the dialog, not in toasts
			if (err.statusCode === 409 || (err.message && err.message.includes("already exists"))) {
				error.set(err.message || "A view with this name already exists");
			} else {
				error.set(err.message || "An error occurred while saving the view");
			}
			// Don't throw - keep dialog open so user can see the error and fix it
			// Don't reset draft - preserve user's input
		} finally {
			saving.set(false);
			// Only reset draft on success (which happens in the try block before catch)
			// On error, we want to keep the user's input so they can fix it
		}
	}

	function requestDelete() {
		if (!get(editing)) return;
		confirmDeleteOpen.set(true);
	}

	function cancelDelete() {
		confirmDeleteOpen.set(false);
	}

	async function confirmDelete() {
		const currentEditing = get(editing);
		if (!currentEditing) return;
		saving.set(true);
		const deletedViewId = currentEditing.id;
		try {
			await deleteView(deletedViewId);
			try {
				const updatedViews = await fetchViews();
				setViews(updatedViews.filter((view) => view.id !== deletedViewId));
				await invalidateViews();
			} catch (error) {
				views.update((existing) => existing.filter((view) => view.id !== deletedViewId));
				await invalidateViews();
			}

			open.set(false);
			confirmDeleteOpen.set(false);

			const currentSelected = get(selectedViewId);
			if (currentSelected === normalizeViewId(deletedViewId)) {
				await navigateToSlug(BUILT_IN_VIEWS.inbox.slug);
			}

			// Show success toast after navigation
			setTimeout(() => {
				toastStore.success("View deleted");
			}, 0);
		} catch (error: any) {
			// Check if this is a linked rules conflict error
			if (error.statusCode === 409 && error.linkedRuleCount !== undefined) {
				const count = error.linkedRuleCount;
				// Show custom confirmation dialog instead of browser confirm
				linkedRuleCount.set(count);
				linkedRulesConfirmOpen.set(true);
				saving.set(false);
				// Don't reset state here - keep the delete confirmation open
				// The linked rules confirmation will handle the actual deletion
				return;
			} else {
				setTimeout(() => {
					toastStore.error("Failed to delete view");
				}, 0);
			}
		} finally {
			saving.set(false);
			// Only reset if we're not showing the linked rules confirmation
			if (!get(linkedRulesConfirmOpen)) {
				resetState();
			}
		}
	}

	async function confirmLinkedRulesDelete() {
		const currentEditing = get(editing);
		if (!currentEditing) return;
		saving.set(true);
		const deletedViewId = currentEditing.id;
		try {
			// User confirmed - try again with force=true
			// The backend will cascade delete due to FK constraint
			await deleteView(deletedViewId, true);
			try {
				const updatedViews = await fetchViews();
				setViews(updatedViews.filter((view) => view.id !== deletedViewId));
				await invalidateViews();
			} catch (fetchError) {
				views.update((existing) => existing.filter((view) => view.id !== deletedViewId));
				await invalidateViews();
			}

			open.set(false);
			confirmDeleteOpen.set(false);
			linkedRulesConfirmOpen.set(false);

			const currentSelected = get(selectedViewId);
			if (currentSelected === normalizeViewId(deletedViewId)) {
				await navigateToSlug(BUILT_IN_VIEWS.inbox.slug);
			}

			setTimeout(() => {
				toastStore.success("View and linked rules deleted");
			}, 0);
		} catch (retryError) {
			setTimeout(() => {
				toastStore.error("Failed to delete view");
			}, 0);
		} finally {
			saving.set(false);
			resetState();
		}
	}

	function cancelLinkedRulesDelete() {
		linkedRulesConfirmOpen.set(false);
		linkedRuleCount.set(0);
		// Keep the delete confirmation dialog open
	}

	function clearError() {
		error.set(null);
	}

	function reset() {
		open.set(false);
		saving.set(false);
		resetState();
	}

	return {
		stores: {
			open,
			saving,
			editing,
			confirmDeleteOpen,
			linkedRulesConfirmOpen,
			linkedRuleCount,
			draft,
			error,
		},
		actions: {
			openNewDialog,
			openNewDialogWithQuery,
			startEditing,
			startEditingWithQuery,
			closeDialog,
			handleSave,
			clearError,
			requestDelete,
			cancelDelete,
			confirmDelete,
			confirmLinkedRulesDelete,
			cancelLinkedRulesDelete,
			reset,
		},
	};
}
