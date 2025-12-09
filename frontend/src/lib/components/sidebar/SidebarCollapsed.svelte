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
	import { normalizeViewId } from "$lib/state/types";
	import { normalizeViewIcon } from "$lib/utils/viewIcons";
	import { getBuiltInIconProps, iconPaths } from "./sidebarIcons";
	import type { NotificationView } from "$lib/api/types";
	import type { Tag } from "$lib/api/tags";

	export let builtInViewList: NotificationView[];
	export let sortedUserViews: NotificationView[];
	export let selectedViewId: string;
	export let selectedViewSlug: string;
	export let inboxView: NotificationView | undefined;
	export let isSettingsRoute: boolean;
	export let viewsCollapsed: boolean;

	export let sortedTags: Tag[] = [];
	export let selectedTagId: string = "";
	export let tagsCollapsed: boolean = false;

	export let onSelectView: (slug: string) => void;
	export let onToggleViewsCollapsed: () => void;
	export let onSelectTag: (tagId: string) => void;
	export let onToggleTagsCollapsed: () => void;

	// Tag icon (from label_outline.svg)
	const tagIconPath =
		"M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16zM16 17H5V7h11l3.55 5L16 17z";
</script>

<div class="flex flex-1 flex-col gap-2 overflow-y-auto pb-8 pt-6 scrollbar-hide">
	<!-- Views and Tags wrapper -->
	<div class="flex flex-col px-2">
		<!-- Built-in views -->
		<ul class="">
			{#each builtInViewList as builtIn (builtIn.id)}
				{@const iconProps = getBuiltInIconProps(builtIn.icon)}
				<li>
					<button
						type="button"
						class="relative flex h-9 w-9 items-center justify-center rounded-xl transition"
						on:click={() => onSelectView(builtIn.slug)}
						title={builtIn.name}
						aria-label={builtIn.name}
					>
						<span
							class="flex items-center justify-center rounded-xl p-2.5 transition text-gray-700 dark:text-gray-100 {!isSettingsRoute &&
							(selectedViewSlug === builtIn.slug || selectedViewId === normalizeViewId(builtIn.id))
								? 'bg-indigo-100 dark:bg-gray-800'
								: 'hover:bg-gray-200 dark:hover:bg-gray-800/50'}"
						>
							<svg
								class="h-[18px] w-[18px]"
								viewBox={iconProps.viewBox}
								fill={iconProps.fill}
								stroke={iconProps.stroke}
								stroke-width={iconProps.strokeWidth}
								stroke-linecap="round"
								stroke-linejoin="round"
								xmlns="http://www.w3.org/2000/svg"
								role="img"
								aria-hidden="true"
							>
								{#each iconProps.paths as path (path)}
									<path d={path} fill={builtIn.icon === "infinity" ? "currentColor" : undefined} />
								{/each}
							</svg>
						</span>
						{#if builtIn.slug === "inbox" && inboxView && inboxView.unreadCount > 0}
							<span
								class="absolute right-1 top-1 h-2 w-2 rounded-full bg-indigo-500"
								aria-label="{inboxView.unreadCount} unread"
							></span>
						{/if}
					</button>
				</li>
			{/each}
		</ul>
		<!-- Clickable header for collapsing/expanding views -->
		<button
			type="button"
			class="flex w-full items-center justify-center rounded-lg px-1 py-3 transition hover:bg-gray-200 dark:hover:bg-gray-800/50 cursor-pointer"
			on:click={onToggleViewsCollapsed}
			aria-label={viewsCollapsed ? "Expand views" : "Collapse views"}
			title={viewsCollapsed ? "Expand views" : "Collapse views"}
		>
			<svg
				class="h-[15px] w-[15px] text-gray-600 dark:text-gray-400 transition-transform dropdown-caret {!viewsCollapsed
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
			<ul class="space-y-1 pt-1">
				{#each sortedUserViews as view (view.id)}
					{@const displayIcon = normalizeViewIcon(view.icon)}
					<li>
						<button
							type="button"
							class="relative flex h-9 w-9 items-center justify-center rounded-xl transition"
							on:click={() => onSelectView(view.slug)}
							title={view.name}
							aria-label={view.name}
						>
							<span
								class="flex items-center justify-center rounded-xl py-1.5 px-2.5 text-base transition {!isSettingsRoute &&
								selectedViewId === normalizeViewId(view.id)
									? 'bg-indigo-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100'
									: 'text-gray-700 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-800/50'}"
							>
								{displayIcon}
							</span>
							{#if view.unreadCount > 0}
								<span
									class="absolute right-1 top-1 h-2 w-2 rounded-full bg-indigo-500"
									aria-label="{view.unreadCount} unread"
								></span>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
			<!-- Spacer to match the Reorder/Add buttons height in expanded view -->
			<div class="mt-1 pl-1 pt-1 pb-6 flex items-center justify-center" aria-hidden="true">
				<svg
					class="h-[18px] w-[18px] text-gray-300 dark:text-gray-600"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					xmlns="http://www.w3.org/2000/svg"
				>
					<line x1="12" x2="12" y1="19" y2="5" />
					<line x1="5" x2="19" y1="12" y2="12" />
				</svg>
			</div>
		{/if}

		<!-- Tags Section -->
		<!-- Clickable header for collapsing/expanding tags -->
		<button
			type="button"
			class="flex w-full items-center justify-center rounded-lg px-1 py-3 transition hover:bg-gray-200 dark:hover:bg-gray-800/50 cursor-pointer"
			on:click={onToggleTagsCollapsed}
			aria-label={tagsCollapsed ? "Expand tags" : "Collapse tags"}
			title={tagsCollapsed ? "Expand tags" : "Collapse tags"}
		>
			<svg
				class="h-[15px] w-[15px] text-gray-600 dark:text-gray-400 transition-transform dropdown-caret {!tagsCollapsed
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
			<ul class="space-y-1 pt-1">
				{#each sortedTags as tag (tag.id)}
					<li>
						<button
							type="button"
							class="relative flex h-9 w-9 items-center justify-center rounded-xl transition"
							on:click={() => onSelectTag(tag.id)}
							title={tag.name}
							aria-label={tag.name}
						>
							<span
								class="flex items-center justify-center rounded-xl py-1.5 px-2.5 text-base transition {!isSettingsRoute &&
								selectedTagId === tag.id
									? 'bg-indigo-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100'
									: 'text-gray-700 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-800/50'}"
							>
								<svg
									class="h-[18px] w-[18px]"
									viewBox="0 0 24 24"
									fill="currentColor"
									xmlns="http://www.w3.org/2000/svg"
									role="img"
									aria-hidden="true"
									style="color: {tag.color || '#6b7280'}"
								>
									{#each iconPaths.tag as path (path)}
										<path d={path} />
									{/each}
								</svg>
							</span>
							{#if tag.unreadCount && tag.unreadCount > 0}
								<span
									class="absolute right-1 top-1 h-2 w-2 rounded-full bg-indigo-500"
									aria-label="{tag.unreadCount} unread"
								></span>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
			<!-- Spacer to match the Reorder/Add buttons height in expanded view -->
			<div class="mt-1 pl-1 pt-1 pb-6 flex items-center justify-center" aria-hidden="true">
				<svg
					class="h-[18px] w-[18px] text-gray-300 dark:text-gray-600"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					xmlns="http://www.w3.org/2000/svg"
				>
					<line x1="12" x2="12" y1="19" y2="5" />
					<line x1="5" x2="19" y1="12" y2="12" />
				</svg>
			</div>
		{/if}
	</div>

	<div class="mx-auto w-10 mt-3 mb-2 border-b border-gray-200 dark:border-gray-800/90"></div>
	<a
		href={resolve("/settings")}
		class="flex items-center justify-center px-2"
		title="Settings"
		aria-label="Settings"
	>
		<span
			class="flex items-center justify-center rounded-xl p-2.5 transition {isSettingsRoute
				? 'bg-indigo-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100'
				: 'text-gray-700 hover:bg-gray-200 dark:text-gray-300 dark:hover:bg-gray-800/50'}"
		>
			<svg
				class="w-5 h-5"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="1.5"
				stroke-linecap="round"
				stroke-linejoin="round"
				xmlns="http://www.w3.org/2000/svg"
				role="img"
				aria-hidden="true"
			>
				{#each iconPaths.settings as path (path)}
					<path d={path} />
				{/each}
			</svg>
		</span>
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
