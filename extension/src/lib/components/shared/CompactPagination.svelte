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

	export let page: number;
	export let totalPages: number;
	export let onPrevious: () => void = () => {};
	export let onNext: () => void = () => {};
</script>

<div class="flex items-center gap-1.5">
	<!-- Previous button -->
	<button
		class="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 dark:border-gray-800 bg-gray-100 dark:bg-gray-900 text-gray-700 dark:text-gray-200 transition hover:border-gray-400 dark:hover:border-gray-700 hover:bg-gray-200 dark:hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-gray-300 dark:disabled:hover:border-gray-800 disabled:hover:bg-gray-100 dark:disabled:hover:bg-gray-900 cursor-pointer"
		type="button"
		on:click={onPrevious}
		disabled={page <= 1}
		aria-label="Previous page"
	>
		<svg
			width="16"
			height="16"
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
		class="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 dark:border-gray-800 bg-gray-100 dark:bg-gray-900 text-gray-700 dark:text-gray-200 transition hover:border-gray-400 dark:hover:border-gray-700 hover:bg-gray-200 dark:hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-gray-300 dark:disabled:hover:border-gray-800 disabled:hover:bg-gray-100 dark:disabled:hover:bg-gray-900 cursor-pointer"
		type="button"
		on:click={onNext}
		disabled={page >= totalPages}
		aria-label="Next page"
	>
		<svg
			width="16"
			height="16"
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
