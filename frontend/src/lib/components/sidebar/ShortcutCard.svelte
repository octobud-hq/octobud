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

	export let onShowMore: () => void = () => {};

	type Shortcut = {
		keys: string[];
		description: string;
	};

	const shortcuts: Shortcut[] = [
		{ keys: ["⌘⇧K"], description: "Open command palette" },
		{ keys: ["J ", " K"], description: "Navigate notification list" },
		{ keys: ["R"], description: "Toggle notification read" },
	];
</script>

<section class="text-gray-900 dark:text-gray-200 px-2">
	<h3 class="text-xs font-semibold text-gray-600 dark:text-gray-400">SHORTCUTS</h3>
	<ul class="mt-4 space-y-2">
		{#each shortcuts as shortcut (shortcut.description)}
			<li class="flex items-center justify-between gap-4">
				<span class="text-sm text-gray-700 dark:text-gray-600">{shortcut.description}</span>
				<span class="flex items-center gap-1">
					{#each shortcut.keys as key, i (key + i)}
						<span
							class="inline-flex items-center rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-100/50 dark:bg-gray-800/50 px-2 py-0.5 text-xs font-medium text-gray-600 dark:text-gray-500"
						>
							{key}
						</span>
					{/each}
				</span>
			</li>
		{/each}
	</ul>
	<button
		type="button"
		class="mt-4 w-full rounded-full border border-gray-300 dark:border-gray-800 bg-transparent px-3 py-2 text-xs font-medium tracking-wide text-gray-700 dark:text-gray-600 transition hover:border-gray-400 dark:hover:border-gray-700 hover:text-gray-800 dark:hover:text-gray-500"
		on:click={onShowMore}
	>
		All Shortcuts
	</button>
</section>
