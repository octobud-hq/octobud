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
	$: initialScrollPosition = context.initialScrollPosition;
	$: apiError = context.apiError;

	// Export list section ref for parent access
	let listSectionComponent: NotificationListSectionComponent | null = null;
	export { listSectionComponent };

	// Export getter function for TypeScript type checking
	export function getListSectionComponent(): NotificationListSectionComponent | null {
		return listSectionComponent;
	}
</script>

{#if detailOpen && currentDetailNotification}
	<!-- Gmail-style inline detail view -->
	<InlineDetailView
		detail={currentDetail}
		loading={detailLoading}
		showingStaleData={detailShowingStaleData}
		isRefreshing={detailIsRefreshing}
		{hasPermissionError}
		isSplitView={false}
		{timelineController}
		markingRead={false}
		archiving={false}
	/>
{:else}
	<!-- List view -->
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
		isSplitView={false}
		{selectionMap}
		{initialScrollPosition}
		{apiError}
	/>
{/if}
