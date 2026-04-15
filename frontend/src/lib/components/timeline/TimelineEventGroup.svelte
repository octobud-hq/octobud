<script lang="ts">
	import type { TimelineGroup } from "$lib/utils/timelineGrouping";
	import { formatRelativeShort } from "$lib/utils/time";
	import { computeAvatarUrl } from "$lib/utils/avatar";

	export let group: TimelineGroup;
	export let showThread: boolean = true;
	export let isLastItem: boolean = false;

	$: actorLogin = group.actor?.login || "Unknown";
	$: authorInitial = actorLogin.charAt(0).toUpperCase();
	$: authorAvatar = computeAvatarUrl(group.actor?.avatarUrl, actorLogin);

	$: formattedTimestamp = (() => {
		if (!group.timestamp) return "";
		const date = new Date(group.timestamp);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	})();

	// Inline label badges for label groups
	$: labelPreviews =
		group.groupType === "labeled" || group.groupType === "unlabeled"
			? group.items.filter((i) => i.label).map((i) => i.label!)
			: [];
</script>

<div class="flex gap-3 pt-4">
	<!-- Avatar and thread line -->
	<div class="flex flex-col items-center flex-shrink-0 relative z-10" style="width: 40px;">
		<div class="flex-shrink-0">
			{#if authorAvatar}
				<img
					src={authorAvatar}
					alt={`${actorLogin}'s avatar`}
					class="h-8 w-8 rounded-full bg-gray-300 dark:bg-gray-700 ring-2 ring-white dark:ring-gray-950"
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

		{#if showThread && !isLastItem}
			<div class="flex-1 w-0.5 bg-gray-300 dark:bg-gray-800 min-h-4 -z-10"></div>
		{/if}
	</div>

	<!-- Group content -->
	<div class="flex-1 min-w-0 py-1">
		<div class="text-sm text-gray-600 dark:text-gray-400">
			<span class="font-semibold text-gray-700 dark:text-gray-300">{actorLogin}</span>
			{#each group.summary as part, i (i)}{#if part.isMention}<span
						class="font-medium text-gray-700 dark:text-gray-300">@{part.text}</span
					>{:else}{part.text}{/if}{/each}
			<!-- Inline label badges -->
			{#if labelPreviews.length > 0}
				{#each labelPreviews as label (label.name)}
					<span
						class="github-label-chip inline-flex items-center px-1.5 py-0 text-xs font-medium rounded-full border"
						style="--label-color: #{label.color}; background-color: #{label.color}20; border-color: #{label.color}40;"
					>
						{label.name}
					</span>
				{/each}
			{/if}
			{#if formattedTimestamp}
				<span class="text-gray-600 dark:text-gray-500">
					{formattedTimestamp}
				</span>
			{/if}
		</div>
	</div>
</div>
