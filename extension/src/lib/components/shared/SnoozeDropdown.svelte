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

	import type { Notification } from "$lib/api/types";
	import { onMount, onDestroy, tick } from "svelte";

	export let notification: Notification;
	export let isOpen: boolean = false;
	export let onSnooze: (until: string) => void = () => {};
	export let onUnsnooze: () => void = () => {};
	export let onOpenCustomDialog: () => void = () => {};
	export let onClose: () => void = () => {};
	export let forceShowUnsnooze: boolean = false; // For bulk operations

	let dropdownElement: HTMLDivElement | null = null;
	let buttonElement: HTMLButtonElement | null = null;
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

	export function setButtonElement(element: HTMLButtonElement | null) {
		buttonElement = element;
	}

	// Reset position when dropdown closes
	$: if (!isOpen) {
		isPositioned = false;
	}

	// Calculate dropdown position based on button position
	async function calculatePosition() {
		if (!dropdownElement || !buttonElement || typeof window === "undefined") {
			return;
		}

		// Wait for the dropdown to be fully rendered
		await tick();

		const buttonRect = buttonElement.getBoundingClientRect();
		const dropdownWidth = 220; // min-w-[220px]
		const dropdownHeight = dropdownElement.scrollHeight || 200; // estimated height
		const viewportWidth = window.innerWidth;
		const viewportHeight = window.innerHeight;
		const padding = 8; // space between button and dropdown

		// Calculate initial position below the button
		let top = buttonRect.bottom + padding;
		let left: number | "auto" = buttonRect.left;
		let right: number | "auto" = "auto";

		// Check if dropdown would overflow on the right
		const wouldOverflowRight = (left as number) + dropdownWidth > viewportWidth - 20;
		if (wouldOverflowRight) {
			// Align to the right edge of the button instead
			right = viewportWidth - buttonRect.right;
			left = buttonRect.right - dropdownWidth;
		}

		// Check if dropdown would overflow on the bottom
		const wouldOverflowBottom = top + dropdownHeight > viewportHeight - 20;
		if (wouldOverflowBottom) {
			// Position above the button instead
			top = buttonRect.top - dropdownHeight - padding;
		}

		// Ensure dropdown doesn't go off the top
		if (top < 10) {
			top = 10;
		}

		// Ensure dropdown doesn't go off the left
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

	// Watch for the element to be bound
	$: if (dropdownElement && buttonElement && isOpen && typeof window !== "undefined") {
		void calculatePosition();
	}

	interface SnoozeOption {
		id: number;
		label: string;
		value: string | null;
		isCustom?: boolean;
	}

	function calculateLaterToday(): string | null {
		const now = new Date();
		const currentHour = now.getHours();

		// Only show if before 6pm
		if (currentHour >= 18) {
			return null;
		}

		const laterToday = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 18, 0, 0, 0);
		return laterToday.toISOString();
	}

	function calculateTomorrow(): string {
		const now = new Date();
		const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 8, 0, 0, 0);
		return tomorrow.toISOString();
	}

	function calculateNextWeek(): string {
		const now = new Date();
		const currentDay = now.getDay(); // 0 = Sunday, 1 = Monday, etc.

		// Calculate days until next Monday
		// If today is Monday (1), we want next Monday (7 days away)
		// If today is Sunday (0), we want next Monday (1 day away)
		const daysUntilMonday = currentDay === 0 ? 1 : currentDay === 1 ? 7 : 8 - currentDay;

		const nextMonday = new Date(
			now.getFullYear(),
			now.getMonth(),
			now.getDate() + daysUntilMonday,
			8,
			0,
			0,
			0
		);
		return nextMonday.toISOString();
	}

	function calculateLaterThisWeek(): string {
		const now = new Date();
		const currentDay = now.getDay(); // 0 = Sunday, 1 = Monday, etc.

		// Calculate days until Wednesday
		// If today is Sunday (0), Wednesday is 3 days away
		// If today is Monday (1), Wednesday is 2 days away
		// If today is Tuesday (2), Wednesday is 1 day away
		// If today is Wednesday (3) or later, this shouldn't be called
		const daysUntilWednesday = currentDay === 0 ? 3 : currentDay === 1 ? 2 : 1;

		const wednesday = new Date(
			now.getFullYear(),
			now.getMonth(),
			now.getDate() + daysUntilWednesday,
			8,
			0,
			0,
			0
		);
		return wednesday.toISOString();
	}

	function formatOptionLabel(option: SnoozeOption): string {
		if (!option.value) return option.label;

		const date = new Date(option.value);
		const now = new Date();

		// Check if it's today
		const isToday =
			date.getDate() === now.getDate() &&
			date.getMonth() === now.getMonth() &&
			date.getFullYear() === now.getFullYear();

		// Check if it's tomorrow
		const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
		const isTomorrow =
			date.getDate() === tomorrow.getDate() &&
			date.getMonth() === tomorrow.getMonth() &&
			date.getFullYear() === tomorrow.getFullYear();

		if (isToday) {
			return `Later today (${date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })})`;
		} else if (isTomorrow) {
			return `Tomorrow (${date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })})`;
		} else {
			// Next week - show day name
			const dayName = date.toLocaleDateString([], { weekday: "long" });
			return `${dayName} (${date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })})`;
		}
	}

	function buildOptions(): SnoozeOption[] {
		const options: SnoozeOption[] = [];
		let optionId = 1;

		const laterToday = calculateLaterToday();
		if (laterToday) {
			options.push({
				id: optionId++,
				label: "Later today (6pm)",
				value: laterToday,
			});
		}

		const tomorrow = calculateTomorrow();
		const nextWeek = calculateNextWeek();

		// Check if tomorrow and next week are the same day
		const tomorrowDate = new Date(tomorrow);
		const nextWeekDate = new Date(nextWeek);
		const isSameDay =
			tomorrowDate.getDate() === nextWeekDate.getDate() &&
			tomorrowDate.getMonth() === nextWeekDate.getMonth() &&
			tomorrowDate.getFullYear() === nextWeekDate.getFullYear();

		options.push({
			id: optionId++,
			label: "Tomorrow (8am)",
			value: tomorrow,
		});

		if (isSameDay) {
			// If tomorrow and next week are the same, show "Later this week" for Wednesday
			options.push({
				id: optionId++,
				label: "Later this week (Wednesday 8am)",
				value: calculateLaterThisWeek(),
			});
		} else {
			options.push({
				id: optionId++,
				label: "Next week (Monday 8am)",
				value: nextWeek,
			});
		}

		options.push({
			id: optionId++,
			label: "Custom date...",
			value: null,
			isCustom: true,
		});

		return options;
	}

	let options: SnoozeOption[] = buildOptions();
	$: if (isOpen) {
		options = buildOptions();
	}
	$: isSnoozed =
		forceShowUnsnooze || (notification.snoozedUntil && notification.snoozedUntil !== null);

	function handleOptionClick(option: SnoozeOption) {
		if (option.isCustom) {
			onOpenCustomDialog();
		} else if (option.value) {
			onSnooze(option.value);
		}
		onClose();
	}

	function handleUnsnooze() {
		onUnsnooze();
		onClose();
	}

	function handleClickOutside(event: MouseEvent) {
		if (
			dropdownElement &&
			!dropdownElement.contains(event.target as Node) &&
			buttonElement &&
			!buttonElement.contains(event.target as Node)
		) {
			onClose();
		}
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (!isOpen) return;

		const key = event.key;

		// Handle number keys 1-4 for quick selection of snooze options
		if (key >= "1" && key <= "4") {
			const index = parseInt(key) - 1;
			if (index < options.length) {
				event.preventDefault();
				handleOptionClick(options[index]);
			}
		}

		// Handle key 5 for unsnooze (if snoozed)
		if (key === "5" && isSnoozed) {
			event.preventDefault();
			handleUnsnooze();
		}

		// Handle Escape to close
		if (key === "Escape") {
			event.preventDefault();
			onClose();
		}
	}

	// Track if we've added event listeners
	let listenersAdded = false;

	function addListeners() {
		if (typeof document !== "undefined" && !listenersAdded) {
			document.addEventListener("click", handleClickOutside);
			document.addEventListener("keydown", handleKeyDown);
			listenersAdded = true;
		}
	}

	function removeListeners() {
		if (typeof document !== "undefined" && listenersAdded) {
			document.removeEventListener("click", handleClickOutside);
			document.removeEventListener("keydown", handleKeyDown);
			listenersAdded = false;
		}
	}

	onMount(() => {
		if (isOpen) {
			addListeners();
		}
	});

	onDestroy(() => {
		removeListeners();
	});

	$: {
		// Only run on client side
		if (typeof document !== "undefined") {
			if (isOpen) {
				addListeners();
			} else {
				removeListeners();
			}
		}
	}
