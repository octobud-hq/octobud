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
	import { computeAvatarUrl, resolveAvatarRedirect } from "$lib/utils/avatar";

	export let item: NotificationTimelineItem;
	export let showThread: boolean = true;
	export let isLastItem: boolean = false;

	$: authorLogin = item.author?.login || item.actor?.login || "Unknown";
	$: directAvatarUrl = item.author?.avatarUrl || item.actor?.avatarUrl;
	$: authorInitial = authorLogin.charAt(0).toUpperCase();
	$: shortSha = item.sha?.substring(0, 7) || "";
	$: commitMessage = (() => {
		const msg = item.message || "";
		return msg.length > 100 ? msg.substring(0, 100) + "…" : msg;
	})();
	$: timestamp = item.createdAt || item.timestamp;

	// Compute base avatar URL (direct or redirect)
	$: authorAvatar = computeAvatarUrl(directAvatarUrl, authorLogin);

	// Resolved avatar URL (after following redirect if needed)
	let resolvedAvatarUrl: string | null = null;

	// Use resolved URL if available, otherwise fall back to original
	$: finalAvatarUrl = resolvedAvatarUrl || authorAvatar;

	// Helper to safely format timestamp with relative time
	$: formattedTimestamp = (() => {
		if (!timestamp) return "";
		const date = new Date(timestamp);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	})();

	let avatarLoadFailed = false;

	async function handleAvatarError() {
		// If we haven't resolved the redirect yet, try one more time
		if (authorAvatar && !directAvatarUrl && !resolvedAvatarUrl && !avatarLoadFailed) {
			const resolved = await resolveAvatarRedirect(authorAvatar);
			if (resolved) {
				resolvedAvatarUrl = resolved;
				avatarLoadFailed = false;
			} else {
				avatarLoadFailed = true;
			}
		} else {
			avatarLoadFailed = true;
		}
	}
</script>

<div class="flex gap-3 pt-4">
	<!-- Avatar and thread line -->
	<div class="flex flex-col items-center flex-shrink-0 relative z-10" style="width: 40px;">
		<!-- Avatar / Initial -->
		<div class="flex-shrink-0">
			{#if finalAvatarUrl && !avatarLoadFailed}
				<img
					src={finalAvatarUrl}
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

	<!-- Commit info (single line) -->
	<div class="flex-1 min-w-0 py-1">
		<div class="text-sm text-gray-600 dark:text-gray-400">
			<span class="font-semibold text-gray-700 dark:text-gray-300">{authorLogin}</span>
			<span
				class="inline-flex items-center justify-center w-6 h-6 mx-1 rounded-full bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 align-middle"
			>
				<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor">
					<path
						d="M11.93 8.5a4.002 4.002 0 0 1-7.86 0H.75a.75.75 0 0 1 0-1.5h3.32a4.002 4.002 0 0 1 7.86 0h3.32a.75.75 0 0 1 0 1.5Zm-1.43-.75a2.5 2.5 0 1 0-5 0 2.5 2.5 0 0 0 5 0Z"
					/>
				</svg>
			</span>
			{#if commitMessage && item.htmlUrl}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a
					href={item.htmlUrl}
					target="_blank"
					rel="noreferrer"
					class="font-medium text-indigo-600 dark:text-indigo-400 hover:underline"
					>{commitMessage}</a
				>
			{:else if commitMessage}
				<span class="text-gray-700 dark:text-gray-300">{commitMessage}</span>
			{/if}
			{#if shortSha}
				<span class="font-mono text-xs text-gray-700 dark:text-gray-300 pl-1">{shortSha}</span>
			{/if}
			{#if formattedTimestamp}
				<span class="text-gray-500 dark:text-gray-600 pl-1">{formattedTimestamp}</span>
			{/if}
		</div>
	</div>
</div>
