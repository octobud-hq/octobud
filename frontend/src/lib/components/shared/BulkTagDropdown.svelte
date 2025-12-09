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

	import type { Tag } from "$lib/api/types";
	import { tick, onMount, onDestroy } from "svelte";

	export let isOpen: boolean = false;
	export let availableTags: Tag[] = [];
	export let onAssignTag: (tagId: string) => Promise<void> = async () => {};
	export let onRemoveTag: (tagId: string) => Promise<void> = async () => {};
	export let onClose: () => void = () => {};

	let dropdownElement: HTMLDivElement | null = null;
	let buttonElement: HTMLButtonElement | null = null;
	let searchInputElement: HTMLInputElement | null = null;
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

		if (typeof left === "number" && left < 10) {
			left = 10;
		}

		dropdownPosition = {
			top,
			left: right === "auto" ? left : "auto",
			right: right === "auto" ? "auto" : right,
		};
		isPositioned = true;

		// Focus the search input after positioning
		await tick();
		if (searchInputElement) {
			searchInputElement.focus();
		}
	}

	// Recalculate position when dropdown opens or window resizes
	$: if (dropdownElement && buttonElement && isOpen && typeof window !== "undefined") {
		void calculatePosition();
	}

	onMount(() => {
		if (typeof window !== "undefined") {
			window.addEventListener("resize", calculatePosition);
		}
	});

	onDestroy(() => {
		if (typeof window !== "undefined") {
			window.removeEventListener("resize", calculatePosition);
		}
	});

	// Filter tags based on search term
	$: filteredTags = availableTags.filter((tag) =>
		tag.name.toLowerCase().includes(searchTerm.toLowerCase())
	);

	function handleTagClick(tag: Tag) {
		// For bulk operations, we'll apply the tag (not toggle)
		// The UI will show which tags are already applied
		void onAssignTag(tag.id);
	}

	function handleRemoveTagClick(tag: Tag, event: MouseEvent) {
		event.stopPropagation();
		void onRemoveTag(tag.id);
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (!isOpen) return;

		if (event.key === "Escape") {
			event.preventDefault();
			onClose();
			return;
		}

		if (event.key === "ArrowDown") {
			event.preventDefault();
			highlightedIndex = Math.min(highlightedIndex + 1, filteredTags.length - 1);
			return;
		}

		if (event.key === "ArrowUp") {
			event.preventDefault();
			highlightedIndex = Math.max(highlightedIndex - 1, -1);
			return;
		}

		if (event.key === "Enter" && highlightedIndex >= 0) {
			event.preventDefault();
			const tag = filteredTags[highlightedIndex];
			if (tag) {
				handleTagClick(tag);
			}
			return;
		}
	}

	onMount(() => {
		if (typeof window !== "undefined") {
			window.addEventListener("keydown", handleKeyDown);
		}
	});

	onDestroy(() => {
		if (typeof window !== "undefined") {
			window.removeEventListener("keydown", handleKeyDown);
		}
	});
</script>

<svelte:window on:keydown={handleKeyDown} />

{#if isOpen}
	<div
		bind:this={dropdownElement}
		class="fixed z-[150] min-w-[280px] max-w-[320px] rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-xl"
		style="top: {dropdownPosition.top}px; {typeof dropdownPosition.left === 'number'
			? `left: ${dropdownPosition.left}px;`
			: ''} {typeof dropdownPosition.right === 'number'
			? `right: ${dropdownPosition.right}px;`
			: ''} {!isPositioned ? 'opacity: 0; pointer-events: none;' : ''}"
		role="menu"
		aria-label="Bulk tag management"
	>
		<div class="p-3">
			<!-- Search input -->
			<input
				bind:this={searchInputElement}
				type="text"
				bind:value={searchTerm}
				placeholder="Search tags..."
				class="w-full rounded-md border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 placeholder-gray-500 dark:placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30"
				autocomplete="off"
			/>

			<!-- Tag list -->
			<div class="mt-2 max-h-[300px] overflow-y-auto">
				{#if filteredTags.length === 0}
					<div class="py-4 text-center text-sm text-gray-600 dark:text-gray-500">
						{#if searchTerm}
							No tags found matching "{searchTerm}"
						{:else}
							No tags available
						{/if}
					</div>
				{:else}
					<div class="space-y-1">
						{#each filteredTags as tag, index (tag.id)}
							<div
								class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm transition hover:bg-gray-100 dark:hover:bg-gray-800 {highlightedIndex ===
								index
									? 'bg-gray-100 dark:bg-gray-800'
									: ''}"
								role="menuitem"
							>
								<button
									type="button"
									class="flex flex-1 items-center gap-2 text-left cursor-pointer"
									on:click={() => handleTagClick(tag)}
								>
									<span
										class="h-3 w-3 rounded-full"
										style="background-color: {tag.color ?? '#6366f1'}"
									></span>
									<span class="text-gray-900 dark:text-gray-200">{tag.name}</span>
								</button>
								<button
									type="button"
									class="rounded px-2 py-1 text-xs text-gray-600 dark:text-gray-400 transition hover:bg-gray-200 dark:hover:bg-gray-700 hover:text-gray-800 dark:hover:text-gray-200 cursor-pointer"
									on:click={(e) => handleRemoveTagClick(tag, e)}
									title="Remove tag"
								>
									Remove
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Help text -->
			<div
				class="mt-3 border-t border-gray-200 dark:border-gray-700 pt-2 text-xs text-gray-600 dark:text-gray-500"
			>
				Click a tag to apply it, or click "Remove" to remove it
			</div>
		</div>
	</div>
{/if}
