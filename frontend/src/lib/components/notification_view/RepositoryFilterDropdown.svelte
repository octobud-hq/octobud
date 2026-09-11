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

	import { getContext, onMount, tick } from "svelte";
	import type { NotificationPageController } from "$lib/state/types";
	import { buildRepositoryOptions } from "$lib/utils/repositorySelection";
	import RepositoryFilterOption from "./RepositoryFilterOption.svelte";

	/** The toggle button; clicks on it are not "outside" so it can close the dropdown itself. */
	export let anchor: HTMLElement | null = null;

	const pageController = getContext<NotificationPageController>("notificationPageController");
	const { repositoryCounts, repositoryCountsLoading, selectedRepositoryIds, pinnedRepositoryIds } =
		pageController.stores;

	let filterText = "";
	let inputElement: HTMLInputElement | null = null;
	let listElement: HTMLDivElement | null = null;
	let rootElement: HTMLDivElement | null = null;

	// Index into `rows`, where row 0 is "All repositories" and the rest are options.
	let highlightIndex = 0;

	$: groups = buildRepositoryOptions(
		$repositoryCounts,
		$selectedRepositoryIds,
		$pinnedRepositoryIds,
		filterText
	);
	$: rows = groups.all;
	$: rowCount = rows.length + 1;
	$: if (highlightIndex >= rowCount) {
		highlightIndex = Math.max(0, rowCount - 1);
	}
	$: selectedCount = $selectedRepositoryIds.length;

	// When the filter text changes, jump to the first match so typing then Enter/Space
	// works without arrowing. "All repositories" (row 0) stays available as a bailout and
	// is the fallback when nothing matches or the filter is cleared. Keyed on the text so
	// toggles and count refreshes (which rebuild `rows`) don't move the highlight.
	let lastFilterText = "";
	$: if (filterText !== lastFilterText) {
		lastFilterText = filterText;
		highlightIndex = filterText.trim() !== "" && rows.length > 0 ? 1 : 0;
	}
	$: allSelected = selectedCount === 0;

	function close() {
		pageController.actions.closeRepositoryFilter();
	}

	function chooseOnly(row: number) {
		if (row === 0) {
			void pageController.actions.clearRepositoryFilter();
		} else {
			const option = rows[row - 1];
			if (option) {
				void pageController.actions.selectOnlyRepository(option.id);
			}
		}
		close();
	}

	function toggleRow(row: number) {
		if (row === 0) {
			void pageController.actions.clearRepositoryFilter();
			return;
		}
		const option = rows[row - 1];
		if (option) {
			void pageController.actions.toggleRepositoryInFilter(option.id);
		}
	}

	function moveHighlight(delta: number) {
		if (rowCount === 0) return;
		highlightIndex = (highlightIndex + delta + rowCount) % rowCount;
		void tick().then(() => {
			const element = listElement?.querySelector<HTMLElement>(
				`[data-repo-row="${highlightIndex}"]`
			);
			element?.scrollIntoView({ block: "nearest" });
		});
	}

	function handleKeydown(event: KeyboardEvent) {
		switch (event.key) {
			case "ArrowDown":
				event.preventDefault();
				moveHighlight(1);
				break;
			case "ArrowUp":
				event.preventDefault();
				moveHighlight(-1);
				break;
			case "Enter":
				event.preventDefault();
				chooseOnly(highlightIndex);
				break;
			case " ":
				// Repository names never contain spaces, so Space is free to toggle.
				event.preventDefault();
				toggleRow(highlightIndex);
				break;
			case "Escape":
				event.preventDefault();
				close();
				break;
			case "Tab":
				close();
				break;
			default:
				return;
		}
		event.stopPropagation();
	}

	// Registered on window in the capture phase, like the other dropdowns, so it still sees
	// clicks on elements that stop mousedown propagation (e.g. the snooze/tag menus).
	function handleClickOutside(event: MouseEvent) {
		const target = event.target as Node | null;
		if (!target) return;
		if (rootElement?.contains(target) || anchor?.contains(target)) return;
		close();
	}

	onMount(() => {
		// Always open on "All repositories". Opening the selector is a context switch to
		// repositories, so neither the focused notification's repo nor the current selection
		// should pre-empt where the cursor starts.
		highlightIndex = 0;

		inputElement?.focus();
		window.addEventListener("mousedown", handleClickOutside, true);
		return () => {
			window.removeEventListener("mousedown", handleClickOutside, true);
			// Give keyboard control back to the list.
			if (document.activeElement instanceof HTMLElement) {
				document.activeElement.blur();
			}
		};
	});
