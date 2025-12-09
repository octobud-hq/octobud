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

	import { getContext, tick } from "svelte";
	import type { Notification } from "$lib/api/types";
	import { getNotificationIcon, getStatusBarColor } from "$lib/utils/notificationIcons";
	import {
		formatRelative,
		normalizeReason,
		mailOpenIcon,
		mailClosedIcon,
	} from "$lib/utils/notificationHelpers";
	import { getArchiveIcon } from "$lib/utils/archiveIcons";
	import { getMuteIcon } from "$lib/utils/muteIcons";
	import SnoozeDropdown from "$lib/components/shared/SnoozeDropdown.svelte";
	import TagDropdown from "$lib/components/shared/TagDropdown.svelte";
	import type { NotificationPageController } from "$lib/state/types";
	import { formatSnoozeMessage } from "$lib/utils/snoozeFormat";
	import { currentTime } from "$lib/stores/timeStore";
	import type { SnoozeDropdownComponent, TagDropdownComponent } from "$lib/types/components";

	// Subscribe to time store to trigger re-renders when time updates
	// This ensures relative timestamps for same-day notifications stay fresh
	// Compute formatted timestamp reactively - depends on both notification date and current time
	$: formattedTimestamp = $currentTime && formatRelative(notification.updatedAt);

	// Get page controller from context
	const pageController = getContext<NotificationPageController>("notificationPageController");

	// Extract stores for reactivity
	const {
		pageData,
		splitModeEnabled,
		pendingFocusIndex,
		keyboardFocusIndex,
		savedListScrollPosition,
	} = pageController.stores;

	export let notification: Notification;
	export let selectionEnabled = false;
	export let selectionDisabled = false;
	export let selected = false;
	export let isDetailOpen = false; // Whether this notification is currently open in detail view

	let rootElement: HTMLDivElement | null = null;
	let snoozeButtonElement: HTMLButtonElement | null = null;
	let snoozeDropdownComponent: SnoozeDropdownComponent | null = null;
	let tagButtonElement: HTMLButtonElement | null = null;
	let tagDropdownComponent: TagDropdownComponent | null = null;

	/**
	 * Focus with minimal scroll - only scrolls if element is not fully visible,
	 * and only scrolls the minimum amount needed.
	 */
	export function focus() {
		if (!rootElement) return;
		scrollIntoViewMinimal(rootElement);
		rootElement.focus({ preventScroll: true });
	}

	function scrollIntoViewMinimal(element: HTMLElement) {
		const scrollParent = findScrollParent(element);
		if (!scrollParent) return;

		const elementRect = element.getBoundingClientRect();
		const parentRect = scrollParent.getBoundingClientRect();

		const overflowTop = parentRect.top - elementRect.top;
		const overflowBottom = elementRect.bottom - parentRect.bottom;

		if (overflowTop > 0) {
			scrollParent.scrollTop -= overflowTop;
		} else if (overflowBottom > 0) {
			scrollParent.scrollTop += overflowBottom;
		}
		// If fully visible, no scroll
	}

	function findScrollParent(element: HTMLElement): HTMLElement | null {
		let current = element.parentElement;
		while (current) {
			const overflowY = window.getComputedStyle(current).overflowY;
			if (overflowY === "auto" || overflowY === "scroll") {
				return current;
			}
			current = current.parentElement;
		}
		return null;
	}

	export function blur() {
		if (!rootElement) return;
		rootElement.blur();
	}

	/**
	 * Handle mousedown to focus without scroll.
	 * Prevents browser's default focus behavior and focuses with preventScroll.
	 */
	function handleMouseDown(event: MouseEvent) {
		if (event.button !== 0) return; // Only left click
		event.preventDefault();
		rootElement?.focus({ preventScroll: true });
	}

	$: repoLabel = notification.repoFullName || "Unknown repository";
	$: iconConfig = getNotificationIcon(
		notification.subjectType,
		notification.subjectRaw,
		notification.isRead,
		notification.subjectState,
		notification.subjectMerged
	);
	$: statusBarColor = getStatusBarColor(notification.isRead);
	$: reasonLabel = normalizeReason(notification.reason);

	// Compute this row's index in the list
	$: myIndex = $pageData.items.findIndex(
		(item) =>
			(item.githubId && item.githubId === notification.githubId) || item.id === notification.id
	);

	// Visual focus indicator (used for border highlight in SingleMode)
	$: hasKeyboardFocus = $keyboardFocusIndex === myIndex && myIndex !== -1;

	// When pendingFocusIndex matches this row, focus the element (for keyboard/button nav only)
	$: {
		if ($pendingFocusIndex === myIndex && myIndex !== -1 && rootElement) {
			void tick().then(() => {
				focus();
				pageController.actions.consumePendingFocus();
			});
		}
	}

	// Note: Prefer extracted fields (subjectMerged, subjectState, etc.) over parsing subjectRaw
	$: isUnread = !notification.isRead;
	$: markReadLabel = isUnread ? "Mark read" : "Mark unread";
	$: isArchived = notification.archived;
	$: archiveLabel = isArchived ? "Unarchive" : "Archive";
	$: isMuted = notification.muted;
	$: muteLabel = isMuted ? "Unmute" : "Mute";
	$: archiveIcon = getArchiveIcon(isArchived);
	$: muteIcon = getMuteIcon(isMuted);
	$: subjectLink =
		notification.htmlUrl || notification.subjectUrl || notification.githubUrl || null;
	$: openSubjectLabel = subjectLink
		? `Open ${iconConfig.label.toLowerCase()}`
		: "Subject link unavailable";
	$: mailIcon = isUnread ? mailClosedIcon : mailOpenIcon;
	// In light mode: unread = full white, read = grayed out
	// In dark mode: keep existing subtle differences
	$: baseBackground = isUnread
		? "bg-white dark:bg-gray-800/60"
		: "bg-gray-200/70 dark:bg-gray-900/40";
	$: hoverBackground = isUnread
		? "hover:bg-gray-50 dark:hover:bg-gray-800/40"
		: "hover:bg-gray-300 dark:hover:bg-gray-900/20";
	// Slightly muted title for read notifications in light mode
	$: titleColor = isUnread
		? "text-gray-900 dark:text-gray-100"
		: "text-gray-700 dark:text-gray-300";
	// Make title heavier (bolder) for unread messages in light mode
	$: titleFontWeight = isUnread ? "font-semibold dark:font-normal" : "font-normal";
	// Button background: lighter background for better contrast in light mode
	// Unread row is bg-white, so use bg-gray-50
	// Read row is bg-gray-100, so use bg-gray-50
	$: buttonGroupBg = isUnread ? "bg-gray-50 dark:bg-gray-950" : "bg-gray-50 dark:bg-gray-950";
	// Slightly darker border on unread notifications in light mode, lighter border for read
	$: borderClass = isUnread
		? "border-gray-300 dark:border-gray-800"
		: "border-gray-300 dark:border-gray-800";
	// Icon opacity - more muted for read rows
	$: iconOpacity = isUnread ? "opacity-90" : "opacity-60";
	// Reason tag color - more muted for read rows in light mode
	$: reasonTagClass = isUnread
		? "bg-blue-500/10 text-blue-600 dark:bg-blue-500/20 dark:text-blue-400"
		: "bg-blue-500/10 text-blue-500 dark:bg-blue-500/10 dark:text-blue-400/60";

	// Subscribe to the snooze dropdown store from page controller
	const snoozeDropdownState = pageController.stores.snoozeDropdownState;
	$: isSnoozeDropdownOpen =
		$snoozeDropdownState?.notificationId === (notification.githubId ?? notification.id) &&
		// Show for direct clicks on this row's button
		($snoozeDropdownState?.source === "notification-row" ||
			// For keyboard shortcuts, only show if NOT in split mode with detail open
			// (defer to DetailActionBar in that case)
			($snoozeDropdownState?.source === "keyboard-shortcut" &&
				!($splitModeEnabled && isDetailOpen)));

	// Pass button element to dropdown when it changes
	$: if (snoozeDropdownComponent && snoozeButtonElement) {
		snoozeDropdownComponent.setButtonElement(snoozeButtonElement);
	}

	// Subscribe to the tag dropdown store from page controller
	const tagDropdownState = pageController.stores.tagDropdownState;
	$: isTagDropdownOpen =
		$tagDropdownState?.notificationId === (notification.githubId ?? notification.id) &&
		// Show for direct clicks on this row's button
		($tagDropdownState?.source === "notification-row" ||
			// For keyboard shortcuts, only show if NOT in split mode with detail open
			// (defer to DetailActionBar in that case)
			($tagDropdownState?.source === "keyboard-shortcut" && !($splitModeEnabled && isDetailOpen)));

	// Pass button element to tag dropdown when it changes
	$: if (tagDropdownComponent && tagButtonElement) {
		tagDropdownComponent.setButtonElement(tagButtonElement);
	}

	const openDetails = () => {
		// In SingleMode, save scroll position before opening detail
		if (!$splitModeEnabled) {
			const scrollParent = rootElement ? findScrollParent(rootElement) : null;
			if (scrollParent) {
				savedListScrollPosition.set(scrollParent.scrollTop);
			}
		}
		void pageController.actions.handleOpenInlineDetail(notification);
	};

	const handleCardKeyDown = (event: KeyboardEvent) => {
		const currentTarget = event.currentTarget as HTMLElement | null;
		if (!currentTarget || event.target !== currentTarget) {
			return;
		}
		if (event.key === "Enter" || event.key === " " || event.key === "Spacebar") {
			event.preventDefault();
			openDetails();
		}
	};

	function handleSnoozeClick() {
		const notificationId = notification.githubId ?? notification.id;
		// Check if dropdown is open for this notification (any source)
		const isOpenForThisNotification = $snoozeDropdownState?.notificationId === notificationId;

		// Toggle: if already open, close it; otherwise open it
		if (isOpenForThisNotification) {
			pageController.actions.closeSnoozeDropdown();
		} else {
			pageController.actions.openSnoozeDropdown(notificationId, "notification-row");
		}
	}

	function handleSnoozeDropdownClose() {
		pageController.actions.closeSnoozeDropdown();
	}

	function handleSnoozeAction(until: string) {
		void pageController.actions.snooze(notification, until);
	}

	function handleUnsnoozeAction() {
		void pageController.actions.unsnooze(notification);
	}

	function handleMuteAction() {
		if (isMuted) {
			void pageController.actions.unmute(notification);
		} else {
			void pageController.actions.mute(notification);
		}
	}

	function handleOpenCustomDialog() {
		pageController.actions.openCustomSnoozeDialog(notification.githubId ?? notification.id);
	}

	function handleTagClick() {
		const notificationId = notification.githubId ?? notification.id;
		// Check if dropdown is open for this notification (any source)
		const isOpenForThisNotification = $tagDropdownState?.notificationId === notificationId;

		// Toggle: if already open, close it; otherwise open it
		if (isOpenForThisNotification) {
			pageController.actions.closeTagDropdown();
		} else {
			pageController.actions.openTagDropdown(notificationId, "notification-row");
		}
	}

	function handleTagDropdownClose() {
		pageController.actions.closeTagDropdown();
	}

	async function handleAssignTag(tagId: string) {
		const githubId = notification.githubId ?? notification.id;
		try {
			await pageController.actions.assignTag(githubId, tagId);
		} catch (error) {}
	}

	async function handleRemoveTag(tagId: string) {
		const githubId = notification.githubId ?? notification.id;
		try {
			await pageController.actions.removeTag(githubId, tagId);
		} catch (error) {}
	}

	$: snoozeMessage = notification.snoozedUntil
		? formatSnoozeMessage(notification.snoozedUntil, notification.snoozedAt)
		: null;
