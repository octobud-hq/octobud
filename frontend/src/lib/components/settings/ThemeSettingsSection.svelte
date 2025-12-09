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

	import { getThemeStore, type Theme } from "$lib/stores/themeStore";
	import { browser } from "$app/environment";

	const themeStore = getThemeStore();
	const { theme } = themeStore;

	function handleToggle() {
		themeStore.toggle();
	}

	$: isDark = $theme === "dark";
</script>

<div class="space-y-4">
	<!-- Theme Toggle -->
	<div class="flex items-center justify-between py-1">
		<div class="flex-1">
			<div class="flex items-center gap-2">
				<span class="text-md font-medium text-gray-900 dark:text-gray-100"> Dark mode </span>
			</div>
			<p class="text-xs text-gray-600 dark:text-gray-400 mt-1">
				Switch between light and dark mode.
			</p>
		</div>
		<label class="relative flex items-center cursor-pointer">
			<input type="checkbox" checked={isDark} on:change={handleToggle} class="sr-only peer" />
			<div
				class="relative w-9 h-5 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"
			></div>
		</label>
	</div>
</div>
