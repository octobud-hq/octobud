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

	export let isOpen: boolean;
</script>

{#if isOpen}
	<div
		class="absolute left-1/2 top-full z-50 mt-2 -translate-x-1/2 overflow-hidden rounded-lg mt-2 border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-950 shadow-2xl backdrop-blur-lg"
		style="width: min(48rem, calc(100vw - 3rem));"
	>
		<slot />
	</div>
{/if}
