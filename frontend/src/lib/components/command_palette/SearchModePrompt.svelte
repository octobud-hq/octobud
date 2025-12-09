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

	export let searchTerm: string;
</script>

<div class="p-3 text-sm text-gray-700 dark:text-gray-300 space-y-2">
	<p class="font-semibold text-gray-900 dark:text-gray-300">
		Press Enter to search all notifications
	</p>
	{#if !searchTerm}
		<p class="text-xs text-gray-600 dark:text-gray-500">
			Start typing to search notification by title, type, author, etc.
		</p>
	{:else}
		<p class="text-xs text-gray-600 dark:text-gray-500">
			Searching for "{searchTerm}"
		</p>
	{/if}
</div>
