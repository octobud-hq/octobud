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
	import octicons from "@primer/octicons";

	export let item: NotificationTimelineItem;
	export let showThread: boolean = true;
	export let isLastItem: boolean = false;

	$: authorLogin = item.author?.login || item.actor?.login || "Unknown";
	$: authorAvatar = item.author?.avatarUrl || item.actor?.avatarUrl;
	$: authorInitial = authorLogin.charAt(0).toUpperCase();
	$: eventTimestamp = item.submittedAt || item.createdAt || item.updatedAt || item.timestamp;

	// Helper to safely format timestamp with relative time
	$: formattedTimestamp = (() => {
		if (!eventTimestamp) return "";
		const date = new Date(eventTimestamp);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	})();

	// Check if this event should have a badge
	$: eventBadgeConfig = getEventBadgeConfig(item.type);

	// Format event label based on type
	$: eventLabel = getEventLabel(item.type);

	// Helper to get the SVG path from an octicon
	function getIconPath(iconName: string): string {
		const icon = octicons[iconName];
		if (!icon || !icon.heights || !icon.heights["16"]) {
			return "";
		}
		return icon.heights["16"].path;
	}

	function getEventBadgeConfig(type: string) {
		switch (type) {
			case "merged":
				return {
					label: "Merged",
					iconPath: getIconPath("git-merge"),
					useIcon: true,
					bgClass: "bg-purple-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "closed":
				return {
					label: "Closed",
					iconPath: getIconPath("issue-closed"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "added_to_merge_queue":
				return {
					label: "Added to merge queue",
					iconPath: getIconPath("git-merge"),
					useIcon: true,
					bgClass: "bg-blue-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "removed_from_merge_queue":
				return {
					label: "Removed from merge queue",
					iconPath: getIconPath("git-merge"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "auto_merge_enabled":
				return {
					label: "Auto-merge enabled",
					iconPath: getIconPath("git-merge"),
					useIcon: true,
					bgClass: "bg-blue-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "auto_merge_disabled":
				return {
					label: "Auto-merge disabled",
					iconPath: getIconPath("git-merge"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "deployed":
				return {
					label: "Deployed",
					iconPath: getIconPath("cloud"),
					useIcon: true,
					bgClass: "bg-indigo-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "head_ref_deleted":
				return {
					label: "Branch deleted",
					iconPath: getIconPath("git-branch"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "ready_for_review":
				return {
					label: "Ready for review",
					iconPath: getIconPath("code-review"),
					useIcon: true,
					bgClass: "bg-blue-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "marked_as_duplicate":
				return {
					label: "Marked as duplicate",
					iconPath: getIconPath("issue-opened"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "assigned":
				return {
					label: "Assigned",
					iconPath: getIconPath("person"),
					useIcon: true,
					bgClass: "bg-blue-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "unassigned":
				return {
					label: "Unassigned",
					iconPath: getIconPath("person"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			case "head_ref_force_pushed":
				return {
					label: "Force pushed to branch",
					iconPath: getIconPath("git-branch"),
					useIcon: true,
					bgClass: "bg-gray-600",
					textClass: "text-white",
					iconClass: "text-white",
				};
			default:
				return null;
		}
	}

	function getEventLabel(type: string): string {
		switch (type) {
			case "merged":
				return "merged this";
			case "enqueued":
			case "queued":
				return "added to merge queue";
			case "dequeued":
				return "removed from merge queue";
			case "closed":
				return "closed this";
			case "reopened":
				return "reopened this";
			case "assigned":
				return "was assigned";
			case "unassigned":
				return "was unassigned";
			case "review_requested":
				return "requested a review";
			case "review_request_removed":
				return "removed a review request";
			case "ready_for_review":
				return "marked as ready for review";
			case "converted_to_draft":
				return "converted to draft";
			case "deployed":
				return "deployed this";
			case "referenced":
				return "referenced this";
			case "cross-referenced":
				return "cross-referenced this";
			case "labeled":
				return "added a label";
			case "unlabeled":
				return "removed a label";
			case "milestoned":
				return "added this to a milestone";
			case "demilestoned":
				return "removed this from a milestone";
			case "renamed":
				return "renamed this";
			case "locked":
				return "locked the conversation";
			case "unlocked":
				return "unlocked the conversation";
			case "head_ref_deleted":
				return "deleted the branch";
			case "head_ref_restored":
				return "restored the branch";
			case "connected":
				return "connected this";
			case "disconnected":
				return "disconnected this";
			case "added_to_project_v2":
				return "added this to a project";
			case "removed_from_project_v2":
				return "removed this from a project";
			case "project_v2_item_status_changed":
				return "changed the project status";
			default:
				return type;
		}
	}

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

	<!-- Event info -->
	<div class="flex-1 min-w-0 py-1">
		<div class="text-sm text-gray-600 dark:text-gray-400">
			<span class="font-semibold text-gray-700 dark:text-gray-300">{authorLogin}</span>
			{#if eventBadgeConfig}
				<span
					class="inline-flex items-center gap-1 px-2 py-0.5 mx-1 text-xs font-medium rounded-md {eventBadgeConfig.bgClass} {eventBadgeConfig.textClass}"
				>
					{#if eventBadgeConfig.useIcon && eventBadgeConfig.iconPath}
						<svg
							viewBox="0 0 16 16"
							width="12"
							height="12"
							class={eventBadgeConfig.iconClass}
							fill="currentColor"
						>
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html eventBadgeConfig.iconPath}
						</svg>
					{/if}
					<span>{eventBadgeConfig.label}</span>
				</span>
			{:else}
				<span class="mx-1">{eventLabel}</span>
			{/if}
			{#if formattedTimestamp}
				<span class="text-gray-600 dark:text-gray-500">
					{formattedTimestamp}
				</span>
			{/if}
		</div>
	</div>
</div>
