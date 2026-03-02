<script lang="ts">
	import type { TimelineReviewComment } from "$lib/api/types";
	import { renderMarkdown } from "$lib/utils/markdown";
	import { formatRelativeShort } from "$lib/utils/time";
	import { computeAvatarUrl } from "$lib/utils/avatar";
	import octicons from "@primer/octicons";
	import hljs from "highlight.js";

	export let comments: TimelineReviewComment[];
	export let count: number | undefined = undefined;

	let expanded = false;

	// Count includes replies nested under each comment
	$: totalCount = count ?? comments.reduce((sum, c) => sum + 1 + (c.replies?.length ?? 0), 0);
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

	const extToLang: Record<string, string> = {
		ts: "typescript",
		tsx: "typescript",
		js: "javascript",
		jsx: "javascript",
		py: "python",
		rb: "ruby",
		rs: "rust",
		yml: "yaml",
		md: "markdown",
		sh: "bash",
		zsh: "bash",
	};

	function langFromPath(path: string): string | undefined {
		const ext = path.split(".").pop()?.toLowerCase();
		if (!ext) return undefined;
		const mapped = extToLang[ext] ?? ext;
		return hljs.getLanguage(mapped) ? mapped : undefined;
	}

	/**
	 * Split highlighted HTML on newlines while keeping <span> tags balanced.
	 * hljs only emits <span class="...">...</span> so we only track those.
	 */
	function splitHighlightedLines(html: string): string[] {
		const rawLines = html.split("\n");
		const result: string[] = [];
		let openTags: string[] = []; // stack of full opening tags e.g. '<span class="hljs-keyword">'

		for (const raw of rawLines) {
			// Re-open any spans still open from previous lines
			const prefix = openTags.join("");

			// Track opens and closes in this line
			const tagRe = /<\/?span[^>]*>/g;
			let m: RegExpExecArray | null;
			while ((m = tagRe.exec(raw)) !== null) {
				if (m[0].startsWith("</")) {
					openTags.pop();
				} else {
					openTags.push(m[0]);
				}
			}

			// Close any still-open spans at end of this line
			const suffix = "</span>".repeat(openTags.length);
			result.push(prefix + raw + suffix);
		}
		return result;
	}

	function formatDiffHunk(diffHunk: string, path: string): string {
		const lines = diffHunk.split("\n").slice(-3);

		// Classify each line and strip the +/- prefix for highlighting
		const classified = lines.map((line) => {
			if (line.startsWith("+")) return { type: "add" as const, code: line.slice(1) };
			if (line.startsWith("-")) return { type: "del" as const, code: line.slice(1) };
			if (line.startsWith(" ")) return { type: "ctx" as const, code: line.slice(1) };
			return { type: "ctx" as const, code: line };
		});

		// Highlight the joined code block
		const codeBlock = classified.map((l) => l.code).join("\n");
		const lang = langFromPath(path);
		let highlighted: string;
		try {
			highlighted = lang
				? hljs.highlight(codeBlock, { language: lang }).value
				: hljs.highlightAuto(codeBlock).value;
		} catch {
			highlighted = codeBlock.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
		}

		// Split back into individual lines with balanced tags
		const hlLines = splitHighlightedLines(highlighted);

		return classified
			.map((info, i) => {
				const content = hlLines[i] ?? "";
				const bgClass =
					info.type === "add"
						? "diff-line-add"
						: info.type === "del"
							? "diff-line-del"
							: "diff-line-ctx";
				return `<div class="${bgClass}">${content}</div>`;
			})
			.join("");
	}

	function formatTimestamp(ts?: string): string {
		if (!ts) return "";
		const date = new Date(ts);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	}

	const maxVisibleReplies = 5;
</script>

<!-- Toggle section -->
<div class="border-t border-gray-200 dark:border-gray-800">
	<button
		class="flex items-center gap-1.5 w-full px-3 py-2 text-xs text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors cursor-pointer"
		aria-expanded={expanded}
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

	<!-- Expanded comments -->
	{#if expanded}
		<div class="px-3 pb-3 pt-1 flex flex-col gap-3 min-w-0 overflow-hidden">
			{#each comments as comment (comment.id)}
				{@const avatarUrl = computeAvatarUrl(comment.author.avatarUrl, comment.author.login)}
				{@const replies = comment.replies ?? []}
				{@const hasHiddenReplies = replies.length > maxVisibleReplies}
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
						<div class="diff-hunk border-b border-gray-200 dark:border-gray-800 overflow-x-auto">
							<pre
								class="hljs text-xs font-mono !p-0 leading-relaxed whitespace-pre !rounded-none">{@html formatDiffHunk(
									comment.diffHunk,
									comment.path
								)}</pre>
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

					<!-- Replies -->
					{#if replies.length > 0}
						<div
							class="divide-y divide-gray-100 dark:divide-gray-800 border-t border-gray-200 dark:border-gray-800"
						>
							{#if hasHiddenReplies && comment.htmlUrl}
								<div class="px-4 py-2">
									<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
									<a
										href={comment.htmlUrl}
										target="_blank"
										rel="noreferrer"
										class="text-xs text-indigo-500 dark:text-indigo-400 hover:underline"
									>
										View earlier replies on GitHub
									</a>
								</div>
							{/if}

							{#each replies.slice(-maxVisibleReplies) as reply (reply.id)}
								{@const replyAvatarUrl = computeAvatarUrl(
									reply.author.avatarUrl,
									reply.author.login
								)}
								<div class="px-4 py-3 flex gap-2.5">
									<div class="flex-shrink-0">
										{#if replyAvatarUrl}
											<img
												src={replyAvatarUrl}
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
											class="overflow-x-auto prose dark:prose-invert prose-sm max-w-none prose-p:text-gray-700 dark:prose-p:text-gray-400 prose-p:my-0.5 text-sm"
										>
											<!-- eslint-disable-next-line svelte/no-at-html-tags -->
											{@html renderMarkdown(reply.body)}
										</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/each}
			{#if hasMore}
				<div class="text-xs text-gray-500 dark:text-gray-500">
					Showing {comments.length} of {totalCount} comments
				</div>
			{/if}
		</div>
	{/if}
</div>
