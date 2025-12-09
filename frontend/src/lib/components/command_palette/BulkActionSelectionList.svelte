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

	import type { BulkAction } from "$lib/state/commandPaletteController";

	export let bulkActions: BulkAction[];
	export let selectedIndex: number;
	export let commandQuery: string;
	export let onSelect: (action: BulkAction, index: number) => void;
	export let onPointerMove: (index: number) => void;
</script>

<div class="max-h-80 overflow-y-auto p-2">
	{#if bulkActions.length > 0}
		<ul class="space-y-1" role="listbox" aria-label="Bulk actions">
			{#each bulkActions as action, index (action.id)}
				<li>
					<button
						type="button"
						data-bulk-action-index={index}
						class={`flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm text-gray-900 dark:text-gray-200 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/60 cursor-pointer ${
							index === selectedIndex
								? "bg-gray-100 dark:bg-gray-700"
								: "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800"
						}`}
						role="option"
						aria-selected={index === selectedIndex}
						on:click={() => onSelect(action, index)}
						on:mouseenter={() => onPointerMove(index)}
						on:focus={() => onPointerMove(index)}
					>
						<span class="flex flex-col gap-1">
							<span class="font-semibold text-gray-700 dark:text-gray-300">
								{action.title}
							</span>
							<span class="text-xs text-gray-600 dark:text-gray-400">
								{action.subtitle}
							</span>
						</span>
					</button>
				</li>
			{/each}
		</ul>
	{:else}
		<div class="px-3 py-6 text-center text-sm text-gray-600 dark:text-gray-400">
			No matching bulk actions found for "{commandQuery}"
		</div>
	{/if}
</div>
