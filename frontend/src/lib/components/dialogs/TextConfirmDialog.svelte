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

	import { onMount, tick } from "svelte";

	export let open = false;
	export let title = "Are you sure?";
	export let body = "This action cannot be undone.";
	export let confirmLabel = "Confirm";
	export let cancelLabel = "Cancel";
	export let confirming = false;
	export let confirmTone: "danger" | "default" = "default";
	export let onConfirm: () => void = () => {};
	export let onCancel: () => void = () => {};
	export let requiredText = "";
	export let inputLabel = "Type to confirm:";

	let inputValue = "";
	let inputElement: HTMLInputElement | null = null;

	// Reset input when dialog closes
	$: if (!open) {
		inputValue = "";
	}

	// Focus the input when dialog opens
	$: if (open && inputElement) {
		tick().then(() => {
			inputElement?.focus();
		});
	}

	// Reactive statement to track if confirm button should be enabled
	$: isConfirmEnabled = inputValue.trim() === requiredText && !confirming;

	const handleConfirm = () => {
		if (isConfirmEnabled) {
			onConfirm();
		}
	};

	const handleKey = (event: KeyboardEvent) => {
		if (!open) return;

		if (event.key === "Escape") {
			event.preventDefault();
			event.stopPropagation();
			event.stopImmediatePropagation();
			onCancel();
		} else if (event.key === "Enter" && isConfirmEnabled) {
			event.preventDefault();
			event.stopPropagation();
			event.stopImmediatePropagation();
			handleConfirm();
		}
	};

	onMount(() => {
		if (typeof window === "undefined") {
			return;
		}

		// Use capture phase to handle before other keyboard shortcuts
		const listenerOptions = { capture: true };
		window.addEventListener("keydown", handleKey, listenerOptions);
		return () => {
			window.removeEventListener("keydown", handleKey, listenerOptions);
		};
	});
</script>

{#if open}
	<div class="fixed inset-0 z-[1200] flex items-center justify-center px-4 py-8">
		<div
			class="absolute inset-0 bg-black/60 dark:bg-gray-950/60 backdrop-blur-sm"
			role="presentation"
			on:click={onCancel}
		></div>
		<div
			class="relative w-full max-w-md rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 text-gray-900 dark:text-gray-200 shadow-2xl"
			role="dialog"
			aria-modal="true"
			aria-label={title}
		>
			<header class="space-y-2">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-gray-300">{title}</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400">{body}</p>
			</header>

			<div class="mt-4 space-y-2">
				<label
					for="confirm-input"
					class="block text-sm font-medium text-gray-700 dark:text-gray-300"
				>
					{inputLabel}
				</label>
				<input
					id="confirm-input"
					bind:this={inputElement}
					type="text"
					bind:value={inputValue}
					placeholder={requiredText}
					disabled={confirming}
					class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:focus:border-indigo-400 dark:focus:ring-indigo-400 disabled:opacity-50 disabled:cursor-not-allowed"
				/>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					Type <span class="font-mono font-medium">{requiredText}</span> to confirm
				</p>
			</div>

			<footer class="mt-6 flex justify-end gap-3">
				<button
					type="button"
					class="rounded-full border border-transparent px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
					on:click={onCancel}
					disabled={confirming}
				>
					{cancelLabel}
				</button>
				<button
					type="button"
					class={`rounded-full px-4 py-2 text-sm font-semibold text-white transition disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer ${
						confirmTone === "danger"
							? "bg-rose-500 hover:bg-rose-400"
							: "bg-indigo-600 hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-400"
					}`}
					on:click={handleConfirm}
					disabled={!isConfirmEnabled}
				>
					{confirming ? "Working…" : confirmLabel}
				</button>
			</footer>
		</div>
	</div>
{/if}
