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

	import SettingsSidebar from "./SettingsSidebar.svelte";

	export let activeSection: string = "account";
</script>

<div class="flex-1 flex flex-col overflow-hidden bg-white dark:bg-gray-950">
	<!-- Page Header -->
	<div class="flex-shrink-0 px-6 pt-6 pb-4 border-b border-gray-200 dark:border-gray-800">
		<h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">Settings</h1>
		<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Manage your account and preferences</p>
	</div>

	<!-- Sidebar + Content -->
	<div class="flex-1 flex overflow-hidden">
		<!-- Settings Sidebar - anchored left -->
		<div class="flex-shrink-0 border-r border-gray-200 dark:border-gray-800">
			<div class="px-4 py-2">
				<SettingsSidebar {activeSection} />
			</div>
		</div>

		<!-- Content Area - scrollable -->
		<div class="flex-1 overflow-y-auto">
			<div class="max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-8 py-6 pb-24">
				<slot />
			</div>
		</div>
	</div>
</div>