</script>

{#if isOpen}
	<div
		bind:this={dropdownElement}
		role="dialog"
		aria-label="Snooze options"
		tabindex="-1"
		class="fixed min-w-[220px] rounded-lg border border-gray-300 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-900 {isPositioned
			? 'opacity-100'
			: 'opacity-0'} transition-opacity duration-75"
		style="z-index: 9999; top: {dropdownPosition.top}px; {dropdownPosition.left !== 'auto'
			? `left: ${dropdownPosition.left}px;`
			: ''} {dropdownPosition.right !== 'auto' ? `right: ${dropdownPosition.right}px;` : ''}"
		on:click|stopPropagation
		on:mousedown|stopPropagation
		on:keydown|stopPropagation
	>
		<!-- Header -->
		<div class="border-b border-gray-200 px-3 py-2 dark:border-gray-800 cursor-default">
			<h3 class="text-xs font-semibold tracking-wide text-gray-500 dark:text-gray-400">
				Snooze until
			</h3>
		</div>

		<!-- Options -->
		<div class="py-1 cursor-default">
			{#each options as option (option.id)}
				<button
					type="button"
					class="flex w-full items-center justify-between px-3 py-2 text-left text-sm text-gray-700 transition hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-800 cursor-pointer"
					on:click|stopPropagation={() => handleOptionClick(option)}
				>
					<span>{formatOptionLabel(option)}</span>
					<span class="ml-2 text-xs text-gray-400 dark:text-gray-500">{option.id}</span>
				</button>
			{/each}
		</div>

		{#if isSnoozed}
			<div class="my-1 border-t border-gray-200 dark:border-gray-800"></div>
			<button
				type="button"
				class="flex w-full items-center justify-between px-3 py-2 text-left text-sm text-red-600 transition hover:bg-gray-100 dark:text-red-400 dark:hover:bg-gray-800 cursor-pointer"
				on:click|stopPropagation={handleUnsnooze}
			>
				<span>Unsnooze</span>
				<span class="ml-2 text-xs text-red-400 dark:text-red-500">5</span>
			</button>
		{/if}
	</div>
{/if}
