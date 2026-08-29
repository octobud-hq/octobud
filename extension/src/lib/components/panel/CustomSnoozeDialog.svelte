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

	// Backs the SnoozeDropdown's "Custom date..." option. The desktop app opens a
	// full CustomSnoozeDateDialog; a panel this narrow gets a single
	// datetime-local field in the app's modal idiom.

	import { tick } from "svelte";
	import {
		defaultSnoozeTarget,
		isFutureInputValue,
		nowAsInputValue,
		toIsoInstant,
	} from "$lib/panel/datetime";

	export let open = false;
	export let onConfirm: (until: string) => void = () => {};
	export let onCancel: () => void = () => {};

	let value = "";
	let minimum = nowAsInputValue();
	let inputElement: HTMLInputElement | null = null;

	$: if (open && !value) {
		value = defaultSnoozeTarget();
		// Recomputed on open rather than once at module load, so a panel left open
		// overnight can't keep yesterday's floor.
		minimum = nowAsInputValue();
		void tick().then(() => inputElement?.focus());
	}

	$: if (!open) {
		value = "";
	}

	$: isValid = isFutureInputValue(value);

	function confirm() {
		if (!isValid) return;
		onConfirm(toIsoInstant(value));
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === "Escape") onCancel();
		if (event.key === "Enter") confirm();
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-[100] flex items-center justify-center bg-gray-950/50 px-4"
		role="presentation"
		on:click|self={onCancel}
	>
		<div
			class="w-full max-w-[18rem] rounded-2xl border border-gray-200 bg-white p-4 shadow-2xl dark:border-gray-800 dark:bg-gray-950"
			role="dialog"
			aria-modal="true"
			aria-label="Snooze until a custom date"
			tabindex="-1"
			on:keydown={handleKeydown}
		>
			<h2 class="text-sm font-semibold text-gray-900 dark:text-gray-100">Snooze until</h2>

			<input
				bind:this={inputElement}
				bind:value
				type="datetime-local"
				min={minimum}
				class="mt-3 w-full rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-sm text-gray-900 focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/30 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100"
			/>

			<div class="mt-4 flex justify-end gap-2">
				<button
					type="button"
					class="rounded-full px-3 py-1.5 text-xs font-semibold text-gray-600 transition hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
					on:click={onCancel}
				>
					Cancel
				</button>
				<button
					type="button"
					class="rounded-full bg-indigo-600 px-3.5 py-1.5 text-xs font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-indigo-500 dark:hover:bg-indigo-400"
					disabled={!isValid}
					on:click={confirm}
				>
					Snooze
				</button>
			</div>
		</div>
	</div>
{/if}
