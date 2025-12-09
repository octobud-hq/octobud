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

	import QuerySyntaxHelp from "../shared/QuerySyntaxHelp.svelte";
	import { onMount } from "svelte";

	export let open = false;
	export let onClose: () => void = () => {};

	function handleClose() {
		open = false;
		onClose();
	}

	function handleBackdropClick(event: MouseEvent) {
		if (event.target === event.currentTarget) {
			handleClose();
		}
	}

	onMount(() => {
		const handleKeyDown = (event: KeyboardEvent) => {
			if (event.key === "Escape" && open) {
				handleClose();
				event.preventDefault();
				event.stopPropagation();
				event.stopImmediatePropagation();
			}
		};

		window.addEventListener("keydown", handleKeyDown, { capture: true });
		return () => {
			window.removeEventListener("keydown", handleKeyDown, {
				capture: true,
			});
		};
	});
</script>

{#if open}
	<div
		class="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50 p-4"
		on:click={handleBackdropClick}
		on:keydown={() => {}}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div
			class="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-2xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950 p-6 shadow-xl"
		>
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-xl font-semibold text-gray-900 dark:text-gray-100">Query Syntax Guide</h2>
				<button
					type="button"
					class="flex h-10 w-10 items-center justify-center rounded-full border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 cursor-pointer"
					on:click={handleClose}
					aria-label="Close"
				>
					<span aria-hidden="true" class="text-xl">✕</span>
				</button>
			</div>

			<div class="overflow-auto">
				<QuerySyntaxHelp />
			</div>

			<div class="mt-6 flex justify-end">
				<button
					type="button"
					class="rounded-full bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-700 cursor-pointer"
					on:click={handleClose}
				>
					Got it
				</button>
			</div>
		</div>
	</div>
{/if}
