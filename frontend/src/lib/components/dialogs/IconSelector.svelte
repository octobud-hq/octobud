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
	import { DEFAULT_VIEW_ICON, VIEW_ICON_SUGGESTIONS } from "$lib/utils/viewIcons";

	export let value: string = DEFAULT_VIEW_ICON;
	export let onChange: (value: string) => void = () => {};
	export let isValid: boolean = true;

	let isOpen = false;
	let dropdownRef: HTMLDivElement | null = null;
	let buttonRef: HTMLButtonElement | null = null;
	let customInput = "";
	let dropdownTop = 0;
	let dropdownLeft = 0;

	function updateDropdownPosition() {
		if (!buttonRef) return;
		const rect = buttonRef.getBoundingClientRect();
		// For absolute positioning, we need to calculate relative to the positioned parent
		// Find the positioned parent (the outer relative container)
		const container = buttonRef.closest(".relative");
		if (container) {
			const containerRect = container.getBoundingClientRect();
			dropdownTop = rect.bottom - containerRect.top + 8; // 8px gap below button
			dropdownLeft = rect.left - containerRect.left;
		} else {
			// Fallback to viewport coordinates if no positioned parent found
			dropdownTop = rect.bottom + 8;
			dropdownLeft = rect.left;
		}
	}

	function toggleDropdown() {
		isOpen = !isOpen;
		if (isOpen) {
			customInput = value.trim() === DEFAULT_VIEW_ICON ? "" : value;
			// Use requestAnimationFrame to ensure position is calculated after layout
			requestAnimationFrame(() => {
				updateDropdownPosition();
			});
		}
	}

	function closeDropdown() {
		isOpen = false;
	}

	function selectSuggestion(emoji: string) {
		customInput = emoji;
	}

	function handleDone() {
		const finalValue = customInput.trim() || DEFAULT_VIEW_ICON;
		onChange(finalValue);
		closeDropdown();
	}

	function handleClickOutside(event: MouseEvent) {
		if (!isOpen) return;

		const target = event.target as Node;
		const clickedOutsideDropdown = dropdownRef && !dropdownRef.contains(target);
		const clickedOutsideButton = buttonRef && !buttonRef.contains(target);

		if (clickedOutsideDropdown && clickedOutsideButton) {
			closeDropdown();
		}
	}

	onMount(() => {
		const handleScroll = () => {
			if (isOpen) {
				requestAnimationFrame(() => {
					updateDropdownPosition();
				});
			}
		};

		const handleResize = () => {
			if (isOpen) {
				requestAnimationFrame(() => {
					updateDropdownPosition();
				});
			}
		};

		// Find all scrollable ancestors and listen to their scroll events
		const scrollContainers: (Element | Window)[] = [window];
		if (buttonRef) {
			let parent: Element | null = buttonRef.parentElement;
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

		document.addEventListener("mousedown", handleClickOutside);
		// Add scroll listeners to all scrollable containers
		scrollContainers.forEach((container) => {
			container.addEventListener("scroll", handleScroll, true);
		});
		window.addEventListener("resize", handleResize);

		return () => {
			document.removeEventListener("mousedown", handleClickOutside);
			scrollContainers.forEach((container) => {
				container.removeEventListener("scroll", handleScroll, true);
			});
			window.removeEventListener("resize", handleResize);
		};
	});

	$: displayIcon = value.trim() || DEFAULT_VIEW_ICON;
</script>

<div class="relative">
	<button
		bind:this={buttonRef}
		type="button"
		class={`flex w-[50px] items-center justify-center rounded-full border py-1 text-xl transition cursor-pointer ${
			isValid
				? "border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 hover:bg-gray-100 dark:hover:bg-gray-700"
				: "border-rose-500 bg-gray-50 dark:bg-gray-900"
		}`}
		on:click={toggleDropdown}
		aria-label="Select icon"
	>
		{displayIcon}
	</button>

	{#if isOpen}
		<div
			bind:this={dropdownRef}
			class="absolute w-72 rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-4 shadow-xl z-[9999]"
			style="top: {dropdownTop}px; left: {dropdownLeft}px;"
		>
			<div class="space-y-4">
				<!-- Custom input -->
				<div class="space-y-2">
					<label
						class="text-xs font-medium text-gray-600 dark:text-gray-400"
						for="custom-icon-input"
					>
						Custom emoji
					</label>
					<input
						id="custom-icon-input"
						type="text"
						class="w-full rounded-lg border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2.5 text-sm text-gray-900 dark:text-gray-200 outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/30"
						placeholder="✨"
						bind:value={customInput}
						autocapitalize="off"
						autocomplete="off"
						spellcheck={false}
					/>
					<p class="text-xs text-gray-600 dark:text-gray-500">Enter a single emoji</p>
				</div>

				<!-- Suggested emojis -->
				<div class="space-y-2">
					<p class="text-xs font-medium text-gray-600 dark:text-gray-400">Suggested emojis</p>
					<div class="grid grid-cols-6 gap-2">
						{#each VIEW_ICON_SUGGESTIONS as suggestion (suggestion)}
							<button
								type="button"
								class={`flex h-10 w-10 items-center justify-center rounded-lg border text-xl transition cursor-pointer ${
									customInput === suggestion
										? "border-indigo-500 bg-indigo-50 dark:bg-gray-500/10 text-indigo-700 dark:text-indigo-200"
										: "border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 text-gray-900 dark:text-gray-200 hover:border-indigo-500/40"
								}`}
								on:click={() => selectSuggestion(suggestion)}
								aria-label={`Use ${suggestion} icon`}
							>
								{suggestion}
							</button>
						{/each}
					</div>
				</div>

				<!-- Done button -->
				<button
					type="button"
					class="w-full rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700 cursor-pointer"
					on:click={handleDone}
				>
					Done
				</button>
			</div>
		</div>
	{/if}
</div>
