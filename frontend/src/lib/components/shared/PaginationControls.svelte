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

	export let pageRangeStart: number;
	export let pageRangeEnd: number;
	export let totalCount: number;
	export let page: number;
	export let totalPages: number;
	export let onPrevious: () => void = () => {};
	export let onNext: () => void = () => {};

	$: pageSummary =
		totalCount === 0
			? "0 of 0"
			: pageRangeStart === 0
				? `0 of ${totalCount}`
				: `${pageRangeStart}-${pageRangeEnd} of ${totalCount}`;
</script>

<div class="flex items-center justify-end gap-2 text-xs text-gray-600 dark:text-gray-400">
	<span>{pageSummary}</span>

	{#if totalPages > 1}
		<div class="flex items-center gap-1">
			<!-- Previous button -->
			<button
				class="flex h-7 items-center gap-1 rounded-lg border border-gray-300 dark:border-gray-700 px-2.5 text-xs font-medium transition hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-800 dark:hover:text-gray-200 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-600 dark:disabled:hover:text-gray-400"
				type="button"
				on:click={onPrevious}
				disabled={page <= 1}
				aria-label="Previous page"
			>
				<svg
					width="12"
					height="12"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<polyline points="15 18 9 12 15 6"></polyline>
				</svg>
			</button>

			<!-- Next button -->
			<button
				class="flex h-7 items-center gap-1 rounded-lg border border-gray-300 dark:border-gray-700 px-2.5 text-xs font-medium transition hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-800 dark:hover:text-gray-200 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-600 dark:disabled:hover:text-gray-400"
				type="button"
				on:click={onNext}
				disabled={page >= totalPages}
				aria-label="Next page"
			>
				<svg
					width="12"
					height="12"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<polyline points="9 18 15 12 9 6"></polyline>
				</svg>
			</button>
		</div>
	{/if}
</div>
