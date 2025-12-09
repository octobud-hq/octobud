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

	import type { NotificationThreadItem } from "$lib/api/types";
	import { formatRelativeShort } from "$lib/utils/time";
	import { renderMarkdown } from "$lib/utils/markdown";

	export let item: NotificationThreadItem;
	export let showThread: boolean = true;
	export let isLastItem: boolean = false;

	$: authorName = item.author.login;
	$: authorInitial = authorName.charAt(0).toUpperCase();
	$: authorAvatarUrl = item.author.avatarUrl;
	$: isBot =
		authorName.toLowerCase().includes("[bot]") || authorName.toLowerCase().endsWith("-bot");

	let avatarLoadFailed = false;

	function handleAvatarError() {
		avatarLoadFailed = true;
	}

	$: timestamp =
		item.type === "comment" || item.type === "commented"
			? item.createdAt
			: item.type === "review" || item.type === "reviewed"
				? item.submittedAt
				: "";

	$: wasEdited =
		(item.type === "comment" || item.type === "commented") &&
		item.updatedAt &&
		item.createdAt !== item.updatedAt;

	// Helper to safely format timestamp with relative time
	$: formattedTimestamp = (() => {
		if (!timestamp) return "";
		const date = new Date(timestamp);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	})();

	// Check if this is a review without a body - should render as simple event
	$: isReviewWithoutBody =
		(item.type === "review" || item.type === "reviewed") && (!item.body || !item.body.trim());

	// Review state styling
	$: reviewBadgeConfig =
		(item.type === "review" || item.type === "reviewed") && item.state
			? getReviewBadgeConfig(item.state)
			: null;

	function getReviewBadgeConfig(state: string) {
		switch (state) {
			case "APPROVED":
				return {
					label: "Approved",
					icon: "✓",
					useIcon: false,
					bgClass: "bg-green-600",
					textClass: "text-white",
				};
			case "CHANGES_REQUESTED":
				return {
					label: "Changes requested",
					icon: "⚠",
					useIcon: false,
					bgClass: "bg-red-600",
					textClass: "text-white",
				};
			case "COMMENTED":
				return {
					label: "Reviewed",
					icon: "/comment.svg",
					useIcon: true, // Use SVG image instead of emoji
					bgClass: "bg-gray-200 dark:bg-gray-600",
					textClass: "text-gray-600 dark:text-white",
				};
			case "DISMISSED":
				return {
					label: "Dismissed",
					icon: "✕",
					useIcon: false,
					bgClass: "bg-gray-200 dark:bg-gray-600",
					textClass: "text-gray-600 dark:text-white",
				};
			default:
				return {
					label: "Pending",
					icon: "○",
					useIcon: false,
					bgClass: "bg-gray-600",
					textClass: "text-white",
				};
		}
	}
</script>

