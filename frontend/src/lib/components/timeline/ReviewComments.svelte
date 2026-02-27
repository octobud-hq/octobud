<script lang="ts">
	import type { TimelineReviewComment } from "$lib/api/types";
	import { renderMarkdown } from "$lib/utils/markdown";
	import { formatRelativeShort } from "$lib/utils/time";
	import { computeAvatarUrl } from "$lib/utils/avatar";
	import octicons from "@primer/octicons";

	export let comments: TimelineReviewComment[];
	export let count: number | undefined = undefined;

	let expanded = false;

	$: totalCount = count ?? comments.length;
	$: hasMore = totalCount > comments.length;

	function getIconPath(iconName: string): string {
		const icon = octicons[iconName];
		if (!icon || !icon.heights || !icon.heights["16"]) {
			return "";
		}
		return icon.heights["16"].path;
	}

	const commentIconPath = getIconPath("comment-discussion");
	$: chevronPath = getIconPath(expanded ? "chevron-down" : "chevron-right");

	function formatTimestamp(ts?: string): string {
		if (!ts) return "";
		const date = new Date(ts);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	}
</script>

<!-- Toggle card -->
<div class="mt-2 border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden">
	<button
		class="flex items-center gap-1.5 w-full px-3 py-2 text-xs text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors cursor-pointer"
		on:click={() => (expanded = !expanded)}
	>
		<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" class="flex-shrink-0">
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html chevronPath}
		</svg>
		<svg viewBox="0 0 16 16" width="12" height="12" fill="currentColor" class="flex-shrink-0">
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html commentIconPath}
		</svg>
		<span class="font-medium">
			{totalCount} review comment{totalCount === 1 ? "" : "s"}
		</span>
	</button>
</div>

<!-- Expanded comments with left border line -->
{#if expanded}
	<div
		class="ml-4 mt-2 pl-4 border-l border-gray-300 dark:border-gray-800 flex flex-col gap-3 min-w-0 overflow-hidden"
	>
		{#each comments as comment (comment.id)}
			{@const avatarUrl = computeAvatarUrl(comment.author.avatarUrl, comment.author.login)}
			<div class="border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden">
				<!-- Header: file path (tinted) -->
				<div
					class="flex items-center justify-between px-3 py-2 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-800"
				>
					<div class="flex items-center gap-2 min-w-0">
						<svg
							viewBox="0 0 16 16"
							width="14"
							height="14"
							fill="currentColor"
							class="flex-shrink-0 text-gray-400 dark:text-gray-500"
						>
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html getIconPath("file")}
						</svg>
						<code class="text-xs text-gray-700 dark:text-gray-300 font-mono truncate"
							>{comment.path}</code
						>
						{#if comment.outdated}
							<span
								class="text-xs px-1.5 py-0.5 rounded bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 font-medium flex-shrink-0"
								>Outdated</span
							>
						{/if}
					</div>
					{#if comment.htmlUrl}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a
							href={comment.htmlUrl}
							target="_blank"
							rel="noreferrer"
							class="text-xs text-indigo-500 dark:text-indigo-400 hover:underline flex-shrink-0 ml-2"
						>
							View
						</a>
					{/if}
				</div>

				<!-- Diff hunk preview -->
				{#if comment.diffHunk}
					<div
						class="border-b border-gray-200 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-900/50 overflow-x-auto"
					>
						<pre
							class="text-xs font-mono px-3 py-2 text-gray-600 dark:text-gray-400 leading-relaxed whitespace-pre">{#each comment.diffHunk
								.split("\n")
								.slice(-3) as line, i (i)}{#if line.startsWith("+")}<span
										class="text-green-700 dark:text-green-400">{line}</span
									>{:else if line.startsWith("-")}<span class="text-red-700 dark:text-red-400"
										>{line}</span
									>{:else}{line}{/if}
							{/each}</pre>
					</div>
				{/if}

				<!-- Comment body -->
				<div class="px-3 py-3">
					<!-- Author line -->
					<div class="flex items-center gap-2 mb-2">
						<div class="flex-shrink-0">
							{#if avatarUrl}
								<img
									src={avatarUrl}
									alt={`${comment.author.login}'s avatar`}
									class="h-6 w-6 rounded-full bg-gray-300 dark:bg-gray-700"
								/>
							{:else}
								<div
									class="h-6 w-6 rounded-full bg-gray-300 dark:bg-gray-700 flex items-center justify-center"
								>
									<span class="text-[10px] font-bold text-gray-800 dark:text-gray-300">
										{comment.author.login.charAt(0).toUpperCase()}
									</span>
								</div>
							{/if}
						</div>
						<span class="text-xs font-semibold text-gray-700 dark:text-gray-300"
							>{comment.author.login}</span
						>
						{#if comment.createdAt}
							<span class="text-xs text-gray-500 dark:text-gray-400"
								>{formatTimestamp(comment.createdAt)}</span
							>
						{/if}
					</div>

					<!-- Body content (ml aligns with author name: 24px avatar + 8px gap) -->
					<div
						class="ml-8 overflow-x-auto prose dark:prose-invert prose-sm max-w-none prose-p:text-gray-700 dark:prose-p:text-gray-300 prose-headings:text-gray-900 dark:prose-headings:text-gray-300 prose-headings:font-semibold prose-a:text-indigo-600 dark:prose-a:text-indigo-300 prose-a:no-underline prose-link-hover prose-strong:text-gray-900 dark:prose-strong:text-gray-300 prose-strong:font-semibold prose-code:text-gray-800 dark:prose-code:text-gray-300 prose-code:bg-gray-100 dark:prose-code:bg-gray-900 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:before:content-[''] prose-code:after:content-[''] prose-pre:text-gray-800 dark:prose-pre:text-gray-300 prose-pre:bg-gray-100 dark:prose-pre:bg-[#161b22] prose-blockquote:text-gray-600 dark:prose-blockquote:text-gray-400 prose-ul:text-gray-700 dark:prose-ul:text-gray-300 prose-ol:text-gray-700 dark:prose-ol:text-gray-300 prose-li:text-gray-700 dark:prose-li:text-gray-300"
					>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html renderMarkdown(comment.body)}
					</div>
				</div>
			</div>
		{/each}
		{#if hasMore}
			<div class="text-xs text-gray-500 dark:text-gray-500">
				Showing {comments.length} of {totalCount} comments
			</div>
		{/if}
	</div>
{/if}
