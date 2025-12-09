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

	import { onMount } from "svelte";
	import type { Tag } from "$lib/api/tags";

	export let selectedTags: string[] = []; // Tag IDs as strings
	export let availableTags: Tag[] = [];
	export let placeholder = "Type to add tags...";

	let inputValue = "";
	let inputElement: HTMLInputElement;
	let isFocused = false;
	let selectedSuggestionIndex = -1;
	let suggestionsElement: HTMLDivElement;

	// Helper to get tag name by ID
	function getTagName(tagId: string): string {
		const tag = availableTags.find((t) => t.id === tagId);
		return tag?.name || tagId;
	}

	// Show filtered suggestions based on input, filtered by name for user friendliness
	$: filteredSuggestions = inputValue.trim()
		? availableTags
				.filter(
					(tag) =>
						tag.name.toLowerCase().includes(inputValue.trim().toLowerCase()) &&
						!selectedTags.includes(tag.id)
				)
				.slice(0, 10)
		: availableTags.filter((tag) => !selectedTags.includes(tag.id)).slice(0, 10);

	// Show suggestions when focused and there are tags to show
	$: showSuggestions = isFocused && filteredSuggestions.length > 0;

	function addTagByID(tagId: string) {
		if (tagId && !selectedTags.includes(tagId)) {
			selectedTags = [...selectedTags, tagId];
			inputValue = "";
			selectedSuggestionIndex = -1;
		}
	}

	function addTagByName(tagName: string) {
		// Try to find tag by name (for backward compatibility or manual entry)
		const trimmed = tagName.trim();
		if (!trimmed) return;

		const tag = availableTags.find((t) => t.name.toLowerCase() === trimmed.toLowerCase());
		if (tag && !selectedTags.includes(tag.id)) {
			addTagByID(tag.id);
		}
	}

	function removeTag(tagId: string) {
		selectedTags = selectedTags.filter((t) => t !== tagId);
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === "Enter") {
			event.preventDefault();
			if (selectedSuggestionIndex >= 0 && filteredSuggestions[selectedSuggestionIndex]) {
				addTagByID(filteredSuggestions[selectedSuggestionIndex].id);
			} else if (inputValue.trim()) {
				addTagByName(inputValue);
			}
		} else if (event.key === "," || event.key === "Tab") {
			event.preventDefault();
			if (inputValue.trim()) {
				addTagByName(inputValue);
			}
		} else if (event.key === "Backspace" && !inputValue && selectedTags.length > 0) {
			removeTag(selectedTags[selectedTags.length - 1]);
		} else if (event.key === "ArrowDown") {
			event.preventDefault();
			selectedSuggestionIndex = Math.min(
				selectedSuggestionIndex + 1,
				filteredSuggestions.length - 1
			);
		} else if (event.key === "ArrowUp") {
			event.preventDefault();
			selectedSuggestionIndex = Math.max(selectedSuggestionIndex - 1, -1);
		} else if (event.key === "Escape") {
			inputValue = "";
			selectedSuggestionIndex = -1;
			isFocused = false;
			inputElement?.blur();
		}
	}

	function handleInputFocus() {
		isFocused = true;
		selectedSuggestionIndex = -1;
	}

	function handleInputBlur() {
		// Delay to allow clicking on suggestions
		setTimeout(() => {
			isFocused = false;
			if (inputValue.trim()) {
				addTagByName(inputValue);
			}
		}, 200);
	}

	function selectSuggestion(tag: Tag) {
		addTagByID(tag.id);
		inputElement?.focus();
	}
</script>

<div class="relative">
	<!-- Tag chips and input container -->
	<div
		class="flex flex-wrap gap-1.5 px-3 py-2 bg-white dark:bg-gray-950 border border-gray-300 dark:border-gray-800 rounded-lg focus-within:border-blue-600 focus-within:ring-2 focus-within:ring-blue-600/30 transition-all min-h-[42px]"
	>
		{#each selectedTags as tagId (tagId)}
			<span
				class="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border border-blue-300 dark:border-blue-800/40 text-xs rounded-full"
			>
				{getTagName(tagId)}
				<button
					type="button"
					on:click={() => removeTag(tagId)}
					class="hover:bg-blue-200 dark:hover:bg-blue-900/50 rounded-full p-0.5 transition-colors"
					aria-label="Remove tag"
				>
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</span>
		{/each}
		<input
			bind:this={inputElement}
			bind:value={inputValue}
			on:keydown={handleKeyDown}
			on:focus={handleInputFocus}
			on:blur={handleInputBlur}
			type="text"
			{placeholder}
			class="flex-1 min-w-[120px] bg-transparent border-none outline-none text-gray-900 dark:text-gray-200 text-sm placeholder-gray-500 dark:placeholder-gray-500"
		/>
	</div>

	<!-- Autocomplete suggestions -->
	{#if showSuggestions}
		<div
			bind:this={suggestionsElement}
			class="absolute z-10 w-full mt-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-60 overflow-y-auto"
		>
			{#each filteredSuggestions as tag, index (tag.id)}
				<button
					type="button"
					on:click={() => selectSuggestion(tag)}
					class="w-full text-left px-3 py-2 text-sm text-gray-900 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors {selectedSuggestionIndex ===
					index
						? 'bg-gray-100 dark:bg-gray-700'
						: ''}"
				>
					<div class="flex items-center justify-between">
						<span>{tag.name}</span>
						{#if tag.color}
							<span class="w-3 h-3 rounded-full" style="background-color: {tag.color}"></span>
						{/if}
					</div>
					{#if tag.description}
						<div class="text-xs text-gray-600 dark:text-gray-500 mt-0.5">
							{tag.description}
						</div>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>
