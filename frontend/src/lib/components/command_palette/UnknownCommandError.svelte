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

	export let commandToken: string;
	export let onSelectCommand: (commandId: string) => void;
</script>

<div class="space-y-3 px-3 py-3 text-sm text-gray-700 dark:text-gray-300">
	<div class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
		Unknown command "{commandToken}".
	</div>
	<div
		class="rounded-xl bg-gray-100 dark:bg-gray-800 px-4 py-3 text-xs text-gray-600 dark:text-gray-400"
	>
		Try <button
			class="font-semibold text-gray-700 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-200 cursor-pointer"
			type="button"
			on:click={() => onSelectCommand("view")}
		>
			&gt;view
		</button>,
		<button
			class="font-semibold text-gray-700 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-200 cursor-pointer"
			type="button"
			on:click={() => onSelectCommand("bulk")}
		>
			&gt;bulk
		</button>
		or
		<button
			class="font-semibold text-gray-700 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-200 cursor-pointer"
			type="button"
			on:click={() => onSelectCommand("shortcuts")}
		>
			&gt;shortcuts
		</button>
		to get started.
	</div>
</div>
