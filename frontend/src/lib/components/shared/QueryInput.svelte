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
	import { getContextAtCursor, parseQuery } from "$lib/utils/queryParser";
	import { FILTER_FIELDS, getValueSuggestionsForField } from "$lib/constants/filterFields";
	import { tokenizeForSyntax, type SyntaxToken } from "$lib/utils/querySyntaxTokenizer";

	export let value: string = "";
	export let onChange: (value: string) => void = () => {};
	export let placeholder: string = "e.g., repo:owner/name type:issue";
	export let showValidation: boolean = true;
	export let showDropdownOnFocus: boolean = true;
	export let inputId: string = "";

	let inputElement: HTMLInputElement | null = null;
	let inputContainerElement: HTMLDivElement | null = null;
	let cursorPos: number = 0;
	let showSuggestions = false;
	let suggestions: string[] = [];
	let selectedIndex = 0;
	let suggestionType: "field" | "value" | "none" = "none";
	let suggestionField: string | undefined;
	let dropdownTop = 0;
	let dropdownLeft = 0;
	let dropdownWidth = 0;
	let dropdownTransformX = 0;
	let dropdownTransformY = 0;
	let validationErrors: string[] = [];
	let highlightLayerElement: HTMLDivElement | null = null;

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
		onChange(value);
	}

	function handleDoneEditing() {
		showSuggestions = false;
		inputElement?.blur();
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
				// If at "Set query" item (-1), go to first suggestion (0) if available
				// Otherwise stay at -1
				if (selectedIndex === -1) {
					if (suggestions.length > 0) {
						selectedIndex = 0;
					}
					// If no suggestions, stay at -1
				} else {
					selectedIndex = Math.min(selectedIndex + 1, suggestions.length - 1);
				}
				break;
			case "ArrowUp":
				event.preventDefault();
				// If at first suggestion (0) and there's a value, go to "Set query" item (-1)
				// Otherwise decrement normally
				if (selectedIndex === 0 && value.trim()) {
					selectedIndex = -1;
				} else {
					selectedIndex = Math.max(selectedIndex - 1, 0);
				}
				break;
			case "Enter":
			case "Tab":
				event.preventDefault();
				if (selectedIndex === -1) {
					// Close dropdown and blur if "Set query" is selected
					handleDoneEditing();
				} else if (
					suggestions.length > 0 &&
					selectedIndex >= 0 &&
					selectedIndex < suggestions.length
				) {
					// Select suggestion if a filter is selected
					selectSuggestion(suggestions[selectedIndex]);
				}
				break;
			case "Escape":
				event.preventDefault();
				event.stopPropagation(); // Prevent the event from bubbling up to close the modal
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

	function handleFocus() {
		if (showDropdownOnFocus) {
			updateSuggestions();
		}
	}

	function updateDropdownPosition() {
		if (!inputContainerElement || typeof window === "undefined") return;

		// Get the input container's viewport coordinates
		// getBoundingClientRect() returns coordinates relative to the viewport
		const containerRect = inputContainerElement.getBoundingClientRect();

		// Position using viewport coordinates
		dropdownTop = containerRect.bottom + 8;
		dropdownLeft = containerRect.left;
		dropdownWidth = containerRect.width;

		// Check if we're inside a modal (which has a transform that creates a new containing block)
		const modalDialog = inputContainerElement.closest('[role="dialog"]');
		if (modalDialog) {
			// The modal's transform creates a coordinate system centered at viewport center
			// Calculate the offset needed: distance from viewport center to modal's top-left corner
			const modalRect = (modalDialog as HTMLElement).getBoundingClientRect();
			const viewportCenterX = window.innerWidth / 2;
			const viewportCenterY = window.innerHeight / 2;

			const offsetX = viewportCenterX - modalRect.width / 2;
			const offsetY = viewportCenterY - modalRect.height / 2;

			// Apply the calculated offset to compensate for modal's coordinate system
			dropdownTransformX = -offsetX;
			dropdownTransformY = -offsetY;
		} else {
			// Not in a modal, no transform needed
			dropdownTransformX = 0;
			dropdownTransformY = 0;
		}
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
			// Show suggestions if we have any, or always show all fields if no matches
			if (suggestions.length === 0) {
				suggestions = FILTER_FIELDS.map((f) => f.value + ":");
			}
		} else if (context.type === "value" && context.field) {
			suggestionType = "value";
			const fieldValues = getValueSuggestionsForField(context.field);
			if (fieldValues.length > 0) {
				const filtered = fieldValues.filter((v) =>
					v.toLowerCase().startsWith(context.partial.toLowerCase())
				);
				suggestions = filtered.length > 0 ? filtered : fieldValues;
			} else {
				// No values for this field, show all filter fields
				suggestionType = "field";
				suggestions = FILTER_FIELDS.map((f) => f.value + ":");
			}
		} else {
			// Show filter suggestions when not in a specific context
			suggestionType = "field";
			suggestions = FILTER_FIELDS.map((f) => f.value + ":");
		}

		// Always show suggestions when dropdown should be shown
		showSuggestions = showDropdownOnFocus;
		// Initialize selectedIndex: -1 for "Set query" item if there's a value, otherwise 0
		selectedIndex = value.trim() ? -1 : 0;

		// Use requestAnimationFrame to ensure position is calculated after layout
		requestAnimationFrame(() => {
			updateDropdownPosition();
		});
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
		const handleScrollEvent = () => {
			if (showSuggestions) {
				updateDropdownPosition();
			}
		};

		const handleResize = () => {
			if (showSuggestions) {
				updateDropdownPosition();
			}
		};

		// Find all scrollable ancestors and listen to their scroll events
		const scrollContainers: (Element | Window)[] = [window];
		if (inputElement) {
			let parent: Element | null = inputElement.parentElement;
			while (parent) {
				const style = window.getComputedStyle(parent);
				if (
					style.overflow === "auto" ||
					style.overflow === "scroll" ||
					style.overflowY === "auto" ||
					style.overflowY === "scroll"
				) {
					scrollContainers.push(parent);
				}
				parent = parent.parentElement;
			}
		}

		// Add scroll listeners to all scrollable containers
		scrollContainers.forEach((container) => {
			container.addEventListener("scroll", handleScrollEvent, true);
		});
		window.addEventListener("resize", handleResize);

		return () => {
			showSuggestions = false;
			scrollContainers.forEach((container) => {
				container.removeEventListener("scroll", handleScrollEvent, true);
			});
			window.removeEventListener("resize", handleResize);
		};
	});
