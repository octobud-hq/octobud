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
	import { getContextAtCursor, isValidField, parseQuery } from "$lib/utils/queryParser";
	import { FILTER_FIELDS, getValueSuggestionsForField } from "$lib/constants/filterFields";

	export let value: string = "";
	export let onChange: (value: string) => void = () => {};
	export let placeholder: string = "Filter notifications...";
	export let showValidation: boolean = true;

	let inputElement: HTMLInputElement | null = null;
	let cursorPos: number = 0;
	let showSuggestions = false;
	let suggestions: string[] = [];
	let selectedIndex = 0;
	let suggestionType: "field" | "value" | "none" = "none";
	let suggestionField: string | undefined;
	let dropdownTop = 0;
	let dropdownLeft = 0;
	let dropdownWidth = 0;
	let validationErrors: string[] = [];
	let showValidationTooltip = false;

	$: {
		if (showValidation && value.trim()) {
			const parsed = parseQuery(value);
			validationErrors = parsed.errors;
		} else {
			validationErrors = [];
		}
	}

	function handleInput(event: Event) {
		const target = event.target as HTMLInputElement;
		value = target.value;
		cursorPos = target.selectionStart || 0;
		updateSuggestions();
		onChange(value);
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (!showSuggestions) {
			if (event.key === " " && event.ctrlKey) {
				// Ctrl+Space to trigger suggestions
				event.preventDefault();
				updateSuggestions();
				showSuggestions = true;
			}
			return;
		}

		switch (event.key) {
			case "ArrowDown":
				event.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, suggestions.length - 1);
				break;
			case "ArrowUp":
				event.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, 0);
				break;
			case "Enter":
			case "Tab":
				if (suggestions.length > 0) {
					event.preventDefault();
					selectSuggestion(suggestions[selectedIndex]);
				}
				break;
			case "Escape":
				event.preventDefault();
				showSuggestions = false;
				break;
		}
	}

	function handleClick() {
		if (inputElement) {
			cursorPos = inputElement.selectionStart || 0;
			updateSuggestions();
		}
	}

	function updateDropdownPosition() {
		if (!inputElement) return;

		const rect = inputElement.getBoundingClientRect();
		dropdownTop = rect.bottom + 4; // 4px gap below input
		dropdownLeft = rect.left;
		dropdownWidth = rect.width;
	}

	function updateSuggestions() {
		if (!inputElement) return;

		const context = getContextAtCursor(value, cursorPos);
		suggestionType = context.type;
		suggestionField = context.field;

		if (context.type === "field") {
			// Suggest field names
			const filtered = FILTER_FIELDS.filter(
				(f) =>
					f.value.toLowerCase().startsWith(context.partial.toLowerCase()) ||
					f.description.toLowerCase().includes(context.partial.toLowerCase())
			);
			suggestions = filtered.map((f) => f.value + ":");
			showSuggestions = suggestions.length > 0 && context.partial.length >= 0;
		} else if (context.type === "value" && context.field) {
			// Suggest values for the field
			const fieldValues = getValueSuggestionsForField(context.field);
			if (fieldValues.length > 0) {
				const filtered = fieldValues.filter((v) =>
					v.toLowerCase().startsWith(context.partial.toLowerCase())
				);
				suggestions = filtered;
				showSuggestions = suggestions.length > 0;
			} else {
				suggestions = [];
				showSuggestions = false;
			}
		} else {
			suggestions = [];
			showSuggestions = false;
		}

		selectedIndex = 0;

		// Update dropdown position when showing suggestions
		if (showSuggestions) {
			updateDropdownPosition();
		}
	}

	function selectSuggestion(suggestion: string) {
		if (!inputElement) return;

		const context = getContextAtCursor(value, cursorPos);

		if (context.type === "field") {
			// Replace the field part
			const beforeCursor = value.slice(0, cursorPos);
			const afterCursor = value.slice(cursorPos);

			// Find the start of the current term
			let termStart = beforeCursor.lastIndexOf(" ");
			termStart = termStart === -1 ? 0 : termStart + 1;

			// Handle negation
			const termPrefix = value.slice(termStart, cursorPos);
			const hasNegation = termPrefix.startsWith("-");
			const prefix = hasNegation ? "-" : "";

			// Build new value
			const before = value.slice(0, termStart);
			value = before + prefix + suggestion;
			cursorPos = value.length;

			// Set cursor position after update
			setTimeout(() => {
				if (inputElement) {
					inputElement.setSelectionRange(cursorPos, cursorPos);
					inputElement.focus();
				}
			}, 0);
		} else if (context.type === "value") {
			// Replace the value part
			const beforeCursor = value.slice(0, cursorPos);
			const afterCursor = value.slice(cursorPos);

			// Find the start of the current value (after last comma or colon)
			let valueStart = Math.max(beforeCursor.lastIndexOf(":"), beforeCursor.lastIndexOf(","));
			valueStart += 1; // Move past the delimiter

			// Skip whitespace
			while (valueStart < beforeCursor.length && beforeCursor[valueStart] === " ") {
				valueStart++;
			}

			// Build new value
			const before = value.slice(0, valueStart);
			const after = afterCursor.trimStart();
			value =
				before + suggestion + (after.startsWith(",") || after.startsWith(" ") ? "" : " ") + after;
			cursorPos = (before + suggestion).length;

			// Set cursor position after update
			setTimeout(() => {
				if (inputElement) {
					inputElement.setSelectionRange(cursorPos, cursorPos);
					inputElement.focus();
				}
			}, 0);
		}

		showSuggestions = false;
		onChange(value);
	}

	function handleBlur() {
		// Delay hiding suggestions to allow click events to fire
		setTimeout(() => {
			showSuggestions = false;
		}, 200);
	}

	// Expose focus method for parent components
	export function focus() {
		inputElement?.focus();
	}

	onMount(() => {
		// Update dropdown position on scroll or resize
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

<div class="relative w-full">
	<input
		bind:this={inputElement}
		type="text"
		class="w-full rounded-lg border px-3 text-sm text-gray-900 dark:text-gray-200 outline-none transition focus:ring-2 {validationErrors.length >
		0
			? 'border-red-500 bg-red-50 dark:bg-red-950/20 pr-10 py-4 focus:border-red-500 focus:ring-red-500/30'
			: 'border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 py-4 focus:border-indigo-500 focus:ring-indigo-500/30'}"
		{placeholder}
		{value}
		on:input={handleInput}
		on:keydown={handleKeyDown}
		on:click={handleClick}
		on:blur={handleBlur}
		autocomplete="off"
		spellcheck={false}
	/>

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

	{#if showSuggestions && suggestions.length > 0}
		<div
			class="fixed z-[9999] max-h-60 overflow-auto rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-soft"
			style="top: {dropdownTop}px; left: {dropdownLeft}px; width: {dropdownWidth}px; max-height: min(15rem, calc(100vh - {dropdownTop}px - 1rem));"
		>
			<ul class="py-1">
				{#each suggestions as suggestion, index (suggestion + index.toString())}
					<li>
						<button
							type="button"
							class={`flex w-full items-center justify-between px-3 py-2 text-left text-sm transition ${
								index === selectedIndex
									? "bg-indigo-50 dark:bg-gray-900/60 text-indigo-700 dark:text-indigo-200"
									: "text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-900/40"
							}`}
							on:mouseenter={() => (selectedIndex = index)}
							on:mousedown|preventDefault={() => selectSuggestion(suggestion)}
						>
							<span class="font-medium">{suggestion}</span>
							{#if suggestionType === "field"}
								<span class="text-xs text-gray-600 dark:text-gray-500">
									{FILTER_FIELDS.find((f) => f.value + ":" === suggestion)?.description || ""}
								</span>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>

<style>
	.shadow-soft {
		box-shadow:
			0 10px 15px -3px rgba(0, 0, 0, 0.3),
			0 4px 6px -2px rgba(0, 0, 0, 0.2);
	}
</style>
