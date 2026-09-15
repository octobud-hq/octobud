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

	/**
	 * Unread badge for repository selector rows, in the sidebar's badge style. Renders a
	 * fixed-width, right-aligned slot even when there is nothing unread, so the row
	 * actions to its left stay aligned across rows.
	 */
	export let count: number;
	export let highlighted: boolean = false;
	export let label: string;
</script>

<span class="flex w-9 flex-shrink-0 justify-end" aria-hidden={count <= 0}>
	{#if count > 0}
		<span
			class={`flex h-6 items-center justify-center rounded-full px-2 text-[11px] font-semibold ${
				highlighted
					? "bg-white/80 text-gray-700 dark:bg-gray-900 dark:text-gray-100"
					: "bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-100"
			}`}
			aria-label={label}
		>
			{count}
		</span>
	{/if}
</span>
