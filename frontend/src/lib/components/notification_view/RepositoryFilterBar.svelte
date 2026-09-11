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
	import type { NotificationPageController } from "$lib/state/types";
	import { formatRepositorySelection } from "$lib/utils/repositorySelection";
	import RepositoryFilterDropdown from "./RepositoryFilterDropdown.svelte";
	import RepositoryAvatar from "./RepositoryAvatar.svelte";

	export let totalCount: number;
	export let pageRangeStart: number;
	export let pageRangeEnd: number;
	/** True when the query differs from the view's query (shown as a hint next to the range). */
	export let hasQueryFilter: boolean = false;

	const pageController = getContext<NotificationPageController>("notificationPageController");
	const { repositoryFilterOpen } = pageController.stores;
	const { selectedRepositories, hasRepositoryFilter } = pageController.derived;

	// The toggle button is the dropdown's click-outside anchor: clicks on it must not be
	// treated as "outside" (they'd close and immediately reopen), but the rest of the bar,
	// including the dead space next to the range, should close it like anywhere else.
	let buttonElement: HTMLButtonElement | null = null;

	$: label = formatRepositorySelection($selectedRepositories, 2);
	$: avatars = $selectedRepositories.slice(0, 3);
	$: title = $hasRepositoryFilter
		? `Filtering ${$selectedRepositories.length} ${$selectedRepositories.length === 1 ? "repository" : "repositories"}: ${$selectedRepositories.map((r) => r.fullName).join(", ")} (F to change, Shift+F to clear)`
		: "Filter by repository (F)";

	function handleToggle() {
		if ($repositoryFilterOpen) {
			pageController.actions.closeRepositoryFilter();
		} else {
			pageController.actions.openRepositoryFilter();
		}
	}
</script>

<div class="relative flex items-center justify-between gap-3 pl-1">
	<button
		bind:this={buttonElement}
		type="button"
		class={`flex min-w-0 items-center gap-2 rounded-lg border px-2.5 py-1 text-[13px] transition cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 ${
			$hasRepositoryFilter
				? "border-indigo-400/60 dark:border-indigo-500/50 bg-indigo-50 dark:bg-indigo-500/10 text-gray-900 dark:text-gray-100"
				: "border-transparent bg-transparent text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-gray-200"
		}`}
		aria-haspopup="dialog"
		aria-expanded={$repositoryFilterOpen}
		{title}
		on:click={handleToggle}
	>
		{#if $hasRepositoryFilter}
			<span class="flex flex-shrink-0 -space-x-1" aria-hidden="true">
				{#each avatars as repo (repo.id)}
					<span class="rounded-sm ring-1 ring-white dark:ring-gray-900">
						<RepositoryAvatar fullName={repo.fullName} url={repo.ownerAvatarUrl} />
					</span>
				{/each}
			</span>
		{:else}
			<svg class="h-4 w-4 flex-shrink-0" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
				<path
					d="M2 2.5A2.5 2.5 0 0 1 4.5 0h8.75a.75.75 0 0 1 .75.75v12.5a.75.75 0 0 1-.75.75h-2.5a.75.75 0 0 1 0-1.5h1.75v-2h-8a1 1 0 0 0-.714 1.7.75.75 0 1 1-1.072 1.05A2.495 2.495 0 0 1 2 11.5Zm10.5-1h-8a1 1 0 0 0-1 1v6.708A2.486 2.486 0 0 1 4.5 9h8ZM5 12.25a.25.25 0 0 1 .25-.25h3.5a.25.25 0 0 1 .25.25v3.25a.25.25 0 0 1-.4.2l-1.45-1.087a.249.249 0 0 0-.3 0L5.4 15.7a.25.25 0 0 1-.4-.2Z"
				/>
			</svg>
		{/if}

		<span class="truncate font-medium">
			{#if label.names.length === 0}
				All repositories
			{:else}
				{label.names.join(", ")}
			{/if}
		</span>

		{#if label.overflow > 0}
			<span
				class="flex-shrink-0 rounded-full bg-indigo-500 px-1.5 py-px text-[10px] font-semibold leading-4 text-white"
			>
				+{label.overflow}
			</span>
		{/if}

		<svg
			class={`h-3 w-3 flex-shrink-0 transition-transform ${$repositoryFilterOpen ? "rotate-180" : ""}`}
			viewBox="0 0 12 12"
			fill="none"
			aria-hidden="true"
		>
			<path
				d="M2 4l4 4 4-4"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	</button>

	<span class="flex-shrink-0 whitespace-nowrap text-xs text-gray-500 tabular-nums">
		{#if totalCount === 0}
			0 of 0
		{:else if pageRangeStart === 0}
			0 of {totalCount}
		{:else}
			{pageRangeStart}-{pageRangeEnd} of {totalCount}
		{/if}
		{#if hasQueryFilter}
			<span class="text-gray-400 dark:text-gray-600"> · filtered</span>
		{/if}
	</span>

	{#if $repositoryFilterOpen}
		<RepositoryFilterDropdown anchor={buttonElement} />
	{/if}
</div>
