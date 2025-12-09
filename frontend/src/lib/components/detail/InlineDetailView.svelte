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

	import type { Notification, NotificationDetail } from "$lib/api/types";
	import { fade } from "svelte/transition";
	import { onDestroy, getContext } from "svelte";
	import { normalizeReason, parseSubjectMetadata } from "$lib/utils/notificationHelpers";
	import { formatRelativeShort } from "$lib/utils/time";
	import { getNotificationIcon } from "$lib/utils/notificationIcons";
	import { getNotificationChip } from "$lib/utils/notificationChips";
	import { formatSnoozeMessage } from "$lib/utils/snoozeFormat";
	import type { TimelineController } from "$lib/state/timelineController";
	import TimelineThread from "$lib/components/timeline/TimelineThread.svelte";
	import DetailActionBar from "./DetailActionBar.svelte";
	import { getNotificationTypeConfig } from "$lib/utils/notificationTypeConfig";
	import { renderMarkdown } from "$lib/utils/markdown";
	import type { NotificationPageController } from "$lib/state/types";
	import { currentTime } from "$lib/stores/timeStore";
	import octicons from "@primer/octicons";

	// Subscribe to time store to trigger re-renders when time updates
	// This ensures relative timestamps for same-day notifications stay fresh
	// Compute formatted timestamp reactively - depends on both notification date and current time
	// Use subject's creation date (when PR/Issue was opened) instead of notification update time
	$: formattedTimestamp = (() => {
		if (!$currentTime || !notification) return "";

		// Prefer subject's creation date (when PR/Issue was actually created/opened)
		const timestamp = detail?.subject?.createdAt ?? notification.updatedAt;
		if (!timestamp) return "";
		const date = new Date(timestamp);
		return !isNaN(date.getTime()) ? formatRelativeShort(date) : "";
	})();

	// Get page controller from context
	const pageController = getContext<NotificationPageController>("notificationPageController");

	// Read notification directly from controller store (reactive)
	// This ensures it updates automatically when the store updates, eliminating prop drilling issues
	// Override prop with store value if available
	const currentDetailNotificationStore = pageController.stores.currentDetailNotification;
	$: notification = $currentDetailNotificationStore ?? notification;
	export let detail: NotificationDetail | null = null;
	export let loading = false;
	export let showingStaleData = false;
	export let isRefreshing = false;
	export let isSplitView: boolean = false;
	export let timelineController: TimelineController | undefined = undefined;
	export let markingRead = false;
	export let archiving = false;
	export let hideBackButton = false;

	// Delayed loading state to avoid jank on fast loads
	let showLoadingSkeleton = false;
	let loadingTimeoutId: ReturnType<typeof setTimeout> | null = null;

	// Auto-mark-as-read timer for split view (2 seconds)
	let autoMarkReadTimeoutId: ReturnType<typeof setTimeout> | null = null;
	let currentViewingNotificationId: string | null = null;
	let wasInitiallyUnread = false;

	$: {
		if (loading) {
			// Start a 200ms timer before showing skeleton
			loadingTimeoutId = setTimeout(() => {
				showLoadingSkeleton = true;
			}, 200);
		} else {
			// Clear timer and hide skeleton immediately when loading completes
			if (loadingTimeoutId !== null) {
				clearTimeout(loadingTimeoutId);
				loadingTimeoutId = null;
			}
			showLoadingSkeleton = false;
		}
	}

	// SPLIT MODE: Auto-mark-as-read after 2 seconds
	$: if (notification) {
		const notificationId = notification.githubId ?? notification.id;

		// Detect if we've switched to a different notification
		if (notificationId !== currentViewingNotificationId) {
			// Clear any existing timer
			if (autoMarkReadTimeoutId !== null) {
				clearTimeout(autoMarkReadTimeoutId);
				autoMarkReadTimeoutId = null;
			}

			// Reset tracking for the new notification
			currentViewingNotificationId = notificationId;

			// Capture initial read state - only start timer if it's initially unread
			wasInitiallyUnread = !notification.isRead;
		}

		// Clear timer if notification is already read (might have been marked read externally)
		// Also reset wasInitiallyUnread so we don't restart the timer if user marks it unread again
		if (notification.isRead) {
			if (autoMarkReadTimeoutId !== null) {
				clearTimeout(autoMarkReadTimeoutId);
				autoMarkReadTimeoutId = null;
			}
			// Reset wasInitiallyUnread so marking unread again won't trigger auto-mark
			wasInitiallyUnread = false;
		}

		// Only set up timer if:
		// 1. In split view mode
		// 2. Notification was initially unread when we started viewing it
		// 3. Notification is currently unread
		// 4. Timer hasn't been set up yet
		if (
			isSplitView &&
			wasInitiallyUnread &&
			!notification.isRead &&
			autoMarkReadTimeoutId === null
		) {
			// Set up new timer for 2 seconds
			autoMarkReadTimeoutId = setTimeout(async () => {
				try {
					await pageController.actions.markRead(notification);
				} catch (err) {}
			}, 1500);
		} else if (!isSplitView) {
			// Clear timer if we switch out of split mode
			if (autoMarkReadTimeoutId !== null) {
				clearTimeout(autoMarkReadTimeoutId);
				// eslint-disable-next-line svelte/infinite-reactive-loop
				autoMarkReadTimeoutId = null;
			}
		}
	}

	onDestroy(() => {
		if (loadingTimeoutId !== null) {
			clearTimeout(loadingTimeoutId);
		}
		if (autoMarkReadTimeoutId !== null) {
			clearTimeout(autoMarkReadTimeoutId);
		}
	});

	// Check if body has visible content (not just HTML comments or whitespace)
	function hasVisibleContent(body: string | undefined | null): boolean {
		if (!body || typeof body !== "string") return false;

		// Remove HTML comments (<!-- ... -->)
		const withoutComments = body.replace(/<!--[\s\S]*?-->/g, "");

		// Remove markdown code blocks (```...```) and inline code (`...`)
		const withoutCodeBlocks = withoutComments
			.replace(/```[\s\S]*?```/g, "")
			.replace(/`[^`]*`/g, "");

		// Remove HTML tags and check if there's any text content
		const withoutHtml = withoutCodeBlocks.replace(/<[^>]*>/g, "");

		// Check if there's any non-whitespace content
		return withoutHtml.trim().length > 0;
	}

	$: subjectTypeLabel = (() => {
		const type = detail?.notification.subjectType || notification.subjectType || "Notification";
		const normalizedType = type.toLowerCase().replace(/[-_\s]/g, "");
		switch (normalizedType) {
			case "pullrequest":
			case "pr":
				return "Pull Request";
			case "issue":
				return "Issue";
			case "discussion":
				return "Discussion";
			case "commit":
				return "Commit";
			case "release":
				return "Release";
			case "checkrun":
			case "checksuite":
				return "CI Activity";
			case "repositoryvulnerabilityalert":
			case "securityalert":
				return "Security Alert";
			default:
				return type;
		}
	})();

	// Get notification type configuration
	$: typeConfig = detail?.notification.subjectType
		? getNotificationTypeConfig(detail.notification.subjectType)
		: notification.subjectType
			? getNotificationTypeConfig(notification.subjectType)
			: null;

	// Icon and status bar color from shared util
	$: iconConfig = detail
		? getNotificationIcon(
				detail.notification.subjectType,
				detail.notification.subjectRaw,
				detail.notification.isRead,
				detail.notification.subjectState,
				detail.notification.subjectMerged
			)
		: getNotificationIcon(
				notification.subjectType,
				notification.subjectRaw,
				notification.isRead,
				notification.subjectState,
				notification.subjectMerged
			);

	// Chip configuration for PRs and Issues
	$: chipConfig = detail
		? getNotificationChip(
				detail.notification.subjectType,
				detail.notification.subjectRaw,
				detail.notification.subjectState,
				detail.notification.subjectMerged
			)
		: getNotificationChip(
				notification.subjectType,
				notification.subjectRaw,
				notification.subjectState,
				notification.subjectMerged
			);

	// Parse subject metadata
	$: subject = detail?.notification.subjectRaw
		? parseSubjectMetadata(detail.notification.subjectRaw)
		: notification.subjectRaw
			? parseSubjectMetadata(notification.subjectRaw)
			: null;

	$: reasonLabel = detail?.notification.reason
		? normalizeReason(detail.notification.reason)
		: notification.reason
			? normalizeReason(notification.reason)
			: null;

	$: commentItems = timelineController?.stores.items;
	$: hasLoadedTimeline = commentItems ? $commentItems.length > 0 : false;

	// Helper to check if a type is CI activity
	function checkIsCIActivity(subjectType: string | undefined | null): boolean {
		if (!subjectType) return false;
		const type = subjectType.toLowerCase().replace(/[-_\s]/g, "");
		return type === "checkrun" || type === "checksuite";
	}

	// Compute GitHub URLs
	$: githubUrl = (() => {
		const subjectType = detail?.notification.subjectType || notification.subjectType;
		const repoFullName = detail?.notification.repoFullName || notification.repoFullName;

		// For CI activity, always use the actions page URL (subject data is not available)
		if (checkIsCIActivity(subjectType) && repoFullName) {
			return `https://github.com/${repoFullName}/actions`;
		}

		// For other types, try to get URL from detail/notification
		if (detail) {
			// Prefer html_url from subject, then htmlUrl from notification
			// Note: githubUrl is the API thread URL, not suitable for display
			const url = detail.subject?.url ?? detail.notification.htmlUrl;
			if (url && !url.startsWith("https://api.github.com")) {
				return url;
			}
		}

		// Fallback: use notification's htmlUrl (constructed from subject data)
		if (notification.htmlUrl && !notification.htmlUrl.startsWith("https://api.github.com")) {
			return notification.htmlUrl;
		}

		return null;
	})();

	$: isPullRequest =
		detail?.notification.subjectType === "pull_request" ||
		notification.subjectType === "pull_request";

	$: filesUrl = isPullRequest && githubUrl ? `${githubUrl}/files` : null;

	// Conversation button text based on notification type
	$: conversationButtonText =
		detail?.notification.subjectType === "issue" || notification.subjectType === "issue"
			? "Issue"
			: detail?.notification.subjectType === "discussion" ||
				  notification.subjectType === "discussion"
				? "Discussion"
				: "Conversation";

	// Author info for comment header
	$: authorName =
		detail?.subject?.author ??
		detail?.notification.authorLogin ??
		notification.authorLogin ??
		"Unknown";
	$: isBot =
		authorName.toLowerCase().includes("[bot]") || authorName.toLowerCase().endsWith("-bot");
	$: authorInitial = authorName.charAt(0).toUpperCase();
	$: authorAvatarUrl =
		authorName !== "Unknown" ? `https://github.com/${encodeURIComponent(authorName)}.png` : null;

	let avatarLoadFailed = false;
	let previousGithubId: string | null = null;

	// Reset avatar error state and timeline when switching to a different notification
	$: if (notification) {
		avatarLoadFailed = false;

		// Only reset timeline controller when switching to a DIFFERENT notification
		const currentGithubId = notification.githubId;
		if (timelineController && currentGithubId !== previousGithubId) {
			timelineController.actions.reset();
			previousGithubId = currentGithubId;
		}
	}

	function handleAvatarError() {
		avatarLoadFailed = true;
	}

	// Helper to get the SVG path from an octicon
	function getIconPath(iconName: string): string {
		const icon = octicons[iconName];
		if (!icon || !icon.heights || !icon.heights["16"]) {
			return "";
		}
		return icon.heights["16"].path;
	}

	// Get icon paths for buttons
	$: conversationIconPath =
		detail?.notification.subjectType === "issue" || notification.subjectType === "issue"
			? getIconPath("issue-opened")
			: getIconPath("comment-discussion");
	const fileDiffIconPath = getIconPath("file-diff");

	// Check if this is a CI activity notification (CheckRun/CheckSuite)
	$: isCIActivity = checkIsCIActivity(detail?.notification.subjectType || notification.subjectType);
