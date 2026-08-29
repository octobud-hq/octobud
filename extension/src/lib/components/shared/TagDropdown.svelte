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

	import type { Notification, Tag } from "$lib/api/types";
	import { tick, onMount, onDestroy } from "svelte";
	import { fetchTags } from "$lib/api/tags";

	export let notification: Notification;
	export let isOpen: boolean = false;
	export let onAssignTag: (tagId: string) => Promise<void> = async () => {};
	export let onRemoveTag: (tagId: string) => Promise<void> = async () => {};
	export let onClose: () => void = () => {};

	let dropdownElement: HTMLDivElement | null = null;
	let buttonElement: HTMLButtonElement | null = null;
	let inputElement: HTMLInputElement | null = null;
	let dropdownPosition: {
		top: number;
		left: number | "auto";
		right: number | "auto";
	} = {
		top: 0,
		left: 0,
		right: "auto",
	};
	let isPositioned = false;
	let searchTerm = "";
	let availableTags: Tag[] = [];
	let highlightedIndex = -1;

	export function setButtonElement(element: HTMLButtonElement | null) {
		buttonElement = element;
	}

	// Reset state when dropdown closes
	$: if (!isOpen) {
		isPositioned = false;
		searchTerm = "";
		highlightedIndex = -1;
		detachOutsideClickListener();
	}

	// Attach outside click listener when dropdown opens
	$: if (isOpen) {
		void tick().then(() => {
			attachOutsideClickListener();
		});
	}

	// Focus input when dropdown opens
	$: if (isOpen && inputElement) {
		void tick().then(() => {
			inputElement?.focus();
		});
	}

	// Load tags when dropdown opens
	$: if (isOpen) {
		void loadTags();
	}

	function handleOutsideClick(event: MouseEvent) {
		const target = event.target as Node | null;

		// Check if click is outside both dropdown and button
		if (
			dropdownElement &&
			!dropdownElement.contains(target) &&
			buttonElement &&
			!buttonElement.contains(target)
		) {
			onClose();
		}
	}

	function attachOutsideClickListener() {
		if (typeof window === "undefined") return;
		// Use capture phase to catch the event before it's stopped by stopPropagation
		window.addEventListener("mousedown", handleOutsideClick, true);
	}

	function detachOutsideClickListener() {
		if (typeof window === "undefined") return;
		window.removeEventListener("mousedown", handleOutsideClick, true);
	}

	onDestroy(() => {
		detachOutsideClickListener();
	});

	async function loadTags() {
		try {
			availableTags = await fetchTags();
		} catch (error) {
			console.error("Failed to load tags:", error);
			availableTags = [];
		}
	}

	// Calculate dropdown position based on button position
	async function calculatePosition() {
		if (!dropdownElement || !buttonElement || typeof window === "undefined") {
			return;
		}

		await tick();

		const buttonRect = buttonElement.getBoundingClientRect();
		const dropdownWidth = 280;
		const dropdownHeight = dropdownElement.scrollHeight || 300;
		const viewportWidth = window.innerWidth;
		const viewportHeight = window.innerHeight;
		const padding = 8;

		let top = buttonRect.bottom + padding;
		let left: number | "auto" = buttonRect.left;
		let right: number | "auto" = "auto";

		const wouldOverflowRight = (left as number) + dropdownWidth > viewportWidth - 20;
		if (wouldOverflowRight) {
			right = viewportWidth - buttonRect.right;
			left = buttonRect.right - dropdownWidth;
		}

		const wouldOverflowBottom = top + dropdownHeight > viewportHeight - 20;
		if (wouldOverflowBottom) {
			top = buttonRect.top - dropdownHeight - padding;
		}

		if (top < 10) {
			top = 10;
		}

		if (left < 10) {
			left = 10;
		}

		dropdownPosition = {
			top,
			left: right === "auto" ? left : "auto",
			right: right === "auto" ? "auto" : right,
		};
		isPositioned = true;
	}

	$: if (dropdownElement && buttonElement && isOpen && (typeof window === "undefined") === false) {
		void calculatePosition();
	}

	// Get tags assigned to this notification (track by ID)
	$: assignedTagIds = new Set((notification.tags ?? []).map((tag) => tag.id));

	// Filter available tags to unassigned only and sort by relevance
	$: unassignedTags = availableTags
		.filter((tag) => !assignedTagIds.has(tag.id))
		.filter((tag) => {
			const matchesSearch = tag.name.toLowerCase().includes(searchTerm.toLowerCase());
			return matchesSearch;
		})
		.sort((a, b) => {
			// Sort by: starts with search term first, then alphabetically
			const aStarts = a.name.toLowerCase().startsWith(searchTerm.toLowerCase());
			const bStarts = b.name.toLowerCase().startsWith(searchTerm.toLowerCase());
			if (aStarts && !bStarts) return -1;
			if (!aStarts && bStarts) return 1;
			return a.name.localeCompare(b.name);
		});

	// First suggestion for auto-fill
	$: firstSuggestion = unassignedTags.length > 0 ? unassignedTags[0] : null;

	// Auto-fill text (the part after the search term)
	$: autoFillText = (() => {
		if (!firstSuggestion || searchTerm.trim().length === 0) return "";
		const suggestion = firstSuggestion.name;
		const searchLower = searchTerm.toLowerCase();
		const suggestionLower = suggestion.toLowerCase();

		// Only auto-fill if suggestion starts with search term
		if (suggestionLower.startsWith(searchLower)) {
			return suggestion.slice(searchTerm.length);
		}
		return "";
	})();

	// Check if search term matches existing tag exactly
	$: exactMatch = unassignedTags.some((tag) => tag.name.toLowerCase() === searchTerm.toLowerCase());

	// Show create option if search term doesn't match exactly
	$: showCreateOption = searchTerm.trim().length > 0 && !exactMatch;

	async function handleTagToggle(tagId: string) {
		if (assignedTagIds.has(tagId)) {
			await onRemoveTag(tagId);
		} else {
			await onAssignTag(tagId);
		}
		// Reload tags after any assignment/removal
		await loadTags();
	}

	async function handleAssignFirstSuggestion() {
		// Don't assign if input is empty
		if (searchTerm.trim().length === 0) {
			return;
		}

		if (firstSuggestion) {
			// Assign the first suggestion
			await onAssignTag(firstSuggestion.id);
			// Clear the input
			searchTerm = "";
			highlightedIndex = -1;
			// Reload tags
			await loadTags();
			// Re-focus input
			await tick();
			inputElement?.focus();
		} else if (showCreateOption) {
			// No suggestions but we can create
			await handleCreateTag();
		}
	}

	async function handleCreateTag() {
		if (!showCreateOption) return;
		// Create and assign the tag inline
		await onAssignTag(searchTerm.trim());
		searchTerm = "";
		highlightedIndex = -1;
		// Reload tags after inline creation
		await loadTags();
		// Re-focus input
		await tick();
		inputElement?.focus();
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === "Escape") {
			// First escape: unfocus input if it has focus
			if (document.activeElement === inputElement) {
				event.preventDefault();
				inputElement?.blur();
				return;
			}
			// Second escape (or escape when input not focused): close dropdown
			onClose();
			return;
		}

		if (event.key === "ArrowDown") {
			event.preventDefault();
			highlightedIndex = Math.min(highlightedIndex + 1, unassignedTags.length - 1);
		} else if (event.key === "ArrowUp") {
			event.preventDefault();
			highlightedIndex = Math.max(highlightedIndex - 1, -1);
		} else if (event.key === "Enter") {
			event.preventDefault();

			// If a suggestion is highlighted, select it
			if (highlightedIndex >= 0 && highlightedIndex < unassignedTags.length) {
				const selected = unassignedTags[highlightedIndex];
				void handleTagToggle(selected.id);
				highlightedIndex = -1;
			} else {
				// No highlight - assign first suggestion
				void handleAssignFirstSuggestion();
			}
		}
	}

	function handleInputChange() {
		// Reset highlight when search changes
		highlightedIndex = -1;
	}
