<script lang="ts">
	import type { NotificationTimelineItem } from "$lib/api/types";
	import { renderMarkdown } from "$lib/utils/markdown";
	import { formatRelativeShort } from "$lib/utils/time";
	import { computeAvatarUrl } from "$lib/utils/avatar";

	type Reply = NonNullable<NotificationTimelineItem["replies"]>[number];

	export let replies: Reply[];
	export let hasMoreReplies: boolean | undefined = undefined;
	export let parentHtmlUrl: string | undefined = undefined;

	$: showLoadMoreLink = hasMoreReplies && parentHtmlUrl;

	function formatTimestamp(ts?: string): string {
		if (!ts) return "";
		const date = new Date(ts);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	}
</script>

<div class="mt-2">
	<!-- Indented replies with thread line -->
	<div class="ml-3 flex gap-3">
		<!-- Thread line -->
		<div class="flex-shrink-0 w-0.5 bg-gray-300 dark:bg-gray-800 rounded-full"></div>

		<!-- Replies column -->
		<div class="flex-1 min-w-0 flex flex-col gap-2">
			{#if showLoadMoreLink}
				<div class="px-1 py-1">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a
						href={parentHtmlUrl}
						target="_blank"
						rel="noreferrer"
						class="text-xs text-indigo-500 dark:text-indigo-400 hover:underline"
					>
						View earlier replies on GitHub
					</a>
				</div>
			{/if}

			<div class="border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden">
			<div class="divide-y divide-gray-100 dark:divide-gray-800">
				{#each replies as reply (reply.id)}
					{@const avatarUrl = computeAvatarUrl(reply.author.avatarUrl, reply.author.login)}
					<div class="px-4 py-3 flex gap-2.5">
						<!-- Reply avatar (smaller) -->
						<div class="flex-shrink-0">
							{#if avatarUrl}
								<img
									src={avatarUrl}
									alt={`${reply.author.login}'s avatar`}
									class="h-6 w-6 rounded-full bg-gray-300 dark:bg-gray-700"
								/>
							{:else}
								<div
									class="h-6 w-6 rounded-full bg-gray-300 dark:bg-gray-700 flex items-center justify-center"
								>
									<span class="text-[10px] font-bold text-gray-800 dark:text-gray-300">
										{reply.author.login.charAt(0).toUpperCase()}
									</span>
								</div>
							{/if}
						</div>

						<!-- Reply content -->
						<div class="flex-1 min-w-0">
							<div
								class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400 mb-1"
							>
								<span class="font-semibold text-gray-700 dark:text-gray-300"
									>{reply.author.login}</span
								>
								{#if reply.createdAt}
									<span>{formatTimestamp(reply.createdAt)}</span>
								{/if}
							</div>
							<div
								class="prose dark:prose-invert prose-sm max-w-none prose-p:text-gray-700 dark:prose-p:text-gray-400 prose-p:my-0.5 text-sm"
							>
								<!-- eslint-disable-next-line svelte/no-at-html-tags -->
								{@html renderMarkdown(reply.body)}
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
		</div>
	</div>
</div>
