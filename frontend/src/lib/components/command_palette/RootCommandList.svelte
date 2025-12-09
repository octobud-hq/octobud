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

	import type { CommandPaletteCommandDefinition } from "$lib/state/commandPaletteController";

	export let commands: CommandPaletteCommandDefinition[];
	export let selectedIndex: number;
	export let rootQuery: string;
	export let onSelect: (command: CommandPaletteCommandDefinition, index: number) => void;
	export let onPointerMove: (index: number) => void;
</script>

<div
	class="max-h-80 overflow-y-auto space-y-2 px-2 py-3"
	role="listbox"
	aria-label="Command suggestions"
>
	{#if commands.length > 0}
		{#each commands as command, index (command.id)}
			<button
				type="button"
				data-root-command-index={index}
				class={`flex w-full items-center justify-between gap-4 rounded-md px-3 py-2 text-left text-sm text-gray-900 dark:text-gray-200 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/60 cursor-pointer ${
					index === selectedIndex
						? "bg-gray-100 dark:bg-gray-700"
						: "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800"
				}`}
				aria-selected={index === selectedIndex}
				on:click={() => onSelect(command, index)}
				on:mouseenter={() => onPointerMove(index)}
				on:focus={() => onPointerMove(index)}
				role="option"
			>
				<span>
					<span class="block font-semibold text-gray-700 dark:text-gray-300">
						{command.title}
					</span>
					{#if command.subtitle}
						<span class="block text-xs text-gray-600 dark:text-gray-400">
							{command.subtitle}
						</span>
					{/if}
				</span>
				{#if command.shortcut}
					<span
						class="rounded-full border border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-950 px-2 py-1 text-xs text-gray-600 dark:text-gray-400"
					>
						{command.shortcut}
					</span>
				{/if}
			</button>
		{/each}
	{:else}
		<div
			class="rounded-md border border-gray-200 dark:border-gray-800/60 bg-gray-50 dark:bg-gray-900 px-3 py-3 text-sm text-gray-600 dark:text-gray-400"
		>
			No commands match "{rootQuery}".
		</div>
	{/if}
	<div
		class="rounded-xl bg-gray-50 dark:bg-gray-900 px-3 py-2 text-xs text-gray-600 dark:text-gray-400"
	>
		Type <span class="font-semibold text-gray-900 dark:text-gray-200">&gt;view</span>
		to switch views or
		<span class="font-semibold text-gray-900 dark:text-gray-200">&gt;bulk</span> to run a quick bulk action.
	</div>
</div>