</script>

{#if notification}
	<div class="flex h-full flex-col overflow-hidden bg-white dark:bg-gray-950">
		<!-- Fixed Action Bar -->
		<DetailActionBar {notification} {isSplitView} {markingRead} {archiving} {hideBackButton} />

		<!-- Scrollable Content -->
		<div class="flex-1 overflow-y-auto">
			<div class="px-8 pb-16">
				<div class="mx-auto max-w-7xl">
					<!-- Header section - use notification data (always available) not detail (loaded async) -->
					<div class="flex-1 space-y-2 pb-3 pt-6">
						<!-- Top Row: Icon/Chip + Type Label + Status Badges -->
						<div class="flex items-center gap-2">
							{#if chipConfig}
								<!-- GitHub-style chip for PRs and Issues -->
								<div
									class="flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-full {chipConfig.bgColorClass}"
								>
									<svg
										class="w-4 h-4 {chipConfig.iconColorClass}"
										viewBox="0 0 16 16"
										fill="currentColor"
										aria-hidden="true"
									>
										<!-- eslint-disable-next-line svelte/no-at-html-tags -->
										{@html chipConfig.path}
									</svg>
									<span class="text-sm font-semibold {chipConfig.textColorClass}">
										{chipConfig.stateLabel}
									</span>
								</div>
							{:else}
								<!-- Regular icon for other notification types -->
								<div class="flex-shrink-0">
									<svg
										class="w-5 h-5 {iconConfig.colorClass}"
										viewBox="0 0 16 16"
										fill="currentColor"
										aria-hidden="true"
									>
										<!-- eslint-disable-next-line svelte/no-at-html-tags -->
										{@html iconConfig.path}
									</svg>
								</div>
							{/if}
							<span
								class="rounded-md bg-gray-100 dark:bg-gray-800/80 px-2.5 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300"
							>
								{subjectTypeLabel}
							</span>

							<!-- Status Badges -->
							{#if notification.snoozedUntil}
								<span
									class="rounded-md bg-orange-500/10 dark:bg-orange-500/20 px-2.5 py-1.5 text-xs font-medium text-orange-600 dark:text-orange-300"
								>
									{formatSnoozeMessage(notification.snoozedUntil, notification.snoozedAt)}
								</span>
							{/if}
							{#if notification.muted}
								<span
									class="rounded-md bg-violet-500/10 dark:bg-violet-500/20 px-2.5 py-1.5 text-xs font-medium text-violet-600 dark:text-violet-300"
								>
									Muted
								</span>
							{/if}
							{#if notification.archived}
								<span
									class="rounded-md bg-blue-500/10 dark:bg-blue-500/20 px-2.5 py-1.5 text-xs font-medium text-blue-600 dark:text-blue-300"
								>
									Archived
								</span>
							{/if}
							{#if notification.filtered}
								<span
									class="rounded-md bg-gray-500/20 px-2.5 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300"
								>
									Skipped Inbox
								</span>
							{/if}

							<!-- Tags -->
							{#if notification.tags && notification.tags.length > 0}
								{#each notification.tags as tag (tag.id)}
									<span
										class="rounded-full border px-2 py-1.5 text-[11px] font-semibold"
										style="background-color: {tag.color}20; color: {tag.color}; border-color: {tag.color}"
										title={tag.name}
									>
										{tag.name}
									</span>
								{/each}
							{/if}
						</div>

						<!-- Second Row: Title -->
						<h3 class="text-3xl font-normal leading-snug text-gray-900 dark:text-gray-100 pt-1">
							{notification.subjectTitle}
						</h3>

						<!-- Third Row: Repo + Number + Author + Reason + Timestamp + Other Metadata -->
						<div class="flex flex-wrap items-center gap-1.5 text-lg">
							<span
								class="text-xl text-gray-600 dark:text-gray-400 font-light dark:font-light pr-1"
							>
								{notification.repoFullName}
							</span>
							{#if subject?.number}
								<span
									class="rounded-md bg-gray-100 dark:bg-gray-800/80 px-2 text-gray-700 dark:text-gray-300 font-medium text-[12px]"
								>
									#{subject.number}
								</span>
							{/if}
							{#if subject?.author || notification.authorLogin}
								<span class="flex items-center gap-1 text-[12px] text-gray-700 dark:text-gray-600">
									<span>by</span>
									<span class="text-gray-600 dark:text-gray-500">
										{subject?.author ?? notification.authorLogin}
									</span>
								</span>
							{/if}
							{#if reasonLabel}
								<span
									class="rounded-md bg-blue-500/10 dark:bg-blue-500/20 px-2 text-blue-600 dark:text-blue-300 font-medium text-[12px]"
								>
									{reasonLabel}
								</span>
							{/if}
							<span class="text-[12px] font-medium text-gray-600 dark:text-gray-500">
								{formattedTimestamp}
							</span>
							{#if detail?.subject?.draft}
								<span
									class="rounded-md bg-gray-200 dark:bg-gray-700/60 px-2 text-gray-600 dark:text-gray-400 text-[12px]"
								>
									Draft
								</span>
							{/if}
							<!-- Don't show state badges for PRs and Issues since they're in the chip -->
							{#if notification.subjectType !== "pull_request" && notification.subjectType !== "issue"}
								{@const currentMerged =
									detail?.notification.subjectMerged ?? notification.subjectMerged}
								{@const currentState =
									detail?.notification.subjectState ?? notification.subjectState}
								{#if currentMerged}
									<span
										class="rounded-md bg-purple-500/10 dark:bg-purple-500/20 py-1 px-2 text-purple-600 dark:text-purple-300 font-medium text-[12px]"
									>
										Merged
									</span>
								{:else if currentState === "closed"}
									<span
										class="rounded-md bg-red-500/10 dark:bg-red-500/20 px-2 py-1 text-red-600 dark:text-red-300 font-medium text-[12px]"
									>
										Closed
									</span>
								{:else if currentState === "open"}
									<span
										class="rounded-md bg-blue-500/10 dark:bg-blue-500/20 px-2 py-1 text-blue-600 dark:text-blue-300 font-medium text-[12px]"
									>
										Open
									</span>
								{/if}
							{/if}
							{#if detail?.subject?.assignees && detail.subject.assignees.length > 0}
								<span class="flex items-center gap-1 text-[12px] text-gray-700 dark:text-gray-600">
									<span>assigned:</span>
									<span class="text-gray-600 dark:text-gray-500">
										{detail.subject.assignees.slice(0, 2).join(", ")}{detail.subject.assignees
											.length > 2
											? ` +${detail.subject.assignees.length - 2}`
											: ""}
									</span>
								</span>
							{/if}
							{#if detail?.subject?.labels && detail.subject.labels.length > 0}
								{#each detail.subject.labels.slice(0, 3) as label (label)}
									<span
										class="rounded-md bg-blue-500/10 dark:bg-blue-500/20 px-2 text-blue-600 dark:text-blue-300 text-[12px]"
									>
										{label}
									</span>
								{/each}
								{#if detail.subject.labels.length > 3}
									<span class="text-[12px] text-gray-600">
										+{detail.subject.labels.length - 3}
									</span>
								{/if}
							{/if}
						</div>
					</div>

					<!-- Open on GitHub buttons -->
					{#if detail}
						<!-- eslint-disable svelte/no-navigation-without-resolve -->
						<div class="mt-8 mb-6 flex items-center gap-2">
							{#if isCIActivity}
								<!-- CI Activity: Open in Actions button -->
								<a
									class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 px-2.5 py-1.5 text-sm font-light text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-800 dark:hover:text-gray-200"
									href={githubUrl}
									target="_blank"
									rel="noreferrer"
								>
									<!-- Play/Actions icon -->
									<svg
										class="h-4 w-4"
										viewBox="0 0 16 16"
										fill="currentColor"
										xmlns="http://www.w3.org/2000/svg"
									>
										<!-- eslint-disable-next-line svelte/no-at-html-tags -->
										{@html getIconPath("play")}
									</svg>
									<span>Open in Actions</span>
									<!-- External link icon -->
									<svg
										class="h-4 w-4"
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
								<!-- Other types: Conversation button -->
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
								<a
									class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 px-2.5 py-1.5 text-sm font-light text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-800 dark:hover:text-gray-200"
									href={githubUrl}
									target="_blank"
									rel="noreferrer"
								>
									<!-- Conversation icon -->
									<svg
										class="h-4 w-4"
										viewBox="0 0 16 16"
										fill="currentColor"
										xmlns="http://www.w3.org/2000/svg"
									>
										<!-- eslint-disable-next-line svelte/no-at-html-tags -->
										{@html conversationIconPath}
									</svg>
									<span>{conversationButtonText}</span>
									<!-- External link icon -->
									<svg
										class="h-4 w-4"
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
								{#if isPullRequest && filesUrl}
									<a
										class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 px-2.5 py-1.5 text-sm font-light text-gray-700 dark:text-gray-200 transition hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-800 dark:hover:text-gray-200"
										href={filesUrl}
										target="_blank"
										rel="noreferrer"
									>
										<!-- File diff icon -->
										<svg
											class="h-4 w-4"
											viewBox="0 0 16 16"
											fill="currentColor"
											xmlns="http://www.w3.org/2000/svg"
										>
											<!-- eslint-disable-next-line svelte/no-at-html-tags -->
											{@html fileDiffIconPath}
										</svg>
										<span>Files changed</span>
										<!-- External link icon -->
										<svg
											class="h-4 w-4"
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
								{/if}
							{/if}
						</div>
					{/if}

					<!-- For CI Activity, show a simple info message -->
					{#if isCIActivity}
						<div
							class="flex items-center gap-3 px-4 py-3 rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-800/30"
						>
							<svg
								class="w-5 h-5 flex-shrink-0 text-gray-500 dark:text-gray-400"
								viewBox="0 0 16 16"
								fill="currentColor"
								aria-hidden="true"
							>
								<!-- eslint-disable-next-line svelte/no-at-html-tags -->
								{@html getIconPath("play")}
							</svg>
							<p class="text-sm text-gray-600 dark:text-gray-400">
								This is a CI/CD workflow notification. View the full details on GitHub Actions.
							</p>
						</div>
					{:else}
						<!-- GitHub-style Comment Header + Content -->
						<div class="flex gap-3">
							<!-- Avatar and thread line -->
							<div
								class="flex flex-col items-center flex-shrink-0 relative z-10"
								style="width: 40px;"
							>
								<!-- Avatar / Initial -->
								<div class="flex-shrink-0">
									{#key notification.id}
										{#if authorAvatarUrl && !avatarLoadFailed}
											<img
												src={authorAvatarUrl}
												alt={`${authorName}'s avatar`}
												class="h-10 w-10 rounded-full bg-gray-300 dark:bg-gray-700 ring-2 ring-white dark:ring-gray-950"
												on:error={handleAvatarError}
											/>
										{:else}
											<div
												class="h-10 w-10 rounded-full bg-gray-300 dark:bg-gray-700 flex items-center justify-center ring-2 ring-white dark:ring-gray-950"
											>
												<span class="text-sm font-bold text-gray-800 dark:text-gray-200">
													{authorInitial}
												</span>
											</div>
										{/if}
									{/key}
								</div>

								<!-- Thread line extending down - only show if notification type supports timeline AND timeline has been loaded -->
								{#if timelineController && notification.githubId && typeConfig?.showCommentThread && hasLoadedTimeline}
									<div class="flex-1 w-0.5 bg-gray-300 dark:bg-gray-800 min-h-4 -z-10"></div>
								{/if}
							</div>

							<!-- Comment box -->
							<div class="flex-1 min-w-0">
								<div class="border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden">
									<!-- Comment Header -->
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
														class="px-1.5 text-xs font-medium border border-gray-300 dark:border-gray-700 rounded-md text-gray-600 dark:text-gray-400"
													>
														Bot
													</span>
												{/if}
												<span class="font-medium text-gray-600 dark:text-gray-500">
													{typeConfig?.timestampVerb || "updated"}
													{formattedTimestamp}
												</span>
											</div>
										</div>

										<!-- Status Indicators -->
										<div class="flex items-center gap-2">
											{#if isRefreshing}
												<div class="group relative flex-shrink-0">
													<svg
														class="h-4 w-4 text-blue-400/60 animate-spin-slow"
														viewBox="0 0 24 24"
														fill="none"
														xmlns="http://www.w3.org/2000/svg"
														aria-label="Refreshing"
													>
														<circle
															class="opacity-25"
															cx="12"
															cy="12"
															r="10"
															stroke="currentColor"
															stroke-width="3"
														/>
														<path
															class="opacity-75"
															fill="currentColor"
															d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
														/>
													</svg>
													<!-- Tooltip -->
													<div
														class="pointer-events-none absolute right-0 top-full z-50 mt-2 hidden w-48 rounded-lg border border-blue-700/50 bg-blue-900/95 px-3 py-2 text-xs text-blue-200 shadow-lg backdrop-blur-sm group-hover:block"
													>
														Refreshing from GitHub...
													</div>
												</div>
											{:else if showingStaleData}
												<div class="group relative flex-shrink-0">
													<svg
														class="h-5 w-5 text-amber-400"
														viewBox="0 0 24 24"
														fill="none"
														xmlns="http://www.w3.org/2000/svg"
														aria-label="Stale data warning"
													>
														<path
															d="M12 9V13M12 17H12.01M10.29 3.86L1.82 18C1.64537 18.3024 1.55299 18.6453 1.55201 18.9945C1.55103 19.3437 1.64149 19.6871 1.81442 19.9905C1.98735 20.2939 2.23672 20.5467 2.53771 20.7239C2.83869 20.901 3.18072 20.9962 3.53 21H20.47C20.8193 20.9962 21.1613 20.901 21.4623 20.7239C21.7633 20.5467 22.0127 20.2939 22.1856 19.9905C22.3585 19.6871 22.449 19.3437 22.448 18.9945C22.447 18.6453 22.3546 18.3024 22.18 18L13.71 3.86C13.5317 3.56611 13.2807 3.32312 12.9812 3.15448C12.6817 2.98585 12.3437 2.89725 12 2.89725C11.6563 2.89725 11.3183 2.98585 11.0188 3.15448C10.7193 3.32312 10.4683 3.56611 10.29 3.86Z"
															stroke="currentColor"
															stroke-width="2"
															stroke-linecap="round"
															stroke-linejoin="round"
														/>
													</svg>
													<!-- Tooltip -->
													<div
														class="pointer-events-none absolute right-0 top-full z-50 mt-2 hidden w-64 rounded-lg border border-amber-700/50 bg-amber-900/95 px-3 py-2 text-sm text-amber-200 shadow-lg backdrop-blur-sm group-hover:block"
													>
														<p>
															<span class="font-semibold">Unable to refresh data.</span>
															<span class="text-amber-300/90"
																>Showing cached information from last sync.</span
															>
														</p>
													</div>
												</div>
											{/if}
										</div>
									</div>

									<!-- Content section -->
									<div
										class={`p-4 leading-relaxed text-gray-900 dark:text-gray-200 min-h-[120px] ${showLoadingSkeleton ? "shimmer" : ""}`}
									>
										{#if showLoadingSkeleton}
											<!-- Empty during loading -->
										{:else if !detail}
											<!-- Waiting for detail to load -->
											<p class="text-sm text-gray-600 dark:text-gray-400 italic">Loading...</p>
										{:else if detail.subject?.body && hasVisibleContent(detail.subject.body)}
											<div
												class="prose dark:prose-invert prose-sm max-w-none prose-p:text-gray-700 dark:prose-p:text-gray-300 prose-headings:text-gray-900 dark:prose-headings:text-gray-300 prose-headings:font-semibold prose-a:text-indigo-600 dark:prose-a:text-indigo-300 prose-a:no-underline hover:prose-a:text-indigo-700 dark:hover:prose-a:text-indigo-200 hover:prose-a:underline prose-strong:text-gray-900 dark:prose-strong:text-gray-300 prose-strong:font-semibold prose-code:text-gray-800 dark:prose-code:text-gray-300 prose-code:bg-gray-100 dark:prose-code:bg-gray-900 prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:before:content-[''] prose-code:after:content-[''] prose-pre:text-gray-800 dark:prose-pre:text-gray-300 prose-pre:bg-gray-100 dark:prose-pre:bg-[#161b22] prose-blockquote:text-gray-600 dark:prose-blockquote:text-gray-400 prose-ul:text-gray-700 dark:prose-ul:text-gray-300 prose-ol:text-gray-700 dark:prose-ol:text-gray-300 prose-li:text-gray-700 dark:prose-li:text-gray-300"
												in:fade={{ duration: 200 }}
											>
												<!-- eslint-disable-next-line svelte/no-at-html-tags -->
												{@html renderMarkdown(detail.subject.body)}
											</div>
										{:else}
											<p
												class="text-sm text-gray-600 dark:text-gray-400 italic"
												in:fade={{ duration: 200 }}
											>
												{typeConfig?.emptyContentMessage || "No description provided."}
											</p>
										{/if}
									</div>
								</div>
							</div>
						</div>
					{/if}

					<!-- Timeline Thread - only show for types that support timeline -->
					{#if timelineController && notification.githubId && typeConfig?.showCommentThread}
						{#key notification.githubId}
							<TimelineThread githubId={notification.githubId} {timelineController} />
						{/key}
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	@keyframes shimmer {
		0% {
			background-position: -1000px 0;
		}
		100% {
			background-position: 1000px 0;
		}
	}

	.shimmer {
		animation: shimmer 2s infinite linear;
		background: linear-gradient(
			to right,
			rgb(229 231 235 / 0.4) 0%,
			rgb(209 213 219 / 0.6) 50%,
			rgb(229 231 235 / 0.4) 100%
		);
		background-size: 1000px 100%;
	}

	.dark .shimmer {
		background: linear-gradient(
			to right,
			rgb(30 41 59 / 0.3) 0%,
			rgb(51 65 85 / 0.5) 50%,
			rgb(30 41 59 / 0.3) 100%
		);
	}

	@keyframes spin-slow {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	.animate-spin-slow {
		animation: spin-slow 1.2s linear infinite;
	}
</style>
