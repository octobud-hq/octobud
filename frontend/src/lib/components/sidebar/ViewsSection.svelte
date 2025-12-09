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
	// along with this program.  If not, see https://www.gnu.org/licenses/.
	import BuiltInViewCard from "./BuiltInViewCard.svelte";
	import ViewCard from "./ViewCard.svelte";
	import { normalizeViewId } from "$lib/state/types";
	import type { NotificationView } from "$lib/api/types";

	export let builtInViewList: NotificationView[];
	export let sortedUserViews: NotificationView[];
	export let userViews: NotificationView[];
	export let selectedViewId: string;
	export let selectedViewSlug: string;
	export let inboxView: NotificationView | undefined;
	export let isSettingsRoute: boolean;
	export let viewsCollapsed: boolean;
	export let reorderMode: boolean;
	export let reordering: boolean;
	export let draggedOverViewId: string | null;

	export let onSelectView: (slug: string) => void;
	export let onToggleViewsCollapsed: () => void;
	export let onNewView: () => void;
	export let onEditView: (view: NotificationView) => void;
	export let onToggleReorderMode: () => void;
	export let onDragStart: (viewId: string) => void;
	export let onDragOver: (e: DragEvent, viewId: string) => void;
	export let onDrop: (e: DragEvent, viewId: string) => void;
	export let onDragEnd: () => void;
</script>

<!-- Built-in views + Custom views -->
<ul class="space-y-0 py-2">
	{#each builtInViewList as builtIn (builtIn.id)}
		<BuiltInViewCard
			view={builtIn}
			selected={!isSettingsRoute &&
				(selectedViewSlug === builtIn.slug || selectedViewId === normalizeViewId(builtIn.id))}
			onSelect={onSelectView}
			icon={builtIn.icon as "inbox" | "star" | "infinity" | "check" | "archive" | "snooze"}
			count={builtIn.slug === "inbox" && inboxView ? inboxView.unreadCount : undefined}
		/>
	{/each}
</ul>
<button
	type="button"
	class="flex w-full min-w-0 flex-nowrap items-center justify-between gap-1 overflow-hidden rounded-lg px-2 py-3 transition hover:bg-gray-200 dark:hover:bg-gray-800/50 cursor-pointer"
	on:click={onToggleViewsCollapsed}
	aria-label={viewsCollapsed ? "Expand views" : "Collapse views"}
>
	<h2
		class="min-w-0 flex-shrink text-xs font-semibold tracking-wide text-gray-600 dark:text-gray-400 whitespace-nowrap"
	>
		Views
	</h2>
	<svg
		class="h-[15px] w-[15px] text-gray-600 dark:text-gray-400 transition-transform flex-shrink-0 dropdown-caret {!viewsCollapsed
			? 'scale-y-[-1]'
			: ''}"
		viewBox="0 0 12 12"
		fill="none"
		stroke="currentColor"
		stroke-width="1.5"
		stroke-linecap="round"
		stroke-linejoin="round"
		xmlns="http://www.w3.org/2000/svg"
	>
		<path d="M3 4.5L6 7.5L9 4.5" />
	</svg>
</button>
{#if !viewsCollapsed}
	{#if userViews.length === 0}
		<div class="flex flex-col items-center justify-center gap-2 py-8 text-center">
			<p class="font-semibold text-sm text-gray-700 dark:text-gray-200">No views created yet</p>
			<p class="text-xs text-gray-600 dark:text-gray-400">
				Views let you quickly revisit your favorite filters.
			</p>
		</div>
	{:else}
		<ul class="space-y-0 pt-1">
			{#each sortedUserViews as view (view.id)}
				<ViewCard
					{view}
					selected={!isSettingsRoute && selectedViewId === normalizeViewId(view.id)}
					onSelect={onSelectView}
					showActions={true}
					onEdit={onEditView}
					count={view.unreadCount}
					{reorderMode}
					{onDragStart}
					{onDragOver}
					{onDrop}
					{onDragEnd}
					isDraggedOver={draggedOverViewId === view.id}
				/>
			{/each}
		</ul>
	{/if}
	<div class="mt-1 pl-1 flex items-center justify-between">
		{#if userViews.length > 1}
			<button
				type="button"
				class="flex-shrink-0 whitespace-nowrap rounded-full px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 cursor-pointer"
				on:click={onToggleReorderMode}
				disabled={reordering}
			>
				{reorderMode ? (reordering ? "Saving..." : "Done") : "Reorder"}
			</button>
		{:else}
			<div></div>
		{/if}
		<button
			type="button"
			class="mr-2 flex-shrink-0 whitespace-nowrap rounded-full px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 cursor-pointer"
			on:click={onNewView}
		>
			Add
		</button>
	</div>
	<div class="pb-4"></div>
{/if}

<style>
	:global(.dropdown-caret) {
		stroke-width: 1.5;
	}
	:global(.dark .dropdown-caret) {
		stroke-width: 2;
	}
</style>
