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

	import UnifiedSearchInput from "../shared/UnifiedSearchInput.svelte";
	import CompactPagination from "../shared/CompactPagination.svelte";
	import type { UnifiedSearchInputComponent } from "$lib/types/components";

	// Combined query string (free text + structured filters)
	export let combinedQuery = "";
	export let canUpdateView = false; // True if we're in a custom view (not inbox/done)

	export let onChangeQuery: (value: string) => void = () => {}; // Called on keystroke
	export let onSubmitQuery: (value: string) => void = () => {}; // Called on Enter/submit
	export let onClear: () => void = () => {};
	export let onUpdateView: () => void = () => {};
	export let onSaveAsNewView: () => void = () => {};

	// Pagination props
	export let page: number;
	export let totalPages: number;
	export let onPrevious: () => void;
	export let onNext: () => void;

	// Multiselect props
	export let multiselectMode = false;
	export let onToggleMultiselect: () => void = () => {};

	// Reading pane props
	export let splitModeEnabled: boolean = false;
	export let onToggleSplitMode: () => void = () => {};

	let unifiedSearchComponent: UnifiedSearchInputComponent | null = null;

	export function focusSearchInput(): boolean {
		unifiedSearchComponent?.focus();
		return unifiedSearchComponent !== null;
	}

	function handleQueryChange(newQuery: string) {
		onChangeQuery(newQuery);
	}

	function handleQuerySubmit(submittedQuery: string) {
		onSubmitQuery(submittedQuery);
	}

	function handleClear() {
		onClear();
	}

	// Always show save button when there's a query
	$: showSaveButton = combinedQuery.trim().length > 0;
</script>

<div class="flex items-center gap-2 pr-3 pt-4">
	<button
		type="button"
		class={`flex h-9 items-center justify-center rounded-full border px-4 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 cursor-pointer ${
			multiselectMode
				? "border-indigo-500 bg-indigo-500 text-white hover:bg-indigo-600"
				: "border-gray-300 dark:border-gray-800 bg-gray-100 dark:bg-gray-900 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-800"
		}`}
		title={multiselectMode ? "Exit multiselect mode" : "Enter multiselect mode"}
		on:click={onToggleMultiselect}
	>
		<svg width="18" height="18" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
			<path
				d="M3 9V19.4C3 19.9601 3 20.2399 3.10899 20.4538C3.20487 20.642 3.35774 20.7952 3.5459 20.8911C3.7596 21 4.0395 21 4.59846 21H15.0001M17 8L13 12L11 10M7 13.8002V6.2002C7 5.08009 7 4.51962 7.21799 4.0918C7.40973 3.71547 7.71547 3.40973 8.0918 3.21799C8.51962 3 9.08009 3 10.2002 3H17.8002C18.9203 3 19.4801 3 19.9079 3.21799C20.2842 3.40973 20.5905 3.71547 20.7822 4.0918C21.0002 4.51962 21.0002 5.07969 21.0002 6.19978L21.0002 13.7998C21.0002 14.9199 21.0002 15.48 20.7822 15.9078C20.5905 16.2841 20.2842 16.5905 19.9079 16.7822C19.4805 17 18.9215 17 17.8036 17H10.1969C9.07899 17 8.5192 17 8.0918 16.7822C7.71547 16.5905 7.40973 16.2842 7.21799 15.9079C7 15.4801 7 14.9203 7 13.8002Z"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	</button>

	<!-- Unified search input with query modified indicator -->
	<div class="relative flex-1">
		<UnifiedSearchInput
			bind:this={unifiedSearchComponent}
			value={combinedQuery}
			onChange={handleQueryChange}
			onSubmit={handleQuerySubmit}
			placeholder="Search or filter notifications"
			{showSaveButton}
			{canUpdateView}
			onSaveAsNew={onSaveAsNewView}
			{onUpdateView}
		/>
	</div>

	<!-- Compact Pagination (hidden in focus mode) -->

	<CompactPagination {page} {totalPages} {onPrevious} {onNext} />

	<!-- Reading pane toggle button -->
	<button
		type="button"
		class="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 dark:border-gray-800 bg-gray-100 dark:bg-gray-900 text-gray-600 dark:text-gray-400 transition hover:bg-gray-200 dark:hover:bg-gray-800 hover:text-gray-800 dark:hover:text-gray-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 cursor-pointer"
		title={splitModeEnabled ? "Hide reading pane (Shift+P)" : "Show reading pane (Shift+P)"}
		on:click={onToggleSplitMode}
	>
		{#if splitModeEnabled}
			<!-- Show list icon when in split mode -->
			<svg
				class="h-4 w-4"
				viewBox="0 0 16 16"
				fill="currentColor"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path d="M2 10V9h12v1H2zm0-4h12v1H2V6zm12-3v1H2V3h12zM2 12v1h12v-1H2z" />
			</svg>
		{:else}
			<!-- Show split icon when in list mode -->
			<svg
				class="h-4 w-4"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
				<line x1="9" y1="3" x2="9" y2="21" />
			</svg>
		{/if}
	</button>
</div>
