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
	import { getContextAtCursor, parseQuery, type QueryTerm } from "$lib/utils/queryParser";
	import { FILTER_FIELDS, getValueSuggestionsForField } from "$lib/constants/filterFields";
	import SyntaxGuideDialog from "../dialogs/SyntaxGuideDialog.svelte";
	import { fetchTags } from "$lib/api/tags";
	import type { Tag } from "$lib/api/types";
	import { tokenizeForSyntax, type SyntaxToken } from "$lib/utils/querySyntaxTokenizer";

	export let value: string = "";
	export let onChange: (value: string) => void = () => {}; // Called on every keystroke for controlled input
	export let onSubmit: (value: string) => void = () => {}; // Called when user submits search
	export let placeholder: string = "Search or filter like repo:owner/name";
	export let showValidation: boolean = true;

	// Save button props
	export let showSaveButton: boolean = false;
	export let canUpdateView: boolean = false; // If true, shows dropdown; if false, direct save
	export let onSaveAsNew: () => void = () => {};
	export let onUpdateView: () => void = () => {};

	let inputElement: HTMLInputElement | null = null;
	let containerElement: HTMLDivElement | null = null;
	let cursorPos: number = 0;
	let showSuggestions = false;
	let suggestions: string[] = [];
	let suggestionHints: string[] = []; // Hints for suggestions (e.g., tag names)
	let selectedIndex = 0;
	let suggestionType: "field" | "value" | "none" = "none";
	let suggestionField: string | undefined;
	let dropdownTop = 0;
	let dropdownLeft = 0;
	let dropdownWidth = 0;
	let validationErrors: string[] = [];
	let showValidationTooltip = false;
	let highlightLayerElement: HTMLDivElement | null = null;
	let saveDropdownOpen = false;
	let syntaxGuideOpen = false;
	let availableTags: Tag[] = [];

	// Load tags on mount
	onMount(async () => {
		try {
			availableTags = await fetchTags();
		} catch (error) {
			console.error("Failed to fetch tags for suggestions:", error);
		}
	});

	$: {
		if (showValidation && value.trim()) {
			const parsed = parseQuery(value);
			validationErrors = parsed.errors;
		} else {
			validationErrors = [];
		}
	}

	// Tokenize the input for syntax highlighting
	$: syntaxTokens = tokenizeForSyntax(value);

	// Sync scroll position between input and highlight layer
	function handleScroll(event: Event) {
		if (highlightLayerElement && event.target === inputElement) {
			highlightLayerElement.scrollLeft = inputElement.scrollLeft;
		}
	}

	function handleInput(event: Event) {
		const target = event.target as HTMLInputElement;
		value = target.value;
		cursorPos = target.selectionStart || 0;
		updateSuggestions();
		onChange(value); // Update controlled value but don't submit
	}

	function handleSubmit() {
		showSuggestions = false;
		onSubmit(value);
		inputElement?.blur();
	}

	function handleSaveClick(event: MouseEvent) {
		event.stopPropagation();
		// Always toggle dropdown (both inbox and custom view)
		saveDropdownOpen = !saveDropdownOpen;
	}

	function handleUpdateViewClick() {
		onUpdateView();
		saveDropdownOpen = false;
	}

	function handleSaveAsNewClick() {
		onSaveAsNew();
		saveDropdownOpen = false;
	}

	function handleKeyDown(event: KeyboardEvent) {
		// Handle Escape to unfocus input
		if (event.key === "Escape") {
			if (showSuggestions) {
				event.preventDefault();
				showSuggestions = false;
			} else if (inputElement) {
				event.preventDefault();
				inputElement.blur();
			}
			return;
		}

		if (!showSuggestions) {
			if (event.key === " " && event.ctrlKey) {
				// Ctrl+Space to trigger suggestions
				event.preventDefault();
				updateSuggestions();
				showSuggestions = true;
			} else if (event.key === "Enter") {
				// Submit search when Enter is pressed and dropdown is closed
				event.preventDefault();
				handleSubmit();
			}
			return;
		}

		switch (event.key) {
			case "ArrowDown":
				event.preventDefault();
				// If at search item (-1), go to first suggestion (0)
				// Otherwise increment normally
				if (selectedIndex === -1) {
					selectedIndex = 0;
				} else {
					selectedIndex = Math.min(selectedIndex + 1, suggestions.length - 1);
				}
				break;
			case "ArrowUp":
				event.preventDefault();
				// If at first suggestion (0) and there's a value, go to search item (-1)
				// Otherwise decrement normally
				if (selectedIndex === 0 && value.trim()) {
					selectedIndex = -1;
				} else {
					selectedIndex = Math.max(selectedIndex - 1, 0);
				}
				break;
			case "Enter":
				event.preventDefault();
				if (selectedIndex === -1) {
					// Submit search if search item is selected
					handleSubmit();
				} else if (suggestions.length > 0) {
					// Select suggestion if a filter is selected
					selectSuggestion(suggestions[selectedIndex]);
				}
				break;
			case "Tab":
				if (selectedIndex >= 0 && suggestions.length > 0) {
					event.preventDefault();
					selectSuggestion(suggestions[selectedIndex]);
				}
				break;
		}
	}

	function handleClick() {
		if (inputElement) {
			cursorPos = inputElement.selectionStart || 0;
			updateSuggestions();
		}
	}

	function handleFocus() {
		// Always show dropdown when focusing
		updateSuggestions();
	}

	function updateDropdownPosition() {
		if (!inputElement) return;

		const rect = inputElement.getBoundingClientRect();
		dropdownTop = rect.bottom + 8;
		dropdownLeft = rect.left;
		dropdownWidth = rect.width;
	}

	function updateSuggestions() {
		if (!inputElement) return;

		const context = getContextAtCursor(value, cursorPos);
		suggestionField = context.field;

		if (context.type === "field") {
			suggestionType = "field";
			const filtered = FILTER_FIELDS.filter(
				(f) =>
					f.value.toLowerCase().startsWith(context.partial.toLowerCase()) ||
					f.description.toLowerCase().includes(context.partial.toLowerCase())
			);
			suggestions = filtered.map((f) => f.value + ":");
			suggestionHints = filtered.map((f) => f.description);
			// Show suggestions if we have any, or always show all fields if no matches
			if (suggestions.length === 0) {
				suggestions = FILTER_FIELDS.map((f) => f.value + ":");
				suggestionHints = FILTER_FIELDS.map((f) => f.description);
			}
		} else if (context.type === "value" && context.field) {
			suggestionType = "value";

			// Special handling for tags field
			if (context.field === "tags") {
				const filtered = availableTags.filter(
					(tag) =>
						tag.slug.toLowerCase().includes(context.partial.toLowerCase()) ||
						tag.name.toLowerCase().includes(context.partial.toLowerCase())
				);
				suggestions = filtered.map((tag) => tag.slug);
				suggestionHints = filtered.map((tag) => tag.name);
			} else {
				// Regular field value suggestions
				const fieldValues = getValueSuggestionsForField(context.field);
				if (fieldValues.length > 0) {
					const filtered = fieldValues.filter((v) =>
						v.toLowerCase().startsWith(context.partial.toLowerCase())
					);
					suggestions = filtered.length > 0 ? filtered : fieldValues;
					suggestionHints = []; // No hints for regular values
				} else {
					// No values for this field, show all filter fields
					suggestionType = "field";
					suggestions = FILTER_FIELDS.map((f) => f.value + ":");
					suggestionHints = FILTER_FIELDS.map((f) => f.description);
				}
			}
		} else {
			// Show filter suggestions when not in a specific context
			suggestionType = "field";
			suggestions = FILTER_FIELDS.map((f) => f.value + ":");
			suggestionHints = FILTER_FIELDS.map((f) => f.description);
		}

		// Always show suggestions
		showSuggestions = true;

		// Start with search item selected if there's a value, otherwise first suggestion
		selectedIndex = value.trim() ? -1 : 0;

		updateDropdownPosition();
	}

	function selectSuggestion(suggestion: string) {
		if (!inputElement) return;

		const context = getContextAtCursor(value, cursorPos);

		// Use suggestionType instead of context.type for better reliability
		if (suggestionType === "field") {
			const beforeCursor = value.slice(0, cursorPos);
			const afterCursor = value.slice(cursorPos);

			let termStart = beforeCursor.lastIndexOf(" ");
			termStart = termStart === -1 ? 0 : termStart + 1;

			const termPrefix = value.slice(termStart, cursorPos);
			const hasNegation = termPrefix.startsWith("-");
			const prefix = hasNegation ? "-" : "";

			const before = value.slice(0, termStart);
			value = before + prefix + suggestion;
			cursorPos = value.length;

			setTimeout(() => {
				if (inputElement) {
					inputElement.setSelectionRange(cursorPos, cursorPos);
					inputElement.focus();
				}
			}, 0);
		} else if (suggestionType === "value") {
			const beforeCursor = value.slice(0, cursorPos);
			const afterCursor = value.slice(cursorPos);

			let valueStart = Math.max(beforeCursor.lastIndexOf(":"), beforeCursor.lastIndexOf(","));
			valueStart += 1;

			while (valueStart < beforeCursor.length && beforeCursor[valueStart] === " ") {
				valueStart++;
			}

			const before = value.slice(0, valueStart);
			const after = afterCursor.trimStart();
			value =
				before + suggestion + (after.startsWith(",") || after.startsWith(" ") ? "" : " ") + after;
			cursorPos = (before + suggestion).length;

			setTimeout(() => {
				if (inputElement) {
					inputElement.setSelectionRange(cursorPos, cursorPos);
					inputElement.focus();
				}
			}, 0);
		}

		// Keep suggestions open and update them
		onChange(value);
		updateSuggestions();
		// Reset to "Submit search" item
		selectedIndex = -1;
	}

	function handleBlur() {
		setTimeout(() => {
			showSuggestions = false;
		}, 200);
	}

	export function focus() {
		inputElement?.focus();
	}

	onMount(() => {
		const handleScroll = () => {
			if (showSuggestions) {
				updateDropdownPosition();
			}
		};

		const handleResize = () => {
			if (showSuggestions) {
				updateDropdownPosition();
			}
		};

		window.addEventListener("scroll", handleScroll, true);
		window.addEventListener("resize", handleResize);

		return () => {
			showSuggestions = false;
			window.removeEventListener("scroll", handleScroll, true);
			window.removeEventListener("resize", handleResize);
		};
	});
