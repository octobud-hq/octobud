<script lang="ts">
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

	import { setContext } from "svelte";
	import { writable } from "svelte/store";
	import NotificationViewHeader from "./NotificationViewHeader.svelte";
	import NotificationSingleMode from "./NotificationSingleMode.svelte";
	import NotificationSplitMode from "./NotificationSplitMode.svelte";
	import ApiErrorState from "$lib/components/shared/ApiErrorState.svelte";
	import type { Notification, NotificationDetail, Tag } from "$lib/api/types";
	import type { TimelineController } from "$lib/state/timelineController";
	import type {
		NotificationViewHeaderComponent,
		NotificationSingleModeComponent,
		NotificationSplitModeComponent,
	} from "$lib/types/components";
	import {
		NOTIFICATION_VIEW_CONTEXT_KEY,
		type NotificationViewContext,
	} from "$lib/types/notificationViewContext";

	// API error state
	export let apiError: string | null | undefined;
	export let apiErrorIsInline: boolean = false; // If true, show error inline instead of full screen
	export let tags: Tag[] = [];

	// View Header props
	export let combinedQuery: string;
	export let canUpdateView: boolean;
	export let onChangeQuery: (value: string) => void;
	export let onSubmitQuery: (value: string) => void;
	export let onClear: () => void;
	export let onUpdateView: () => void;
	export let onSaveAsNewView: () => void;
	export let page: number;
	export let totalPages: number;
	export let onPrevious: () => void;
	export let onNext: () => void;
	export let multiselectMode: boolean;
	export let onToggleMultiselect: () => void;
	export let splitModeEnabled: boolean;
	export let onToggleSplitMode: () => void;

	// List view props
	export let hasActiveFilters: boolean;
	export let pageRangeStart: number;
	export let pageRangeEnd: number;
	export let items: Notification[];
	export let isLoading: boolean;
	export let selectionEnabled: boolean;
	export let individualSelectionDisabled: boolean;
	export let detailNotificationId: string | null;
	export let selectionMap: Map<string, boolean>;
	export let totalCount: number;

	// Detail view props
	export let detailOpen: boolean;
	export let currentDetailNotification: Notification | null;
	export let currentDetail: NotificationDetail | null;
	export let detailLoading: boolean;
	export let detailShowingStaleData: boolean;
	export let detailIsRefreshing: boolean;
	export let timelineController: TimelineController | undefined;

	// Split view specific props
	export let listPaneWidth: number = 40;
	export let onPaneResize: (event: CustomEvent<{ deltaX: number }>) => void;

	// Scroll restoration (SingleMode only)
	export let initialScrollPosition: number = 0;

	// Create context store - use writable store so child components can reactively subscribe
	const contextStore = writable<NotificationViewContext>({
		listView: {
			hasActiveFilters,
			totalCount,
			pageRangeStart,
			pageRangeEnd,
			items,
			isLoading,
			selectionEnabled,
			individualSelectionDisabled,
			detailNotificationId,
			selectionMap,
		},
		detailView: {
			detailOpen,
			currentDetailNotification,
			currentDetail,
			detailLoading,
			detailShowingStaleData,
			detailIsRefreshing,
			timelineController,
		},
		splitView: {
			listPaneWidth,
			onPaneResize,
		},
		initialScrollPosition,
		apiError: apiErrorIsInline ? apiError : null,
	});

	// Update context store reactively when props change
	$: contextStore.set({
		listView: {
			hasActiveFilters,
			totalCount,
			pageRangeStart,
			pageRangeEnd,
			items,
			isLoading,
			selectionEnabled,
			individualSelectionDisabled,
			detailNotificationId,
			selectionMap,
		},
		detailView: {
			detailOpen,
			currentDetailNotification,
			currentDetail,
			detailLoading,
			detailShowingStaleData,
			detailIsRefreshing,
			timelineController,
		},
		splitView: {
			listPaneWidth,
			onPaneResize,
		},
		initialScrollPosition,
		apiError: apiErrorIsInline ? apiError : null,
	});

	// Set context store so child components can subscribe to it
	setContext(NOTIFICATION_VIEW_CONTEXT_KEY, contextStore);

	// Export refs for parent access
	let headerComponent: NotificationViewHeaderComponent | null = null;
	let singleModeComponent: NotificationSingleModeComponent | null = null;
	let splitModeComponent: NotificationSplitModeComponent | null = null;

	export function focusSearchInput() {
		return headerComponent?.focusSearchInput() ?? false;
	}

	export function toggleFilterDropdown() {
		// This is a placeholder - implement if needed
	}

	export function isFilterDropdownOpen() {
		return false;
	}

	export function openBulkSnoozeDropdown() {
		return headerComponent?.openBulkSnoozeDropdown() ?? false;
	}

	export function closeBulkSnoozeDropdown() {
		return headerComponent?.closeBulkSnoozeDropdown() ?? false;
	}

	export function getBulkSnoozeDropdownOpen(): boolean {
		return headerComponent?.getBulkSnoozeDropdownOpen() ?? false;
	}

	export function openBulkTagDropdown() {
		return headerComponent?.openBulkTagDropdown() ?? false;
	}

	export function closeBulkTagDropdown() {
		return headerComponent?.closeBulkTagDropdown() ?? false;
	}

	export function getBulkTagDropdownOpen(): boolean {
		return headerComponent?.getBulkTagDropdownOpen() ?? false;
	}

	// Scroll position management
	export function saveScrollPosition() {
		const listSection = splitModeEnabled
			? splitModeComponent?.getListSectionComponent()
			: singleModeComponent?.getListSectionComponent();
		return listSection?.saveScrollPosition() ?? 0;
	}

	export function getScrollPosition() {
		const listSection = splitModeEnabled
			? splitModeComponent?.getListSectionComponent()
			: singleModeComponent?.getListSectionComponent();
		return listSection?.getScrollPosition() ?? 0;
	}

	export function setScrollPosition(scrollTop: number) {
		const listSection = splitModeEnabled
			? splitModeComponent?.getListSectionComponent()
			: singleModeComponent?.getListSectionComponent();
		listSection?.setScrollPosition(scrollTop);
	}
</script>

{#if apiError && !apiErrorIsInline}
	<ApiErrorState errorMessage={apiError} />
{:else}
	<section class="flex min-h-0 flex-1 flex-col gap-5 overflow-hidden">
		<!-- NotificationViewHeader - always at top, but hidden when detail is open in list mode -->
		{#if splitModeEnabled || !detailOpen}
			<NotificationViewHeader
				bind:this={headerComponent}
				{combinedQuery}
				{canUpdateView}
				{onChangeQuery}
				{onSubmitQuery}
				{onClear}
				{onUpdateView}
				{onSaveAsNewView}
				{page}
				{totalPages}
				{onPrevious}
				{onNext}
				{multiselectMode}
				{onToggleMultiselect}
				{splitModeEnabled}
				{onToggleSplitMode}
				{tags}
			/>
		{/if}

		<!-- Content area: reading pane OR list OR detail -->
		{#if splitModeEnabled}
			<NotificationSplitMode bind:this={splitModeComponent} />
		{:else}
			<NotificationSingleMode bind:this={singleModeComponent} />
		{/if}
	</section>
{/if}
