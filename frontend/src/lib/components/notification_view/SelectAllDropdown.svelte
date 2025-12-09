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

	import type { SelectAllMode } from "$lib/state/types";

	export let currentMode: SelectAllMode = "none";
	export let pageCount: number = 0;
	export let totalCount: number = 0;
	export let onSelectPage: () => void = () => {};
	export let onSelectAll: () => void = () => {};
	export let onClearSelection: () => void = () => {};

	let isOpen = false;

	function toggleDropdown() {
		isOpen = !isOpen;
	}

	function handleSelectPage() {
		if (currentMode === "page") {
			// Clicking on already selected "page" mode - clear selection
			onClearSelection();
		} else {
			onSelectPage();
		}
		isOpen = false;
	}

	function handleSelectAll() {
		if (currentMode === "all") {
			// Clicking on already selected "all" mode - clear selection
			onClearSelection();
		} else {
			onSelectAll();
		}
		isOpen = false;
	}

	function handleClickOutside(event: MouseEvent) {
		const target = event.target as Node;
		const dropdown = document.getElementById("select-all-dropdown");
		if (dropdown && !dropdown.contains(target)) {
			isOpen = false;
		}
	}

	$: {
		if (isOpen) {
			document.addEventListener("click", handleClickOutside);
		} else {
			document.removeEventListener("click", handleClickOutside);
		}
	}

	$: isAllSelected = currentMode === "all";
	$: isPageSelected = currentMode === "page";
	$: isAnySelected = isAllSelected || isPageSelected;
</script>

<div id="select-all-dropdown" class="relative">
	<button
		type="button"
		class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer"
		on:click={toggleDropdown}
	>
		<div class="flex items-center gap-1.5">
			<!-- Checkbox indicator -->
			<div
				class={`flex h-3.5 w-3.5 items-center justify-center rounded border ${isAnySelected ? "border-gray-700 dark:border-white bg-gray-700 dark:bg-white" : "border-gray-400 dark:border-gray-600 bg-transparent"}`}
			>
				{#if isAnySelected}
					<svg
						class="h-2.5 w-2.5 text-white dark:text-gray-700"
						viewBox="0 0 12 12"
						fill="none"
						xmlns="http://www.w3.org/2000/svg"
					>
						<path
							d="M10 3L4.5 8.5L2 6"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
				{/if}
			</div>
			<span>Select all</span>
		</div>
		<svg
			class={`h-3 w-3 transition-transform ${isOpen ? "rotate-180" : ""}`}
			viewBox="0 0 12 12"
			fill="currentColor"
		>
			<path
				d="M2 4l4 4 4-4"
				stroke="currentColor"
				stroke-width="2"
				fill="none"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	</button>

	{#if isOpen}
		<div
			class="absolute left-0 top-full z-50 mt-1 min-w-[200px] rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 py-1 shadow-lg"
		>
			<button
				type="button"
				class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
				on:click={handleSelectPage}
			>
				<div
					class={`flex h-4 w-4 items-center justify-center rounded border ${isPageSelected ? "border-indigo-500 bg-indigo-500" : "border-gray-400 dark:border-gray-600 bg-transparent"}`}
				>
					{#if isPageSelected}
						<svg
							class="h-3 w-3 text-white"
							viewBox="0 0 12 12"
							fill="none"
							xmlns="http://www.w3.org/2000/svg"
						>
							<path
								d="M10 3L4.5 8.5L2 6"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
					{/if}
				</div>
				<span>Select all on page ({pageCount})</span>
			</button>

			<button
				type="button"
				class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
				on:click={handleSelectAll}
			>
				<div
					class={`flex h-4 w-4 items-center justify-center rounded border ${isAllSelected ? "border-indigo-500 bg-indigo-500" : "border-gray-400 dark:border-gray-600 bg-transparent"}`}
				>
					{#if isAllSelected}
						<svg
							class="h-3 w-3 text-white"
							viewBox="0 0 12 12"
							fill="none"
							xmlns="http://www.w3.org/2000/svg"
						>
							<path
								d="M10 3L4.5 8.5L2 6"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
					{/if}
				</div>
				<span>Select all ({totalCount})</span>
			</button>
		</div>
	{/if}
</div>
