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

	import Modal from "$lib/components/shared/Modal.svelte";
	import { tick } from "svelte";

	export let isOpen: boolean = false;
	export let onConfirm: (until: string) => void = () => {};
	export let onClose: () => void = () => {};

	let dateTimeValue: string = "";
	let errorMessage: string = "";
	let inputElement: HTMLInputElement | null = null;
	let previousIsOpen = false;

	$: if (isOpen) {
		// Focus the input when dialog opens
		tick().then(() => {
			inputElement?.focus();
		});
		// Set default value only when dialog transitions from closed to open
		if (!previousIsOpen) {
			dateTimeValue = getDefaultDateTime();
		}
		previousIsOpen = true;
	} else {
		previousIsOpen = false;
	}

	function getMinDateTime(): string {
		// Set minimum to current date and time
		const now = new Date();
		// Format as YYYY-MM-DDTHH:MM (datetime-local format)
		const year = now.getFullYear();
		const month = String(now.getMonth() + 1).padStart(2, "0");
		const day = String(now.getDate()).padStart(2, "0");
		const hours = String(now.getHours()).padStart(2, "0");
		const minutes = String(now.getMinutes()).padStart(2, "0");
		return `${year}-${month}-${day}T${hours}:${minutes}`;
	}

	function getDefaultDateTime(): string {
		// Default to tomorrow at 9am
		const now = new Date();
		const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1, 9, 0, 0, 0);

		const year = tomorrow.getFullYear();
		const month = String(tomorrow.getMonth() + 1).padStart(2, "0");
		const day = String(tomorrow.getDate()).padStart(2, "0");
		const hours = String(tomorrow.getHours()).padStart(2, "0");
		const minutes = String(tomorrow.getMinutes()).padStart(2, "0");
		return `${year}-${month}-${day}T${hours}:${minutes}`;
	}

	function handleConfirm() {
		if (!dateTimeValue) {
			errorMessage = "Please select a date and time";
			return;
		}

		const selectedDate = new Date(dateTimeValue);
		const now = new Date();

		if (selectedDate <= now) {
			errorMessage = "Please select a future date and time";
			return;
		}

		errorMessage = "";
		onConfirm(selectedDate.toISOString());
		handleClose();
	}

	function handleClose() {
		dateTimeValue = "";
		errorMessage = "";
		onClose();
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === "Enter" && dateTimeValue) {
			event.preventDefault();
			handleConfirm();
		}
	}
</script>

<Modal open={isOpen} onClose={handleClose} title="Snooze notification until">
	<div class="space-y-4">
		<div>
			<label
				for="snooze-datetime"
				class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
			>
				Snooze until
			</label>
			<input
				bind:this={inputElement}
				id="snooze-datetime"
				type="datetime-local"
				bind:value={dateTimeValue}
				min={getMinDateTime()}
				on:keydown={handleKeyDown}
				class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100"
			/>
			{#if errorMessage}
				<p class="mt-2 text-sm text-red-600 dark:text-red-400">
					{errorMessage}
				</p>
			{/if}
		</div>

		<div class="flex items-center justify-end gap-3">
			<button
				type="button"
				on:click={handleClose}
				class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700 cursor-pointer"
			>
				Cancel
			</button>
			<button
				type="button"
				on:click={handleConfirm}
				class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500/20 dark:bg-blue-500 dark:hover:bg-blue-600 cursor-pointer"
			>
				Confirm
			</button>
		</div>
	</div>
</Modal>
