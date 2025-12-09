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

	import { resolve } from "$app/paths";
	import ViewsSection from "./ViewsSection.svelte";
	import TagsSection from "./TagsSection.svelte";
	import { iconPaths } from "./sidebarIcons";
	import type { NotificationView } from "$lib/api/types";
	import type { Tag } from "$lib/api/tags";

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

	export let tags: Tag[] = [];
	export let sortedTags: Tag[] = [];
	export let selectedTagId: string = "";
	export let tagsCollapsed: boolean = false;
	export let tagReorderMode: boolean = false;
	export let tagReordering: boolean = false;
	export let draggedOverTagId: string | null = null;

	export let onSelectView: (slug: string) => void;
	export let onToggleViewsCollapsed: () => void;
	export let onNewView: () => void;
	export let onEditView: (view: NotificationView) => void;
	export let onToggleReorderMode: () => void;
	export let onDragStart: (viewId: string) => void;
	export let onDragOver: (e: DragEvent, viewId: string) => void;
	export let onDrop: (e: DragEvent, viewId: string) => void;
	export let onDragEnd: () => void;

	export let onSelectTag: (tagId: string) => void;
	export let onToggleTagsCollapsed: () => void;
	export let onNewTag: () => void;
	export let onEditTag: (tag: Tag) => void;
	export let onToggleTagReorderMode: () => void;
	export let onTagDragStart: (tagId: string) => void;
	export let onTagDragOver: (e: DragEvent, tagId: string) => void;
	export let onTagDrop: (e: DragEvent, tagId: string) => void;
	export let onTagDragEnd: () => void;
</script>

<div
	class="flex flex-1 flex-col gap-2 overflow-y-auto pr-2 pb-32 pt-4"
	style="scrollbar-gutter: stable;"
>
	<!-- Views + Tags wrapper -->
	<div class="flex flex-col">
		<ViewsSection
			{builtInViewList}
			{sortedUserViews}
			{userViews}
			{selectedViewId}
			{selectedViewSlug}
			{inboxView}
			{isSettingsRoute}
			{viewsCollapsed}
			{reorderMode}
			{reordering}
			{draggedOverViewId}
			{onSelectView}
			{onToggleViewsCollapsed}
			{onNewView}
			{onEditView}
			{onToggleReorderMode}
			{onDragStart}
			{onDragOver}
			{onDrop}
			{onDragEnd}
		/>

		<TagsSection
			{sortedTags}
			{tags}
			{selectedTagId}
			{tagsCollapsed}
			{tagReorderMode}
			{tagReordering}
			{draggedOverTagId}
			{onSelectTag}
			{onToggleTagsCollapsed}
			{onNewTag}
			{onEditTag}
			{onToggleTagReorderMode}
			{onTagDragStart}
			{onTagDragOver}
			{onTagDrop}
			{onTagDragEnd}
		/>
	</div>

	<div class="mx-3 mt-3 mb-2 border-b border-gray-200 dark:border-gray-800/90"></div>
	<a
		href={resolve("/settings")}
		class="flex items-center gap-3 px-3 py-2 text-sm rounded-md transition-colors {isSettingsRoute
			? 'bg-indigo-100 text-gray-900 dark:bg-gray-800 dark:text-gray-200'
			: 'text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-800/50'}"
	>
		<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			{#each iconPaths.settings as path (path)}
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={path} />
			{/each}
		</svg>
		<span>Settings</span>
	</a>
</div>

<style>
	:global(.dropdown-caret) {
		stroke-width: 1.5;
	}
	:global(.dark .dropdown-caret) {
		stroke-width: 2;
	}
</style>
