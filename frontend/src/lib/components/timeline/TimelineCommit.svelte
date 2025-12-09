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

	import type { NotificationTimelineItem } from "$lib/api/types";
	import { formatRelativeShort } from "$lib/utils/time";

	export let item: NotificationTimelineItem;
	export let showThread: boolean = true;
	export let isLastItem: boolean = false;

	$: authorLogin = item.author?.login || item.actor?.login || "Unknown";
	$: authorAvatar = item.author?.avatarUrl || item.actor?.avatarUrl;
	$: authorInitial = authorLogin.charAt(0).toUpperCase();
	$: shortSha = item.sha?.substring(0, 7) || "";
	$: commitMessage = item.message || "";
	$: timestamp = item.createdAt || item.timestamp;

	// Helper to safely format timestamp with relative time
	$: formattedTimestamp = (() => {
		if (!timestamp) return "";
		const date = new Date(timestamp);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	})();

	let avatarLoadFailed = false;

	function handleAvatarError() {
		avatarLoadFailed = true;
	}
</script>

<div class="flex gap-3 pt-4">
	<!-- Avatar and thread line -->
	<div class="flex flex-col items-center flex-shrink-0 relative z-10" style="width: 40px;">
		<!-- Avatar / Initial -->
		<div class="flex-shrink-0">
			{#if authorAvatar && !avatarLoadFailed}
				<img
					src={authorAvatar}
					alt={`${authorLogin}'s avatar`}
					class="h-8 w-8 rounded-full bg-gray-300 dark:bg-gray-700 ring-2 ring-white dark:ring-gray-950"
					on:error={handleAvatarError}
				/>
			{:else}
				<div
					class="h-8 w-8 rounded-full bg-gray-300 dark:bg-gray-700 flex items-center justify-center ring-2 ring-white dark:ring-gray-950"
				>
					<span class="text-xs font-bold text-gray-800 dark:text-gray-300">
						{authorInitial}
					</span>
				</div>
			{/if}
		</div>

		<!-- Thread line extending down -->
		{#if showThread && !isLastItem}
			<div class="flex-1 w-0.5 bg-gray-300 dark:bg-gray-800 min-h-4 -z-10"></div>
		{/if}
	</div>

	<!-- Commit info -->
	<div class="flex-1 min-w-0">
		<div
			class="border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden bg-gray-50/50 dark:bg-gray-900/50"
		>
			<div class="px-3 py-2">
				<div class="flex items-start gap-2">
					<!-- Commit icon -->
					<svg
						class="w-4 h-4 mt-0.5 text-gray-600 dark:text-gray-400 flex-shrink-0"
						viewBox="0 0 16 16"
						fill="currentColor"
					>
						<path
							d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"
						/>
					</svg>

					<!-- Commit details -->
					<div class="flex-1 min-w-0">
						<div class="text-sm text-gray-700 dark:text-gray-300">
							<span class="font-semibold">{authorLogin}</span>
							<span class="text-gray-600 dark:text-gray-400 mx-1">committed</span>
							{#if formattedTimestamp}
								<span class="text-gray-600 dark:text-gray-500">
									{formattedTimestamp}
								</span>
							{/if}
						</div>
						{#if commitMessage}
							<div class="mt-1 text-sm text-gray-700 dark:text-gray-300">
								{commitMessage}
							</div>
						{/if}
						{#if shortSha}
							<div class="mt-1">
								{#if item.htmlUrl}
									<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
									<a
										href={item.htmlUrl}
										target="_blank"
										rel="noreferrer"
										class="inline-flex items-center gap-1 text-xs font-mono text-indigo-400 hover:text-indigo-300 hover:underline"
									>
										{shortSha}
										<svg
											class="w-3 h-3"
											viewBox="0 0 24 24"
											fill="none"
											xmlns="http://www.w3.org/2000/svg"
										>
											<path
												d="M10.0002 5H8.2002C7.08009 5 6.51962 5 6.0918 5.21799C5.71547 5.40973 5.40973 5.71547 5.21799 6.0918C5 6.51962 5 7.08009 5 8.2002V15.8002C5 16.9203 5 17.4801 5.21799 17.9079C5.40973 18.2842 5.71547 18.5905 6.0918 18.7822C6.5192 19 7.07899 19 8.19691 19H15.8031C16.921 19 17.48 19 17.9074 18.7822C18.2837 18.5905 18.5905 18.2839 18.7822 17.9076C19 17.4802 19 16.921 19 15.8031V14M20 9V4M20 4H15M20 4L13 11"
												stroke="currentColor"
												stroke-width="2"
												stroke-linecap="round"
												stroke-linejoin="round"
											/>
										</svg>
									</a>
								{:else}
									<span class="text-xs font-mono text-gray-600 dark:text-gray-500">
										{shortSha}
									</span>
								{/if}
							</div>
						{/if}
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