</script>

{#if isOpen}
	<div
		bind:this={dropdownElement}
		class="fixed z-[150] min-w-[280px] max-w-[320px] rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-800 dark:bg-gray-900"
		style="top: {dropdownPosition.top}px; {dropdownPosition.left !== 'auto'
			? `left: ${dropdownPosition.left}px`
			: `right: ${dropdownPosition.right}px`}; opacity: {isPositioned ? 1 : 0};"
		role="dialog"
		aria-label="Manage tags"
		tabindex="-1"
		on:click|stopPropagation
		on:mousedown|stopPropagation
		on:keydown|stopPropagation
	>
		<!-- Input with auto-fill suggestion -->
		<div class="p-3 border-b border-gray-200 dark:border-gray-800 cursor-default">
			<div class="relative">
				<!-- Auto-fill suggestion (background) -->
				{#if autoFillText}
					<div
						class="pointer-events-none absolute inset-0 flex items-center px-3 py-2 text-sm text-gray-400 dark:text-gray-600"
						aria-hidden="true"
					>
						<span class="invisible">{searchTerm}</span><span>{autoFillText}</span>
					</div>
				{/if}
				<!-- Actual input -->
				<input
					bind:this={inputElement}
					bind:value={searchTerm}
					on:input={handleInputChange}
					on:keydown={handleKeyDown}
					type="text"
					placeholder="Search or create tag..."
					class="relative w-full rounded-lg border border-gray-300 bg-transparent px-3 py-2 text-sm text-gray-900 placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30 dark:border-gray-700 dark:text-gray-100 dark:placeholder-gray-500"
				/>
			</div>
		</div>

		<!-- Assigned tags section -->
		<div class="p-3 border-b border-gray-200 dark:border-gray-800 cursor-default">
			<div class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">Assigned Tags</div>
			{#if assignedTagIds.size === 0}
				<div class="text-xs text-gray-500 dark:text-gray-400 italic py-1">No tags assigned</div>
			{:else}
				<div class="flex flex-wrap gap-1.5">
					{#each notification.tags ?? [] as tag (tag.id)}
						<button
							type="button"
							class="group flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition hover:opacity-80 cursor-pointer"
							style="background-color: {tag.color}20; color: {tag.color}"
							on:click={() => handleTagToggle(tag.id)}
						>
							<span>{tag.name}</span>
							<svg
								class="h-3 w-3"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path d="M18 6L6 18M6 6l12 12" />
							</svg>
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Suggestions list (unassigned tags only) -->
		<div class="max-h-[240px] overflow-y-auto p-2 cursor-default">
			{#if unassignedTags.length === 0 && searchTerm.trim().length === 0}
				<div class="px-3 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
					All tags assigned
				</div>
			{:else if unassignedTags.length === 0 && searchTerm.trim().length > 0}
				<div class="px-3 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
					{#if showCreateOption}
						<div class="mb-2">No matching tags</div>
						<div class="text-xs">
							Press Enter to create "{searchTerm.trim()}"
						</div>
					{:else}
						No matching tags
					{/if}
				</div>
			{:else}
				{#each unassignedTags as tag, index (tag.id)}
					{@const isHighlighted =
						highlightedIndex === index ||
						(highlightedIndex === -1 && index === 0 && searchTerm.trim().length > 0)}
					<button
						type="button"
						class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition cursor-pointer {isHighlighted
							? 'bg-blue-50 dark:bg-blue-900/20'
							: 'hover:bg-gray-50 dark:hover:bg-gray-800/50'}"
						on:click={() => handleTagToggle(tag.id)}
					>
						<span
							class="flex-1 {isHighlighted
								? 'font-medium text-blue-700 dark:text-blue-400'
								: 'text-gray-700 dark:text-gray-300'}"
						>
							{tag.name}
						</span>
						{#if tag.color}
							<div
								class="h-4 w-4 rounded-full border border-gray-200 dark:border-gray-700"
								style="background-color: {tag.color}"
							></div>
						{/if}
					</button>
				{/each}
			{/if}
		</div>
	</div>
{/if}
