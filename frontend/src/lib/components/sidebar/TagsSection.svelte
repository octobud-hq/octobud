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
	import TagCard from "./TagCard.svelte";
	import type { Tag } from "$lib/api/tags";

	export let sortedTags: Tag[];
	export let tags: Tag[];
	export let selectedTagId: string = "";
	export let tagsCollapsed: boolean = false;
	export let tagReorderMode: boolean = false;
	export let tagReordering: boolean = false;
	export let draggedOverTagId: string | null = null;

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

<!-- Tags Section -->
<button
	type="button"
	class="flex w-full min-w-0 flex-nowrap items-center justify-between gap-1 overflow-hidden rounded-lg px-2 py-3 transition hover:bg-gray-200 dark:hover:bg-gray-800/50 cursor-pointer"
	on:click={onToggleTagsCollapsed}
	aria-label={tagsCollapsed ? "Expand tags" : "Collapse tags"}
>
	<h2
		class="min-w-0 flex-shrink text-xs font-semibold tracking-wide text-gray-600 dark:text-gray-400 whitespace-nowrap"
	>
		Tags
	</h2>
	<svg
		class="h-[15px] w-[15px] text-gray-600 dark:text-gray-400 transition-transform flex-shrink-0 dropdown-caret {!tagsCollapsed
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
{#if !tagsCollapsed}
	{#if tags.length === 0}
		<div class="flex flex-col items-center justify-center gap-2 py-8 text-center">
			<p class="font-semibold text-sm text-gray-700 dark:text-gray-200">No tags created yet</p>
			<p class="text-xs text-gray-600 dark:text-gray-400">
				Tags help you organize your notifications.
			</p>
		</div>
	{:else}
		<ul class="space-y-0 pt-1">
			{#each sortedTags as tag (tag.id)}
				<TagCard
					{tag}
					selected={selectedTagId === tag.id}
					onSelect={onSelectTag}
					showActions={true}
					onEdit={onEditTag}
					reorderMode={tagReorderMode}
					onDragStart={onTagDragStart}
					onDragOver={onTagDragOver}
					onDrop={onTagDrop}
					onDragEnd={onTagDragEnd}
					isDraggedOver={draggedOverTagId === tag.id}
				/>
			{/each}
		</ul>
	{/if}
	<div class="mt-1 pl-1 flex items-center justify-between">
		{#if tags.length > 1}
			<button
				type="button"
				class="flex-shrink-0 whitespace-nowrap rounded-full px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 cursor-pointer"
				on:click={onToggleTagReorderMode}
				disabled={tagReordering}
			>
				{tagReorderMode ? (tagReordering ? "Saving..." : "Done") : "Reorder"}
			</button>
		{:else}
			<div></div>
		{/if}
		<button
			type="button"
			class="mr-2 flex-shrink-0 whitespace-nowrap rounded-full px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 cursor-pointer"
			on:click={onNewTag}
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