</script>

<div class="relative w-full">
	<!-- Main input container -->
	<div
		bind:this={inputContainerElement}
		class="relative flex items-center gap-2 rounded-lg border px-4 py-1.5 text-sm text-gray-900 dark:text-gray-200 transition focus-within:ring-2 {validationErrors.length >
		0
			? 'border-red-500 bg-red-50 dark:bg-red-950/20 focus-within:border-red-500 focus-within:ring-red-500/30'
			: 'border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 focus-within:border-indigo-500 focus-within:ring-indigo-500/30'}"
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
				id={inputId}
				type="text"
				class="relative w-full appearance-none border-none bg-transparent p-0 text-sm leading-normal text-gray-900/10 dark:text-gray-200/10 placeholder:text-gray-500 dark:placeholder:text-gray-500 caret-gray-900 dark:caret-gray-200 outline-none focus:ring-0 disabled:cursor-not-allowed"
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

		<slot name="actions" />

		<!-- Autocomplete suggestions dropdown -->
		{#if showSuggestions && (suggestions.length > 0 || value.trim())}
			<div
				class="fixed z-[9999] max-h-60 overflow-auto rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 shadow-soft"
				style="top: {dropdownTop}px; left: {dropdownLeft}px; width: {dropdownWidth}px; transform: translate({dropdownTransformX}px, {dropdownTransformY}px);"
			>
				<slot name="dropdown-header" />

				<!-- Set query section (when there's a value) -->
				{#if value.trim()}
					<div class="border-b border-gray-200 dark:border-gray-800 py-1">
						<div class="px-3 py-1">
							<div
								class="text-[10px] font-semibold uppercase tracking-wider text-gray-600 dark:text-gray-500"
							>
								Query
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
							on:mousedown|preventDefault={handleDoneEditing}
						>
							<span class="flex-1 truncate font-medium text-gray-700 dark:text-gray-200"
								>{value}</span
							>
							<span class="ml-2 text-xs text-gray-600 dark:text-gray-500">Set query</span>
						</button>
					</div>
				{/if}

				<!-- Suggested Filters section -->
				<div class="py-1">
					<div class="flex items-center justify-between px-3 py-1">
						<div
							class="text-[10px] font-semibold uppercase tracking-wider text-gray-600 dark:text-gray-500"
						>
							Suggested Filters
						</div>
						<slot name="dropdown-actions" />
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
			</div>
		{/if}
	</div>
</div>

<style>
	.shadow-soft {
		box-shadow:
			0 10px 15px -3px rgba(0, 0, 0, 0.3),
			0 4px 6px -2px rgba(0, 0, 0, 0.2);
	}
</style>