{#if isReviewWithoutBody}
	<!-- Render review without body as a simple timeline event -->
	<div class="flex gap-3 pt-4">
		<!-- Avatar and thread line -->
		<div class="flex flex-col items-center flex-shrink-0 relative z-10" style="width: 40px;">
			<!-- Avatar / Initial -->
			<div class="flex-shrink-0">
				{#if authorAvatarUrl && !avatarLoadFailed}
					<img
						src={authorAvatarUrl}
						alt={`${authorName}'s avatar`}
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

		<!-- Event info -->
		<div class="flex-1 min-w-0 py-1">
			<div class="text-sm text-gray-600 dark:text-gray-400">
				<span class="font-semibold text-gray-700 dark:text-gray-300">{authorName}</span>
				{#if reviewBadgeConfig}
					<span
						class="inline-flex items-center gap-1 px-2 py-0.5 mx-1 text-xs font-medium rounded-md {reviewBadgeConfig.bgClass} {reviewBadgeConfig.textClass}"
					>
						{#if reviewBadgeConfig.useIcon}
							<img
								src={reviewBadgeConfig.icon}
								alt=""
								class="w-3 h-3 brightness-0 opacity-50 dark:invert dark:opacity-60"
							/>
						{:else}
							<span>{reviewBadgeConfig.icon}</span>
						{/if}
						<span>{reviewBadgeConfig.label}</span>
					</span>
					{#if formattedTimestamp}
						<span class="text-gray-600 dark:text-gray-500">
							{formattedTimestamp}
						</span>
					{/if}
				{:else}
					<span class="mx-1"
						>reviewed{#if formattedTimestamp}
							{formattedTimestamp}{/if}</span
					>
				{/if}
			</div>
		</div>
	</div>
{:else}
	<!-- Render comment or review with body as full card -->
	<div class="flex gap-3 pt-4" class:pb-4={isLastItem}>
		<!-- Avatar column with thread line -->
		<div class="flex flex-col items-center flex-shrink-0 relative z-10" style="width: 40px;">
			<!-- Avatar -->
			<div class="flex-shrink-0">
				{#if authorAvatarUrl && !avatarLoadFailed}
					<img
						src={authorAvatarUrl}
						alt={`${authorName}'s avatar`}
						class="h-10 w-10 rounded-full bg-gray-700 ring-2 ring-gray-200 dark:ring-gray-950"
						on:error={handleAvatarError}
					/>
				{:else}
					<div
						class="h-10 w-10 rounded-full bg-gray-700 flex items-center justify-center ring-2 ring-gray-200 dark:ring-gray-950"
					>
						<span class="text-sm font-bold text-gray-800 dark:text-gray-200">
							{authorInitial}
						</span>
					</div>
				{/if}
			</div>

			<!-- Thread line extending down - only if not the last item -->
			{#if !isLastItem}
				<div class="flex-1 w-0.5 bg-gray-300 dark:bg-gray-800 min-h-4 -z-10"></div>
			{/if}
		</div>

		<!-- Comment/Review content -->
		<div class="flex-1">
			<div class="border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden">
				<!-- Header -->
				<div
					class="flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-800"
				>
					<div class="flex items-center gap-3">
						<!-- Author Info -->
						<div class="flex items-center gap-2 text-sm">
							<span class="font-semibold text-gray-900 dark:text-gray-200">
								{authorName}
							</span>
							{#if isBot}
								<span
									class="px-1.5 py-0.5 text-xs font-medium border border-gray-300 dark:border-gray-700 rounded-md text-gray-600 dark:text-gray-400"
								>
									Bot
								</span>
							{/if}
							{#if reviewBadgeConfig}
								<span
									class="flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-md {reviewBadgeConfig.bgClass} {reviewBadgeConfig.textClass}"
								>
									{#if reviewBadgeConfig.useIcon}
										<img
											src={reviewBadgeConfig.icon}
											alt=""
											class="w-3 h-3 brightness-0 dark:invert opacity-60 dark:opacity-60"
										/>
									{:else}
										<span>{reviewBadgeConfig.icon}</span>
									{/if}
									<span>{reviewBadgeConfig.label}</span>
								</span>
								{#if formattedTimestamp}
									<span class="text-gray-600 dark:text-gray-500">
										{formattedTimestamp}
									</span>
								{/if}
							{:else}
								<span class="text-gray-600 dark:text-gray-500">
									commented {#if formattedTimestamp}
										{formattedTimestamp}{/if}
								</span>
							{/if}
							{#if wasEdited}
								<span class="text-gray-700 dark:text-gray-600 text-xs">(edited)</span>
							{/if}
						</div>
					</div>

					<!-- Link to GitHub for reviews -->
					{#if item.type === "review" || item.type === "reviewed"}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a
							href={item.htmlUrl}
							target="_blank"
							rel="noreferrer"
							class="text-xs text-indigo-300 hover:text-indigo-200 hover:underline"
						>
							View review in GitHub →
						</a>
					{/if}
				</div>

				<!-- Body -->
				<div class="p-4 leading-relaxed text-gray-900 dark:text-gray-200">
					<div
						class="prose dark:prose-invert prose-sm max-w-none prose-p:text-gray-700 dark:prose-p:text-gray-300 prose-headings:text-gray-900 dark:prose-headings:text-gray-300 prose-headings:font-semibold prose-a:text-indigo-600 dark:prose-a:text-indigo-300 prose-a:no-underline hover:prose-a:text-indigo-700 dark:hover:prose-a:text-indigo-200 hover:prose-a:underline prose-strong:text-gray-900 dark:prose-strong:text-gray-300 prose-strong:font-semibold prose-code:text-gray-800 dark:prose-code:text-gray-300 prose-code:bg-gray-100 dark:prose-code:bg-gray-900 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:before:content-[''] prose-code:after:content-[''] prose-pre:text-gray-800 dark:prose-pre:text-gray-300 prose-pre:bg-gray-100 dark:prose-pre:bg-[#161b22] prose-blockquote:text-gray-600 dark:prose-blockquote:text-gray-400 prose-ul:text-gray-700 dark:prose-ul:text-gray-300 prose-ol:text-gray-700 dark:prose-ol:text-gray-300 prose-li:text-gray-700 dark:prose-li:text-gray-300"
					>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html renderMarkdown(item.body)}
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}
