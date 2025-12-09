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

	import { onMount, createEventDispatcher } from "svelte";

	export let selectedActions: string[] = [];
	export let availableActions: Array<{
		id: string;
		label: string;
		description: string;
		icon?: string;
	}> = [];

	const dispatch = createEventDispatcher<{ change: string[] }>();

	let dropdownOpen = false;
	let dropdownElement: HTMLDivElement;
	let buttonElement: HTMLButtonElement;

	// Toggle action selection
	function toggleAction(actionId: string) {
		let newSelected: string[];
		if (selectedActions.includes(actionId)) {
			newSelected = selectedActions.filter((id) => id !== actionId);
		} else {
			newSelected = [...selectedActions, actionId];
		}
		selectedActions = newSelected;
		dispatch("change", newSelected);
	}

	// Close dropdown when clicking outside
	function handleClickOutside(event: MouseEvent) {
		if (
			dropdownElement &&
			!dropdownElement.contains(event.target as Node) &&
			buttonElement &&
			!buttonElement.contains(event.target as Node)
		) {
			dropdownOpen = false;
		}
	}

	onMount(() => {
		document.addEventListener("click", handleClickOutside);
		return () => {
			document.removeEventListener("click", handleClickOutside);
		};
	});

	function toggleDropdown() {
		dropdownOpen = !dropdownOpen;
	}

	function getActionLabel(actionId: string): string {
		return availableActions.find((a) => a.id === actionId)?.label || actionId;
	}

	function getActionDescription(actionId: string): string {
		return availableActions.find((a) => a.id === actionId)?.description || "";
	}

	$: selectedLabels = selectedActions.map((id) => getActionLabel(id));
	$: displayText =
		selectedActions.length === 0
			? "Select actions..."
			: selectedActions.length === 1
				? selectedLabels[0]
				: `${selectedActions.length} actions selected`;
</script>

<div class="relative">
	<!-- Dropdown button -->
	<button
		bind:this={buttonElement}
		type="button"
		on:click={toggleDropdown}
		class="w-full rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-left outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30 flex items-center justify-between min-h-[42px]"
	>
		<span
			class="flex-1 text-gray-900 dark:text-gray-200 {selectedActions.length === 0
				? 'text-gray-600 dark:text-gray-500'
				: ''}"
		>
			{displayText}
		</span>
		<svg
			class="w-4 h-4 text-gray-600 dark:text-gray-400 transition-transform flex-shrink-0 ml-2 {dropdownOpen
				? 'rotate-180'
				: ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	<!-- Dropdown menu -->
	{#if dropdownOpen}
		<div
			bind:this={dropdownElement}
			class="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-60 overflow-y-auto"
			role="listbox"
		>
			{#if availableActions.length === 0}
				<div class="px-3 py-2 text-sm text-gray-600 dark:text-gray-500 text-center">
					No actions available
				</div>
			{:else}
				{#each availableActions as action (action.id)}
					<button
						type="button"
						on:click={() => toggleAction(action.id)}
						class="w-full px-3 py-2 text-left hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors flex items-start gap-3 {selectedActions.includes(
							action.id
						)
							? 'bg-gray-100 dark:bg-gray-700'
							: ''}"
					>
						<input
							type="checkbox"
							checked={selectedActions.includes(action.id)}
							class="mt-0.5 w-4 h-4 rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-indigo-600 focus:ring-2 focus:ring-indigo-600/30 focus:ring-offset-0 flex-shrink-0"
							readonly
						/>
						<div class="flex-1 min-w-0">
							<div class="text-sm text-gray-900 dark:text-gray-200 font-medium">
								{action.label}
							</div>
							{#if action.description}
								<div class="text-xs text-gray-600 dark:text-gray-500 mt-0.5">
									{action.description}
								</div>
							{/if}
						</div>
						{#if selectedActions.includes(action.id)}
							<svg
								class="w-4 h-4 text-blue-400 flex-shrink-0 mt-0.5"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fill-rule="evenodd"
									d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
									clip-rule="evenodd"
								/>
							</svg>
						{/if}
					</button>
				{/each}
			{/if}
		</div>
	{/if}
</div>
