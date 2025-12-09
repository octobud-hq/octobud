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

	import { getContext } from "svelte";
	import type { Writable } from "svelte/store";
	import NotificationListSection from "./NotificationListSection.svelte";
	import InlineDetailView from "$lib/components/detail/InlineDetailView.svelte";
	import ResizeHandle from "$lib/components/shared/ResizeHandle.svelte";
	import type { NotificationListSectionComponent } from "$lib/types/components";
	import {
		NOTIFICATION_VIEW_CONTEXT_KEY,
		type NotificationViewContext,
	} from "$lib/types/notificationViewContext";

	// Get context store from parent
	const contextStore = getContext<Writable<NotificationViewContext>>(NOTIFICATION_VIEW_CONTEXT_KEY);

	// Subscribe to context store and extract individual properties for reactivity
	$: context = $contextStore;
	$: detailOpen = context.detailView.detailOpen;
	$: currentDetailNotification = context.detailView.currentDetailNotification;
	$: currentDetail = context.detailView.currentDetail;
	$: detailLoading = context.detailView.detailLoading;
	$: detailShowingStaleData = context.detailView.detailShowingStaleData;
	$: detailIsRefreshing = context.detailView.detailIsRefreshing;
	$: hasPermissionError = context.detailView.hasPermissionError;
	$: timelineController = context.detailView.timelineController;
	$: hasActiveFilters = context.listView.hasActiveFilters;
	$: totalCount = context.listView.totalCount;
	$: pageRangeStart = context.listView.pageRangeStart;
	$: pageRangeEnd = context.listView.pageRangeEnd;
	$: items = context.listView.items;
	$: isLoading = context.listView.isLoading;
	$: selectionEnabled = context.listView.selectionEnabled;
	$: individualSelectionDisabled = context.listView.individualSelectionDisabled;
	$: detailNotificationId = context.listView.detailNotificationId;
	$: selectionMap = context.listView.selectionMap;
	$: listPaneWidth = context.splitView.listPaneWidth;
	$: onPaneResize = context.splitView.onPaneResize;
	$: apiError = context.apiError;

	// Export list section ref for parent access
	let listSectionComponent: NotificationListSectionComponent | null = null;
	export { listSectionComponent };

	// Export getter function for TypeScript type checking
	export function getListSectionComponent(): NotificationListSectionComponent | null {
		return listSectionComponent;
	}
</script>

<!-- Reading pane mode: Show list AND detail side-by-side -->
<div class="flex flex-1 min-h-0 overflow-hidden">
	<!-- Left: Notification List -->
	<div
		class="flex flex-col overflow-hidden border-r border-gray-200 dark:border-gray-800"
		style="width: {listPaneWidth}%"
	>
		<NotificationListSection
			bind:this={listSectionComponent}
			{hasActiveFilters}
			{totalCount}
			{pageRangeStart}
			{pageRangeEnd}
			{items}
			{isLoading}
			{selectionEnabled}
			{individualSelectionDisabled}
			{detailNotificationId}
			isSplitView={true}
			{selectionMap}
			{apiError}
		/>
	</div>

	<!-- Resize Handle -->
	<ResizeHandle on:resize={onPaneResize} />

	<!-- Right: Detail Pane -->
	<div class="flex flex-col overflow-hidden" style="width: {100 - listPaneWidth}%">
		{#if detailOpen && currentDetailNotification}
			<InlineDetailView
				detail={currentDetail}
				loading={detailLoading}
				showingStaleData={detailShowingStaleData}
				isRefreshing={detailIsRefreshing}
				{hasPermissionError}
				isSplitView={true}
				hideBackButton={true}
				{timelineController}
				markingRead={false}
				archiving={false}
			/>
		{:else}
			<!-- Empty state: Select a notification -->
			<div class="flex flex-1 items-center justify-center bg-white dark:bg-gray-950">
				<div class="flex flex-col items-center justify-center gap-3 text-center px-8">
					<p class="text-lg font-semibold text-gray-700 dark:text-gray-200">
						Select a notification
					</p>
					<p class="text-sm text-gray-600 dark:text-gray-400">
						Choose a notification from the list to view details.
					</p>
				</div>
			</div>
		{/if}
	</div>
</div>