</script>

<div class="relative flex items-start gap-2 pl-2">
	<!-- Checkbox (outside card, to the left of everything) -->
	{#if selectionEnabled}
		<div class="flex items-center self-center">
			<label class="relative flex cursor-pointer items-center">
				<input
					type="checkbox"
					class="peer sr-only"
					checked={selected}
					disabled={selectionDisabled}
					on:click|stopPropagation
					on:change|stopPropagation={() => pageController.actions.toggleSelection(notification)}
					aria-label={`Select notification ${notification.subjectTitle}`}
				/>
				<div
					class="h-5 w-5 rounded border transition-all peer-disabled:cursor-not-allowed peer-disabled:opacity-50 {selected
						? 'border-indigo-500 bg-indigo-500'
						: 'border-gray-400 bg-transparent hover:border-gray-500 dark:border-gray-600 dark:hover:border-gray-500'}"
					title={selectionDisabled
						? "Individual selection is disabled when selecting all across pages"
						: ""}
				>
					{#if selected}
						<svg
							class="h-full w-full text-white p-0.5"
							viewBox="0 0 16 16"
							fill="none"
							xmlns="http://www.w3.org/2000/svg"
						>
							<path
								d="M13 4L6 11L3 8"
								stroke="currentColor"
								stroke-width="2.5"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
					{/if}
				</div>
			</label>
		</div>
	{/if}

	<!-- Vertical Status Bar (outside card) -->
	<div class="w-1 h-12 mt-2 rounded-full {statusBarColor}" aria-hidden="true"></div>

	<div
		bind:this={rootElement}
		data-notification-key={notification.githubId ?? notification.id}
		class={`group relative flex flex-1 cursor-pointer gap-2.5 rounded-lg border px-2.5 py-2 transition ${baseBackground} ${hoverBackground} focus-visible:outline-none ${hasKeyboardFocus || isDetailOpen ? `border-transparent ring-1 ring-blue-600` : borderClass}`}
		role="button"
		tabindex="0"
		aria-label={`Open details for ${notification.subjectTitle}`}
		on:mousedown={handleMouseDown}
		on:click={openDetails}
		on:keydown={handleCardKeyDown}
	>
		<div class="flex-1 text-left space-y-1.5">
			<!-- Top Row: Icon + Title -->
			<div class="flex items-start gap-2">
				<div class="flex-shrink-0 mt-0.5">
					<svg
						class="w-4 h-4 {iconConfig.colorClass} {iconOpacity}"
						viewBox="0 0 16 16"
						fill="currentColor"
						aria-hidden="true"
					>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html iconConfig.path}
					</svg>
				</div>
				<div class="flex items-center gap-1.5 flex-1 flex-wrap">
					<!-- Status badges (muted takes priority over archived, filtered has lowest precedence) -->
					{#if isMuted}
						<span
							class="rounded-md bg-violet-500/10 dark:bg-violet-500/20 px-1.5 py-0.5 text-[10px] font-medium text-violet-600 dark:text-violet-300"
						>
							Muted
						</span>
					{:else if isArchived}
						<span
							class="rounded-md bg-blue-500/10 dark:bg-blue-500/20 px-1.5 py-0.5 text-[10px] font-medium text-blue-600 dark:text-blue-300"
						>
							Archived
						</span>
					{:else if notification.filtered}
						<span
							class="rounded-md bg-gray-500/10 dark:bg-gray-500/20 px-1.5 py-0.5 text-[10px] font-medium text-gray-700 dark:text-gray-300"
						>
							Skipped Inbox
						</span>
					{/if}
					<span class={`text-sm ${titleFontWeight} leading-snug ${titleColor} transition`}>
						{notification.subjectTitle}
					</span>
				</div>
			</div>

			<!-- Bottom Row: Repo + Metadata -->
			<div class="flex flex-wrap items-center gap-1.5 pl-6">
				<span
					class={`text-sm ${isUnread ? "font-medium dark:font-normal text-gray-600" : "font-normal text-gray-500"} dark:text-gray-400`}
					>{repoLabel}</span
				>
				{#if notification.subjectNumber}
					<span
						class={`rounded-md px-1.5 py-0.5 text-[11px] font-medium ${isUnread ? "bg-gray-200/80 text-gray-700 dark:bg-gray-800/80 dark:text-gray-400" : "bg-gray-300/60 text-gray-700 dark:bg-gray-800/50 dark:text-gray-500"}`}
					>
						#{notification.subjectNumber}
					</span>
				{/if}
				{#if notification?.authorLogin}
					<span class="flex items-center gap-1 text-[11px] text-gray-600 dark:text-gray-600">
						<span>by</span>
						<span class="text-gray-600 dark:text-gray-500">{notification.authorLogin}</span>
					</span>
				{/if}
				<span class={`rounded-md px-1.5 py-0.5 text-[11px] font-medium ${reasonTagClass}`}>
					{reasonLabel}
				</span>
				<span class="text-[11px] font-medium text-gray-500 dark:text-gray-500">
					{formattedTimestamp}
				</span>
				{#if notification.tags && notification.tags.length > 0}
					<div class="flex items-center gap-1">
						{#each notification.tags.slice(0, 3) as tag (tag.id)}
							<span
								class="rounded-full border px-1.5 py-0.5 text-[10px] font-semibold {isUnread
									? ''
									: 'dark:opacity-60'}"
								style="background-color: {tag.color}20; color: {tag.color}; border-color: {tag.color}"
								title={tag.name}
							>
								{tag.name}
							</span>
						{/each}
						{#if notification.tags.length > 3}
							<span
								class="text-[10px] text-gray-500 dark:text-gray-500"
								title="{notification.tags.length} tags total"
							>
								+{notification.tags.length - 3}
							</span>
						{/if}
					</div>
				{/if}
			</div>
		</div>

		<div class="flex flex-col items-end justify-start self-stretch">
			<!-- Status indicators (top right) - Snooze message, starred icon, muted icon -->
			{#if snoozeMessage || notification.starred || isMuted}
				<div
					class="flex items-center gap-2 transition-opacity {isSnoozeDropdownOpen
						? 'opacity-0'
						: 'group-hover:opacity-0'}"
				>
					{#if snoozeMessage}
						<span
							class="text-xs font-medium text-orange-600 dark:text-orange-400"
							title={`Snoozed until ${new Date(notification.snoozedUntil).toLocaleString()}`}
						>
							{snoozeMessage}
						</span>
					{/if}
					{#if notification.starred}
						<span title="Starred">
							<svg
								class="h-[17px] w-[17px] text-violet-500"
								viewBox="0 0 24 24"
								fill="currentColor"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<path
									d="M11.245 4.174C11.4765 3.50808 11.5922 3.17513 11.7634 3.08285C11.9115 3.00298 12.0898 3.00298 12.238 3.08285C12.4091 3.17513 12.5248 3.50808 12.7563 4.174L14.2866 8.57639C14.3525 8.76592 14.3854 8.86068 14.4448 8.93125C14.4972 8.99359 14.5641 9.04218 14.6396 9.07278C14.725 9.10743 14.8253 9.10947 15.0259 9.11356L19.6857 9.20852C20.3906 9.22288 20.743 9.23007 20.8837 9.36432C21.0054 9.48051 21.0605 9.65014 21.0303 9.81569C20.9955 10.007 20.7146 10.2199 20.1528 10.6459L16.4387 13.4616C16.2788 13.5829 16.1989 13.6435 16.1501 13.7217C16.107 13.7909 16.0815 13.8695 16.0757 13.9507C16.0692 14.0427 16.0982 14.1387 16.1563 14.3308L17.506 18.7919C17.7101 19.4667 17.8122 19.8041 17.728 19.9793C17.6551 20.131 17.5108 20.2358 17.344 20.2583C17.1513 20.2842 16.862 20.0829 16.2833 19.6802L12.4576 17.0181C12.2929 16.9035 12.2106 16.8462 12.1211 16.8239C12.042 16.8043 11.9593 16.8043 11.8803 16.8239C11.7908 16.8462 11.7084 16.9035 11.5437 17.0181L7.71805 19.6802C7.13937 20.0829 6.85003 20.2842 6.65733 20.2583C6.49056 20.2358 6.34626 20.131 6.27337 19.9793C6.18915 19.8041 6.29123 19.4667 6.49538 18.7919L7.84503 14.3308C7.90313 14.1387 7.93218 14.0427 7.92564 13.9507C7.91986 13.8695 7.89432 13.7909 7.85123 13.7217C7.80246 13.6435 7.72251 13.5829 7.56262 13.4616L3.84858 10.6459C3.28678 10.2199 3.00588 10.007 2.97101 9.81569C2.94082 9.65014 2.99594 9.48051 3.11767 9.36432C3.25831 9.23007 3.61074 9.22289 4.31559 9.20852L8.9754 9.11356C9.176 9.10947 9.27631 9.10743 9.36177 9.07278C9.43726 9.04218 9.50414 8.99359 9.55657 8.93125C9.61593 8.86068 9.64887 8.76592 9.71475 8.57639L11.245 4.174Z"
								/>
							</svg>
						</span>
					{/if}
					{#if isMuted}
						<span title="Muted">
							<svg
								class="h-4 w-4 text-violet-400"
								viewBox="0 0 24 24"
								fill={muteIcon.useStroke ? "none" : "currentColor"}
								stroke={muteIcon.useStroke ? "currentColor" : "none"}
								stroke-width={muteIcon.useStroke ? "2" : undefined}
								stroke-linecap={muteIcon.useStroke ? "round" : undefined}
								stroke-linejoin={muteIcon.useStroke ? "round" : undefined}
								aria-hidden="true"
							>
								<path d={muteIcon.path} />
							</svg>
						</span>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Action Buttons (bottom right, absolutely positioned) -->
		<footer
			class="absolute right-1 flex items-center gap-1 transition-opacity {isSnoozeDropdownOpen ||
			isTagDropdownOpen
				? 'opacity-100'
				: 'opacity-0 group-hover:opacity-100'}"
		>
			<div class="{buttonGroupBg} rounded-full p-0.5 flex items-center gap-0.5 shadow-lg">
				{#if subjectLink}
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a
						class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600 cursor-pointer"
						href={subjectLink}
						target="_blank"
						rel="noopener noreferrer"
						title={openSubjectLabel}
						aria-label={openSubjectLabel}
						on:click|stopPropagation
					>
						<svg
							class="h-5 w-5"
							viewBox="0 0 24 24"
							fill="none"
							xmlns="http://www.w3.org/2000/svg"
							role="img"
							aria-hidden="true"
						>
							<path
								d="M10.0002 5H8.2002C7.08009 5 6.51962 5 6.0918 5.21799C5.71547 5.40973 5.40973 5.71547 5.21799 6.0918C5 6.51962 5 7.08009 5 8.2002V15.8002C5 16.9203 5 17.4801 5.21799 17.9079C5.40973 18.2842 5.71547 18.5905 6.0918 18.7822C6.5192 19 7.07899 19 8.19691 19H15.8031C16.921 19 17.48 19 17.9074 18.7822C18.2837 18.5905 18.5905 18.2839 18.7822 17.9076C19 17.4802 19 16.921 19 15.8031V14M20 9V4M20 4H15M20 4L13 11"
								stroke="currentColor"
								stroke-width="1.5"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
					</a>
				{/if}

				<button
					type="button"
					class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer"
					on:click|stopPropagation={handleMuteAction}
					aria-label={muteLabel}
					title={muteLabel}
				>
					<svg
						class="h-5 w-5"
						viewBox="0 0 24 24"
						fill={muteIcon.useStroke ? "none" : "currentColor"}
						stroke={muteIcon.useStroke ? "currentColor" : "none"}
						stroke-width={muteIcon.useStroke ? "2" : undefined}
						stroke-linecap={muteIcon.useStroke ? "round" : undefined}
						stroke-linejoin={muteIcon.useStroke ? "round" : undefined}
						aria-hidden="true"
					>
						<path d={muteIcon.path} />
					</svg>
				</button>
				<button
					type="button"
					class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer"
					on:click|stopPropagation={() => void pageController.actions.archive(notification)}
					aria-label={archiveLabel}
					title={archiveLabel}
				>
					<svg
						class="h-5 w-5"
						viewBox="0 0 24 24"
						fill={archiveIcon.useStroke ? "none" : "currentColor"}
						stroke={archiveIcon.useStroke ? "currentColor" : "none"}
						stroke-width={archiveIcon.useStroke ? "1.5" : undefined}
						stroke-linecap={archiveIcon.useStroke ? "round" : undefined}
						stroke-linejoin={archiveIcon.useStroke ? "round" : undefined}
						aria-hidden="true"
					>
						{#each archiveIcon.paths as path (path)}
							<path d={path} />
						{/each}
					</svg>
				</button>
				{#if notification.filtered}
					<button
						type="button"
						class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer"
						on:click|stopPropagation={() => void pageController.actions.unfilter(notification)}
						aria-label="Move to inbox"
						title="Move to inbox"
					>
						<svg
							class="h-5 w-5"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
						>
							<path
								d="M3 12V15.8C3 16.9201 3 17.4802 3.21799 17.908C3.40973 18.2843 3.71569 18.5903 4.09202 18.782C4.51984 19 5.0799 19 6.2 19H17.8C18.9201 19 19.4802 19 19.908 18.782C20.2843 18.5903 20.5903 18.2843 20.782 17.908C21 17.4802 21 16.9201 21 15.8V12M3 12H6.67452C7.16369 12 7.40829 12 7.63846 12.0553C7.84254 12.1043 8.03763 12.1851 8.21657 12.2947C8.4184 12.4184 8.59136 12.5914 8.93726 12.9373L9.06274 13.0627C9.40865 13.4086 9.5816 13.5816 9.78343 13.7053C9.96237 13.8149 10.1575 13.8957 10.3615 13.9447C10.5917 14 10.8363 14 11.3255 14H12.6745C13.1637 14 13.4083 14 13.6385 13.9447C13.8425 13.8957 14.0376 13.8149 14.2166 13.7053C14.4184 13.5816 14.5914 13.4086 14.9373 13.0627L15.0627 12.9373C15.4086 12.5914 15.5816 12.4184 15.7834 12.2947C15.9624 12.1851 16.1575 12.1043 16.3615 12.0553C16.5917 12 16.8363 12 17.3255 12H21M3 12L5.32639 6.83025C5.78752 5.8055 6.0181 5.29312 6.38026 4.91755C6.70041 4.58556 7.09278 4.33186 7.52691 4.17615C8.01802 4 8.57988 4 9.70361 4H14.2964C15.4201 4 15.982 4 16.4731 4.17615C16.9072 4.33186 17.2996 4.58556 17.6197 4.91755C17.9819 5.29312 18.2125 5.8055 18.6736 6.83025L21 12"
							/>
						</svg>
					</button>
				{/if}
				<div class="relative">
					<button
						bind:this={tagButtonElement}
						type="button"
						class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer"
						on:click|stopPropagation={handleTagClick}
						aria-label="Manage tags"
						title="Manage tags"
					>
						{#if notification.tags && notification.tags.length > 0}
							<!-- Filled icon when tags are assigned -->
							<svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
								<path
									d="M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16z"
								/>
							</svg>
						{:else}
							<!-- Outline icon when no tags assigned -->
							<svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
								<path
									d="M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16zM16 17H5V7h11l3.55 5L16 17z"
								/>
							</svg>
						{/if}
					</button>
					<TagDropdown
						bind:this={tagDropdownComponent}
						{notification}
						isOpen={isTagDropdownOpen}
						onAssignTag={handleAssignTag}
						onRemoveTag={handleRemoveTag}
						onClose={handleTagDropdownClose}
					/>
				</div>
				<button
					type="button"
					class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer {notification.starred
						? 'text-violet-500'
						: 'text-gray-700 dark:text-gray-300'}"
					on:click|stopPropagation={() =>
						notification.starred
							? void pageController.actions.unstar(notification)
							: void pageController.actions.star(notification)}
					aria-label={notification.starred ? "Unstar" : "Star"}
					title={notification.starred ? "Unstar" : "Star"}
				>
					<svg
						class="h-5 w-5"
						viewBox="0 0 24 24"
						fill={notification.starred ? "currentColor" : "none"}
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path
							d="M11.245 4.174C11.4765 3.50808 11.5922 3.17513 11.7634 3.08285C11.9115 3.00298 12.0898 3.00298 12.238 3.08285C12.4091 3.17513 12.5248 3.50808 12.7563 4.174L14.2866 8.57639C14.3525 8.76592 14.3854 8.86068 14.4448 8.93125C14.4972 8.99359 14.5641 9.04218 14.6396 9.07278C14.725 9.10743 14.8253 9.10947 15.0259 9.11356L19.6857 9.20852C20.3906 9.22288 20.743 9.23007 20.8837 9.36432C21.0054 9.48051 21.0605 9.65014 21.0303 9.81569C20.9955 10.007 20.7146 10.2199 20.1528 10.6459L16.4387 13.4616C16.2788 13.5829 16.1989 13.6435 16.1501 13.7217C16.107 13.7909 16.0815 13.8695 16.0757 13.9507C16.0692 14.0427 16.0982 14.1387 16.1563 14.3308L17.506 18.7919C17.7101 19.4667 17.8122 19.8041 17.728 19.9793C17.6551 20.131 17.5108 20.2358 17.344 20.2583C17.1513 20.2842 16.862 20.0829 16.2833 19.6802L12.4576 17.0181C12.2929 16.9035 12.2106 16.8462 12.1211 16.8239C12.042 16.8043 11.9593 16.8043 11.8803 16.8239C11.7908 16.8462 11.7084 16.9035 11.5437 17.0181L7.71805 19.6802C7.13937 20.0829 6.85003 20.2842 6.65733 20.2583C6.49056 20.2358 6.34626 20.131 6.27337 19.9793C6.18915 19.8041 6.29123 19.4667 6.49538 18.7919L7.84503 14.3308C7.90313 14.1387 7.93218 14.0427 7.92564 13.9507C7.91986 13.8695 7.89432 13.7909 7.85123 13.7217C7.80246 13.6435 7.72251 13.5829 7.56262 13.4616L3.84858 10.6459C3.28678 10.2199 3.00588 10.007 2.97101 9.81569C2.94082 9.65014 2.99594 9.48051 3.11767 9.36432C3.25831 9.23007 3.61074 9.22289 4.31559 9.20852L8.9754 9.11356C9.176 9.10947 9.27631 9.10743 9.36177 9.07278C9.43726 9.04218 9.50414 8.99359 9.55657 8.93125C9.61593 8.86068 9.64887 8.76592 9.71475 8.57639L11.245 4.174Z"
						/>
					</svg>
				</button>
				<button
					type="button"
					class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer"
					on:click|stopPropagation={() => void pageController.actions.markRead(notification)}
					aria-label={markReadLabel}
					title={markReadLabel}
				>
					<svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
						<path
							d={mailIcon.path}
							fill-rule={mailIcon.fillRule}
							clip-rule={mailIcon.clipRule}
							stroke="currentColor"
							stroke-width=".1"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
				</button>
				<div class="relative">
					<button
						bind:this={snoozeButtonElement}
						type="button"
						class="flex h-10 w-10 items-center justify-center rounded-full bg-transparent text-gray-700 dark:text-gray-300 transition hover:bg-gray-200 dark:hover:bg-gray-700/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue-600/40 cursor-pointer"
						on:click|stopPropagation={handleSnoozeClick}
						aria-label="Snooze"
						title="Snooze"
					>
						<svg
							class="h-5 w-5"
							viewBox="0 0 24 24"
							fill="none"
							xmlns="http://www.w3.org/2000/svg"
							role="img"
							aria-hidden="true"
						>
							<path
								d="M12 6V12L16 14M22 12C22 17.5228 17.5228 22 12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12Z"
								stroke="currentColor"
								stroke-width="1.8"
								stroke-linecap="round"
								stroke-linejoin="round"
							/>
						</svg>
					</button>
					<SnoozeDropdown
						bind:this={snoozeDropdownComponent}
						{notification}
						isOpen={isSnoozeDropdownOpen}
						onSnooze={handleSnoozeAction}
						onUnsnooze={handleUnsnoozeAction}
						onOpenCustomDialog={handleOpenCustomDialog}
						onClose={handleSnoozeDropdownClose}
					/>
				</div>
			</div>
		</footer>
	</div>
</div>
