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

	import type { RepositoryOption } from "$lib/utils/repositorySelection";
	import RepositoryAvatar from "./RepositoryAvatar.svelte";

	/** One checkbox row in the repository filter dropdown. */
	export let option: RepositoryOption;
	export let row: number;
	export let highlighted: boolean = false;
	export let onToggle: () => void;
	export let onChooseOnly: () => void;
	export let onHover: () => void;
	export let onTogglePin: () => void;

	// Unread badge mirrors the sidebar's view/tag badges: no badge means nothing unread,
	// not that the repository is empty (pinned repos can legitimately have no matches).
	$: unreadLabel = `${option.unread} unread notification${option.unread === 1 ? "" : "s"}`;
</script>

<div class="group relative">
	<button
		type="button"
		data-repo-row={row}
		role="option"
		aria-selected={option.selected}
		class={`flex w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 pr-9 text-left transition cursor-pointer ${
			highlighted ? "bg-gray-200 dark:bg-gray-700" : "hover:bg-gray-100 dark:hover:bg-gray-800"
		} ${option.total === 0 ? "opacity-60" : ""}`}
		on:click={onToggle}
		on:dblclick={onChooseOnly}
		on:mousemove={onHover}
	>
		<span
			class={`flex h-4 w-4 flex-shrink-0 items-center justify-center rounded border ${
				option.selected ? "border-indigo-500 bg-indigo-500" : "border-gray-400 dark:border-gray-600"
			}`}
			aria-hidden="true"
		>
			{#if option.selected}
				<svg class="h-3 w-3 text-white" viewBox="0 0 12 12" fill="none">
					<path
						d="M10 3L4.5 8.5L2 6"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
				</svg>
			{/if}
		</span>
		<RepositoryAvatar fullName={option.fullName} url={option.ownerAvatarUrl} />
		<span class="min-w-0 flex-1 truncate">{option.fullName}</span>
		{#if option.unread > 0}
			<span
				class={`flex h-6 flex-shrink-0 items-center justify-center rounded-full px-2 text-[11px] font-semibold ${
					highlighted
						? "bg-white/80 text-gray-700 dark:bg-gray-900 dark:text-gray-100"
						: "bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-100"
				}`}
				aria-label={unreadLabel}
			>
				{option.unread}
			</span>
		{/if}
	</button>
	<button
		type="button"
		class={`absolute right-1.5 top-1/2 -translate-y-1/2 rounded-md p-1 transition cursor-pointer ${
			option.pinned
				? "text-indigo-500 hover:bg-gray-200 dark:hover:bg-gray-700"
				: `text-gray-400 hover:bg-gray-200 hover:text-gray-700 dark:hover:bg-gray-700 dark:hover:text-gray-200 ${
						highlighted ? "opacity-100" : "opacity-0 group-hover:opacity-100"
					}`
		}`}
		title={option.pinned ? "Unpin repository" : "Pin repository"}
		aria-label={`${option.pinned ? "Unpin" : "Pin"} ${option.fullName}`}
		on:click|stopPropagation={onTogglePin}
	>
		{#if option.pinned}
			<svg class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
				<path
					d="M4.456.734a1.75 1.75 0 0 1 2.826.504l.613 1.327a3.08 3.08 0 0 0 2.084 1.707l2.454.584c1.332.317 1.8 1.972.832 2.94L11.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06L10 11.06l-2.204 2.205c-.968.968-2.623.5-2.94-.832l-.584-2.454a3.08 3.08 0 0 0-1.707-2.084l-1.327-.613a1.75 1.75 0 0 1-.504-2.826z"
				/>
			</svg>
		{:else}
			<svg class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
				<path
					d="M4.456.734a1.75 1.75 0 0 1 2.826.504l.613 1.327a3.08 3.08 0 0 0 2.084 1.707l2.454.584c1.332.317 1.8 1.972.832 2.94L11.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06L10 11.06l-2.204 2.205c-.968.968-2.623.5-2.94-.832l-.584-2.454a3.08 3.08 0 0 0-1.707-2.084l-1.327-.613a1.75 1.75 0 0 1-.504-2.826ZM5.92 1.866a.25.25 0 0 0-.404-.072L1.794 5.516a.25.25 0 0 0 .072.404l1.328.613A4.582 4.582 0 0 1 5.73 9.63l.584 2.454a.25.25 0 0 0 .42.12l5.47-5.47a.25.25 0 0 0-.12-.42L9.63 5.73a4.581 4.581 0 0 1-3.098-2.537z"
				/>
			</svg>
		{/if}
	</button>
</div>
