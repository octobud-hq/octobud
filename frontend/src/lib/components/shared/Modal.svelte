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

	export let open = false;
	export let title = "";
	export let size: "sm" | "md" | "lg" = "md";
	export let closeOnOverlay = true;
	export let customHeader = false;
	export let maxHeight: string = "calc(100vh - 16rem)";
	export let onClose: () => void = () => {};

	let dialogEl: HTMLDivElement | null = null;

	const handleKey = (event: KeyboardEvent) => {
		if (event.key === "Escape" && open) {
			event.preventDefault();
			event.stopPropagation();
			event.stopImmediatePropagation();
			onClose();
		}
	};

	// Prevent body scrolling when modal is open
	$: if (open && typeof document !== "undefined") {
		document.body.style.overflow = "hidden";
	} else if (!open && typeof document !== "undefined") {
		document.body.style.overflow = "";
	}

	onMount(() => {
		if (typeof window === "undefined") {
			return;
		}

		window.addEventListener("keydown", handleKey, { capture: true });
		return () => {
			window.removeEventListener("keydown", handleKey, { capture: true });
			// Clean up body overflow on unmount
			if (typeof document !== "undefined") {
				document.body.style.overflow = "";
			}
		};
	});

	$: dialogWidth = size === "sm" ? "max-w-lg" : size === "lg" ? "max-w-4xl" : "max-w-2xl";
</script>

{#if open}
	<div class="fixed inset-0 z-[100] overflow-hidden">
		<div
			class="fixed inset-0 bg-black/70 backdrop-blur-sm"
			role="presentation"
			on:click={() => {
				if (closeOnOverlay) onClose();
			}}
		></div>
		<div
			class={`fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 ${dialogWidth} w-[calc(100vw-2rem)] max-h-[calc(100vh-4rem)] rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 shadow-2xl flex flex-col`}
			role="dialog"
			aria-modal="true"
			aria-label={title}
			tabindex="-1"
			bind:this={dialogEl}
		>
			{#if customHeader}
				<slot name="header">
					<div
						class="flex items-center justify-between border-b border-gray-200 dark:border-gray-800 px-6 py-4 flex-shrink-0"
					>
						<h2 class="text-lg font-semibold text-gray-900 dark:text-gray-300">
							{title}
						</h2>
						<button
							type="button"
							class="flex h-10 w-10 items-center justify-center rounded-full border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 cursor-pointer"
							on:click={onClose}
							aria-label="Close dialog"
						>
							<span aria-hidden="true" class="text-xl">✕</span>
						</button>
					</div>
				</slot>
			{:else}
				<div
					class="flex items-center justify-between border-b border-gray-200 dark:border-gray-800 px-6 py-4 flex-shrink-0"
				>
					<h2 class="text-lg font-semibold text-gray-900 dark:text-gray-300">
						{title}
					</h2>
					<button
						type="button"
						class="flex h-10 w-10 items-center justify-center rounded-full border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 cursor-pointer"
						on:click={onClose}
						aria-label="Close dialog"
					>
						<span aria-hidden="true" class="text-xl">✕</span>
					</button>
				</div>
			{/if}
			<div class="overflow-y-auto px-6 py-4 flex-1 min-h-0" style="max-height: {maxHeight}">
				<slot />
			</div>
		</div>
	</div>
{/if}
