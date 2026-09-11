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

	import { computeAvatarUrl } from "$lib/utils/avatar";

	/**
	 * Owner avatar for a repository. Falls back to the github.com/<owner>.png redirect
	 * when no direct URL is known (like the other avatars in the app), then to an initial.
	 */
	export let fullName: string;
	export let url: string | null | undefined = null;

	let failed = false;
	$: owner = fullName.split("/")[0] ?? fullName;
	$: initial = owner.charAt(0).toUpperCase();
	$: src = computeAvatarUrl(url, owner);
</script>

{#if src && !failed}
	<img
		{src}
		alt=""
		loading="lazy"
		class="h-4 w-4 flex-shrink-0 rounded-sm bg-gray-200 dark:bg-gray-800 object-cover"
		on:error={() => (failed = true)}
	/>
{:else}
	<span
		class="flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-sm bg-gray-300 dark:bg-gray-700 text-[10px] font-semibold leading-none text-gray-700 dark:text-gray-200"
		aria-hidden="true"
	>
		{initial}
	</span>
{/if}