</script>

<svelte:window
	on:click={(e) => {
		if (saveDropdownOpen && e.target instanceof Element) {
			const saveButton = e.target.closest(".relative");
			const isInsideSaveDropdown = saveButton?.querySelector('[title*="Save"]');
			if (!isInsideSaveDropdown) {
				saveDropdownOpen = false;
			}
		}
	}}
/>

<div class="relative w-full" bind:this={containerElement}>
	<!-- Main input container -->
	<div
		class="flex items-center gap-2 rounded-lg border px-4 py-1.5 text-sm text-gray-900 dark:text-gray-200 transition focus-within:ring-2 {validationErrors.length >
		0
			? 'border-red-500 bg-red-50 dark:bg-red-950/20 focus-within:border-red-500 focus-within:ring-red-500/30'
			: 'border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 focus-within:border-indigo-500 focus-within:ring-indigo-500/30'}"
	>
		<div class="relative min-w-0 flex-1">
			<!-- Syntax highlighting background layer -->
			<div
				bind:this={highlightLayerElement}
				class="pointer-events-none absolute inset-0 flex items-center overflow-hidden whitespace-pre text-sm leading-normal"
				aria-hidden="true"
			>
				<div style="white-space: pre;">
					<!--
                    -->{#each syntaxTokens as token, i (token.text + i)}<!--
                        -->{#if token.type === "key"}<!--
                            --><span
								class="text-blue-600 dark:text-blue-500">{token.text}</span
							><!--
                        -->{:else if token.type === "negation"}<!--
                            --><span
								class="text-red-600 dark:text-red-400">{token.text}</span
							><!--
                        -->{:else if token.type === "colon"}<!--
                            --><span
								class="text-gray-600 dark:text-gray-400">{token.text}</span
							><!--
                        -->{:else if token.type === "value"}<!--
                            --><span
								class="text-blue-400 dark:text-blue-300">{token.text}</span
							><!--
                        -->{:else if token.type === "operator"}<!--
                            --><span
								class="text-purple-600 dark:text-purple-400">{token.text}</span
							><!--
                        -->{:else if token.type === "paren"}<!--
                            --><span
								class="text-yellow-600 dark:text-yellow-400">{token.text}</span
							><!--
                        -->{:else if token.type === "whitespace"}<!--
                            --><span
								>{token.text}</span
							><!--
                        -->{:else}<!--
                            --><span
								class="text-gray-900 dark:text-gray-200">{token.text}</span
							><!--
                        -->{/if}<!--
                    -->{/each}<!--
                -->
				</div>
			</div>

			<!-- Actual input (semi-transparent text) -->
			<input
				bind:this={inputElement}
				type="text"
				class="relative w-full appearance-none border-none bg-transparent p-0 text-sm leading-normal text-gray-900/10 dark:text-gray-200/10 caret-gray-900 dark:caret-gray-200 outline-none focus:ring-0 disabled:cursor-not-allowed"
				{placeholder}
				{value}
				on:input={handleInput}
				on:keydown={handleKeyDown}
				on:click={handleClick}
				on:focus={handleFocus}
				on:blur={handleBlur}
				on:scroll={handleScroll}
				autocomplete="off"
				spellcheck={false}
			/>
		</div>

		<!-- Save button (only shown when query is valid) -->
		{#if showSaveButton && validationErrors.length === 0}
			<div class="relative flex-shrink-0">
				<button
					type="button"
					class="text-xs font-medium text-blue-500 transition hover:text-blue-300 cursor-pointer"
					on:click={handleSaveClick}
					title={canUpdateView ? "Save query" : "Save as new view"}
				>
					Save
				</button>

				<!-- Dropdown menu (always show when open) -->
				{#if saveDropdownOpen}
					<div
						class="absolute right-0 z-[10000] mt-2 w-56 rounded-2xl border border-gray-200 dark:border-gray-800 px-1 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-gray-200 shadow-soft"
					>
						<div class="py-1">
							<!-- Update this view option (only in custom views) -->
							{#if canUpdateView}
								<button
									type="button"
									class="flex w-full cursor-pointer items-center px-4 py-2.5 text-left transition hover:bg-gray-100 dark:hover:bg-gray-800 rounded-xl"
									on:click={handleUpdateViewClick}
								>
									<div>
										<div class="font-medium">Update this view</div>
										<div class="text-xs text-gray-600 dark:text-gray-400">
											Save changes to the current view
										</div>
									</div>
								</button>
							{/if}

							<!-- Save as new view option (always show) -->
							<button
								type="button"
								class="flex w-full cursor-pointer items-center px-4 py-2.5 text-left transition hover:bg-gray-100 dark:hover:bg-gray-800 rounded-xl"
								on:click={handleSaveAsNewClick}
							>
								<div>
									<div class="font-medium">Save as new view</div>
									<div class="text-xs text-gray-600 dark:text-gray-400">
										Create a new view with this query
									</div>
								</div>
							</button>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Validation errors -->
	{#if validationErrors.length > 0}
		<div
			class="absolute right-3 top-1/2 -translate-y-1/2 cursor-help"
			role="button"
			tabindex="-1"
			on:mouseenter={() => (showValidationTooltip = true)}
			on:mouseleave={() => (showValidationTooltip = false)}
			on:focus={() => (showValidationTooltip = true)}
			on:blur={() => (showValidationTooltip = false)}
		>
			<svg class="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>

			{#if showValidationTooltip}
				<div
					class="absolute right-0 top-full z-[10000] mt-2 w-80 rounded-lg border border-red-700 bg-red-950 p-3 shadow-lg"
				>
					<div class="space-y-1">
						<p class="text-xs font-semibold text-red-300">Query Validation Errors:</p>
						{#each validationErrors as error (error)}
							<p class="text-xs text-red-400">• {error}</p>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Autocomplete suggestions dropdown -->
	{#if showSuggestions}
		<div
			class="fixed z-[9999] max-h-60 overflow-auto rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-soft"
			style="top: {dropdownTop}px; left: {dropdownLeft}px; width: {dropdownWidth}px; max-height: min(15rem, calc(100vh - {dropdownTop}px - 1rem));"
		>
			<!-- Search section (when there's a value) -->
			{#if value.trim()}
				<div class="border-b border-gray-200 dark:border-gray-800 py-1">
					<div class="px-3 py-1">
						<div
							class="text-[10px] font-semibold uppercase tracking-wider text-gray-600 dark:text-gray-500"
						>
							Search
						</div>
					</div>
					<button
						type="button"
						class={`flex w-full items-center justify-between px-3 py-2 text-left text-sm transition ${
							selectedIndex === -1
								? "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-200"
								: "text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-900/40"
						}`}
						on:mouseenter={() => (selectedIndex = -1)}
						on:mousedown|preventDefault={handleSubmit}
					>
						<span class="flex-1 truncate font-medium text-gray-700 dark:text-gray-200">{value}</span
						>
						<span class="ml-2 text-xs text-gray-600 dark:text-gray-500">Submit search</span>
					</button>
				</div>
			{/if}

			<!-- Suggested Filters section -->
			{#if suggestions.length > 0}
				<div class="py-1">
					<div class="flex items-center justify-between px-3 py-1">
						<div
							class="text-[10px] font-semibold uppercase tracking-wider text-gray-600 dark:text-gray-500"
						>
							Suggested Filters
						</div>
						<button
							type="button"
							class="text-[11px] font-medium text-indigo-400 transition hover:text-indigo-300"
							on:mousedown|preventDefault={() => (syntaxGuideOpen = true)}
						>
							Query syntax
						</button>
					</div>
					<ul>
						{#each suggestions as suggestion, index (suggestion + index.toString())}
							<li>
								<button
									type="button"
									class={`flex w-full items-center justify-between px-3 py-2 text-left text-sm transition ${
										index === selectedIndex
											? "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-200"
											: "text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-900/40"
									}`}
									on:mouseenter={() => (selectedIndex = index)}
									on:mousedown|preventDefault={() => selectSuggestion(suggestion)}
								>
									<span class="font-medium">{suggestion}</span>
									{#if suggestionHints[index]}
										<span class="text-xs text-gray-600 dark:text-gray-500">
											{suggestionHints[index]}
										</span>
									{/if}
								</button>
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		</div>
	{/if}
</div>

<SyntaxGuideDialog bind:open={syntaxGuideOpen} onClose={() => (syntaxGuideOpen = false)} />

<style>
	.shadow-soft {
		box-shadow:
			0 10px 15px -3px rgba(0, 0, 0, 0.3),
			0 4px 6px -2px rgba(0, 0, 0, 0.2);
	}
</style>
