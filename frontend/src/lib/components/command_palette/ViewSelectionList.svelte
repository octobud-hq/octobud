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

	import type { ViewMatchEntry } from "$lib/utils/viewMatching";
	import { viewKey } from "$lib/utils/viewMatching";

	export let viewMatchEntries: ViewMatchEntry[];
	export let builtInViewMatches: ViewMatchEntry[];
	export let savedViewMatches: ViewMatchEntry[];
	export let tagViewMatches: ViewMatchEntry[] = [];
	export let settingsViewMatches: ViewMatchEntry[] = [];
	export let selectedIndex: number;
	export let effectiveSelectedSlug: string;
	export let commandQuery: string;
	export let onSelect: (entry: ViewMatchEntry) => void;
	export let onPointerMove: (index: number) => void;
</script>

<div class="max-h-80 overflow-y-auto p-2 space-y-3">
	{#if viewMatchEntries.length > 0}
		{#if builtInViewMatches.length > 0}
			<section>
				<header
					class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-500"
				>
					Pinned
				</header>
				<ul class="space-y-1" role="listbox" aria-label="Pinned views">
					{#each builtInViewMatches as match (viewKey(match.view))}
						<li>
							<button
								type="button"
								data-view-index={match.index}
								class={`flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm text-gray-900 dark:text-gray-200 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/60 cursor-pointer ${
									match.index === selectedIndex
										? "bg-gray-200 dark:bg-gray-700"
										: "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800"
								}`}
								role="option"
								aria-selected={match.index === selectedIndex}
								on:click={() => onSelect(match)}
								on:mouseenter={() => onPointerMove(match.index)}
								on:focus={() => onPointerMove(match.index)}
							>
								<span class="flex flex-col gap-1">
									<span class="font-semibold text-gray-700 dark:text-gray-300">
										{match.view.name}
									</span>
									<span class="text-xs text-gray-600 dark:text-gray-400">
										{match.view.slug}
									</span>
								</span>
								{#if viewKey(match.view) === effectiveSelectedSlug}
									<span
										class="flex items-center gap-1 text-xs font-semibold text-gray-600 dark:text-gray-500"
									>
										Current view
									</span>
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		{#if savedViewMatches.length > 0}
			<section>
				<header
					class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-500"
				>
					Saved views
				</header>
				<ul class="space-y-1" role="listbox" aria-label="Saved views">
					{#each savedViewMatches as match (viewKey(match.view))}
						<li>
							<button
								type="button"
								data-view-index={match.index}
								class={`flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm text-gray-900 dark:text-gray-200 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/60 cursor-pointer ${
									match.index === selectedIndex
										? "bg-gray-200 dark:bg-gray-700"
										: "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800"
								}`}
								role="option"
								aria-selected={match.index === selectedIndex}
								on:click={() => onSelect(match)}
								on:mouseenter={() => onPointerMove(match.index)}
								on:focus={() => onPointerMove(match.index)}
							>
								<span class="flex flex-col gap-1">
									<span class="font-semibold text-gray-700 dark:text-gray-300">
										{match.view.name}
									</span>
									<span class="text-xs text-gray-600 dark:text-gray-400">
										{match.view.description ? match.view.description : match.view.slug}
									</span>
								</span>
								{#if viewKey(match.view) === effectiveSelectedSlug}
									<span class="text-xs font-semibold text-gray-500"> Current view </span>
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		{#if tagViewMatches.length > 0}
			<section>
				<header
					class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-500"
				>
					Tag views
				</header>
				<ul class="space-y-1" role="listbox" aria-label="Tag views">
					{#each tagViewMatches as match (viewKey(match.view))}
						<li>
							<button
								type="button"
								data-view-index={match.index}
								class={`flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm text-gray-900 dark:text-gray-200 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/60 cursor-pointer ${
									match.index === selectedIndex
										? "bg-gray-200 dark:bg-gray-700"
										: "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800"
								}`}
								role="option"
								aria-selected={match.index === selectedIndex}
								on:click={() => onSelect(match)}
								on:mouseenter={() => onPointerMove(match.index)}
								on:focus={() => onPointerMove(match.index)}
							>
								<span class="flex flex-col gap-1">
									<span class="font-semibold text-gray-700 dark:text-gray-300">
										{match.view.name}
									</span>
									<span class="text-xs text-gray-600 dark:text-gray-400">
										{match.view.description ? match.view.description : match.view.slug}
									</span>
								</span>
								{#if viewKey(match.view) === effectiveSelectedSlug}
									<span class="text-xs font-semibold text-gray-500"> Current view </span>
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			</section>
		{/if}

		{#if settingsViewMatches.length > 0}
			<section>
				<header
					class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-500"
				>
					Settings
				</header>
				<ul class="space-y-1" role="listbox" aria-label="Settings">
					{#each settingsViewMatches as match (viewKey(match.view))}
						<li>
							<button
								type="button"
								data-view-index={match.index}
								class={`flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm text-gray-900 dark:text-gray-200 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/60 cursor-pointer ${
									match.index === selectedIndex
										? "bg-gray-200 dark:bg-gray-700"
										: "bg-transparent hover:bg-gray-100 dark:hover:bg-gray-800"
								}`}
								role="option"
								aria-selected={match.index === selectedIndex}
								on:click={() => onSelect(match)}
								on:mouseenter={() => onPointerMove(match.index)}
								on:focus={() => onPointerMove(match.index)}
							>
								<span class="flex flex-col gap-1">
									<span class="font-semibold text-gray-700 dark:text-gray-300">
										{match.view.name}
									</span>
									<span class="text-xs text-gray-600 dark:text-gray-400">
										{match.view.description ? match.view.description : match.view.slug}
									</span>
								</span>
								{#if viewKey(match.view) === effectiveSelectedSlug}
									<span class="text-xs font-semibold text-gray-500"> Current view </span>
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	{:else}
		<div
			class="flex flex-col gap-2 rounded-xl border border-gray-200 dark:border-gray-800/60 bg-gray-50 dark:bg-gray-900 px-3 py-3 text-sm text-gray-600 dark:text-gray-400"
		>
			{#if commandQuery.trim().length > 0}
				<span>No views match "{commandQuery.trim()}".</span>
				<span>Try a different keyword or use the sidebar to create a view.</span>
			{:else}
				<span>You haven't saved any custom views yet.</span>
				<span>Open the view builder to capture a filter you use frequently.</span>
			{/if}
		</div>
	{/if}
</div>