</script>

<div
	bind:this={rootElement}
	class="absolute left-0 top-full z-50 mt-1 flex w-full max-w-md flex-col rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-gray-200 shadow-lg"
	role="dialog"
	aria-label="Filter by repository"
	tabindex="-1"
	on:keydown={handleKeydown}
>
	<div class="border-b border-gray-200 dark:border-gray-800 p-2">
		<input
			bind:this={inputElement}
			bind:value={filterText}
			type="text"
			placeholder="Filter repositories…"
			aria-label="Filter repositories"
			autocomplete="off"
			spellcheck="false"
			class="w-full rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-950 px-2.5 py-1.5 text-sm text-gray-900 dark:text-gray-100 placeholder:text-gray-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/40"
		/>
	</div>

	<div
		bind:this={listElement}
		class="max-h-80 overflow-y-auto p-1"
		role="listbox"
		aria-multiselectable="true"
	>
		<!-- "All repositories" behaves like a radio: checked when nothing is selected -->
		<button
			type="button"
			data-repo-row="0"
			role="option"
			aria-selected={allSelected}
			class={`flex w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-left transition cursor-pointer ${
				highlightIndex === 0
					? "bg-gray-200 dark:bg-gray-700"
					: "hover:bg-gray-100 dark:hover:bg-gray-800"
			}`}
			on:click={() => chooseOnly(0)}
			on:mousemove={() => (highlightIndex = 0)}
		>
			<span
				class={`flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full border ${
					allSelected ? "border-indigo-500 bg-indigo-500" : "border-gray-400 dark:border-gray-600"
				}`}
				aria-hidden="true"
			>
				{#if allSelected}
					<span class="h-1.5 w-1.5 rounded-full bg-white"></span>
				{/if}
			</span>
			<span class="flex-1 font-medium">All repositories</span>
		</button>

		{#each rows as option, index (option.id)}
			{#if index === 0 && option.pinned}
				<div
					class="px-2.5 pt-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500"
				>
					Pinned
				</div>
			{:else if !option.pinned && groups.pinned.length > 0 && index === groups.pinned.length}
				<div
					class="px-2.5 pt-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-500"
				>
					Repositories
				</div>
			{/if}
			<RepositoryFilterOption
				{option}
				row={option.row}
				highlighted={highlightIndex === option.row}
				onToggle={() => toggleRow(option.row)}
				onChooseOnly={() => chooseOnly(option.row)}
				onHover={() => (highlightIndex = option.row)}
				onTogglePin={() => pageController.actions.toggleRepositoryPin(option.id)}
			/>
		{/each}

		{#if rows.length === 0}
			<div class="px-2.5 py-6 text-center text-xs text-gray-500">
				{#if $repositoryCountsLoading}
					Loading repositories…
				{:else if filterText.trim()}
					No repositories match “{filterText.trim()}”
				{:else}
					No repositories have notifications here
				{/if}
			</div>
		{/if}
	</div>

	<div
		class="flex items-center justify-between gap-3 border-t border-gray-200 dark:border-gray-800 px-3 py-1.5 text-[11px] text-gray-500"
	>
		<span>
			{#if selectedCount > 0}
				{selectedCount} selected ·
				<button
					type="button"
					class="font-medium text-blue-500 hover:text-blue-400 cursor-pointer"
					on:click={() => chooseOnly(0)}
				>
					Clear
				</button>
			{:else}
				All repositories
			{/if}
		</span>
		<span class="hidden sm:inline whitespace-nowrap"
			>↑↓ move · space toggle · ⏎ only this · esc</span
		>
	</div>
</div>
