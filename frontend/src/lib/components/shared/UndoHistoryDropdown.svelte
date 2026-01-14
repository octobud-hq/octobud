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

	import { onDestroy, tick } from "svelte";
	import { SvelteSet } from "svelte/reactivity";
	import { fly } from "svelte/transition";
	import { quintOut } from "svelte/easing";
	import { browser } from "$app/environment";
	import { undoStore } from "$lib/stores/undoStore";
	import type { UndoableAction, RecentActionNotification } from "$lib/undo/types";
	import { getActionIconConfig, getActionLabel } from "$lib/undo/actionIcons";
	import { getNotificationIcon } from "$lib/utils/notificationIcons";

	export let open = false;
	export let onClose: () => void = () => {};
	export let onNavigateToNotification: ((notification: RecentActionNotification) => void) | null =
		null;
	export let triggerElement: HTMLElement | null = null;

	let dropdownElement: HTMLDivElement | null = null;

	// Keyboard navigation state
	// -1 means no item focused (mouse mode), 0+ means keyboard navigation active
	let focusedIndex = -1;
	let actionItemElements: HTMLElement[] = [];

	// Exported methods for external control (called via commands from listShortcuts)
	export function navigateDown(): boolean {
		if (!open || allActions.length === 0) return false;
		if (focusedIndex === -1) {
			focusedIndex = 0;
		} else {
			focusedIndex = Math.min(focusedIndex + 1, allActions.length - 1);
		}
		scrollToFocused();
		return true;
	}

	export function navigateUp(): boolean {
		if (!open || allActions.length === 0) return false;
		if (focusedIndex === -1) {
			focusedIndex = allActions.length - 1;
		} else {
			focusedIndex = Math.max(focusedIndex - 1, 0);
		}
		scrollToFocused();
		return true;
	}

	export function undoFocused(): boolean {
		if (!open || allActions.length === 0 || focusedIndex < 0) return false;
		const action = allActions[focusedIndex];
		if (action) {
			void handleUndo(action);
			return true;
		}
		return false;
	}

	export function openFocusedNotification(): boolean {
		if (!open || allActions.length === 0 || focusedIndex < 0) return false;
		const action = allActions[focusedIndex];
		if (!action) return false;

		// For single notification actions, open the notification
		if (action.notifications.length === 1 && onNavigateToNotification) {
			onNavigateToNotification(action.notifications[0]);
			onClose();
			return true;
		}

		// For bulk actions, toggle the expanded state
		if (action.notifications.length > 1) {
			toggleExpanded(action.id);
			return true;
		}

		return false;
	}

	function handleClickOutside(event: MouseEvent) {
		if (!open || !dropdownElement) return;

		const target = event.target as Node;
		// Check if click is outside the dropdown AND outside the trigger button
		const isOutsideDropdown = !dropdownElement.contains(target);
		const isOutsideTrigger = !triggerElement || !triggerElement.contains(target);

		if (isOutsideDropdown && isOutsideTrigger) {
			onClose();
		}
	}

	function scrollToFocused() {
		tick().then(() => {
			const element = actionItemElements[focusedIndex];
			if (element) {
				element.scrollIntoView({ block: "nearest", behavior: "smooth" });
			}
		});
	}

	// Reset state when dropdown opens/closes
	$: if (open) {
		// Always start with no item focused - user must press arrow keys to navigate
		focusedIndex = -1;
		// Focus the dropdown for keyboard events
		tick().then(() => {
			dropdownElement?.focus();
		});
	} else {
		// Clean up expanded states when dropdown closes
		expandedActionIds = new SvelteSet();
	}

	// Reset action item element references when actions change
	// Required for bind:this to properly populate after keyed {#each} re-renders
	$: if (allActions) {
		actionItemElements = [];
	}

	// Unified reactive block to manage click-outside listener
	// Always remove first, then add if open - prevents race conditions
	$: {
		if (browser) {
			document.removeEventListener("click", handleClickOutside, true);
			if (open) {
				// Small delay to avoid closing immediately on the same click that opened it
				setTimeout(() => {
					document.addEventListener("click", handleClickOutside, true);
				}, 0);
			}
		}
	}

	onDestroy(() => {
		if (browser) {
			document.removeEventListener("click", handleClickOutside, true);
		}
	});

	$: activeToastAction = $undoStore.activeToastAction;
	$: isUndoing = $undoStore.isUndoing;

	// Actions are now stored in history immediately (not just when toast expires),
	// so we just display the history directly. The activeToastAction is a reference
	// to the item whose toast is currently visible, used for keyboard undo.
	$: allActions = $undoStore.history;

	function formatRelativeTime(timestamp: number): string {
		const seconds = Math.floor((Date.now() - timestamp) / 1000);

		if (seconds < 5) return "just now";
		if (seconds < 60) return `${seconds}s ago`;

		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;

		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;

		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}

	async function handleUndo(action: UndoableAction) {
		// Check if this is the active toast action or a history item
		if (activeToastAction && action.id === activeToastAction.id) {
			await undoStore.undoActiveToast();
		} else {
			await undoStore.undoFromHistory(action.id);
		}
		// Close dropdown after undo if nothing left
		if ($undoStore.history.length === 0 && $undoStore.activeToastAction === null) {
			onClose();
		}
	}

	// Expanded states for bulk actions
	let expandedActionIds = new SvelteSet<string>();

	function toggleExpanded(actionId: string) {
		if (expandedActionIds.has(actionId)) {
			expandedActionIds.delete(actionId);
		} else {
			expandedActionIds.add(actionId);
		}
	}

	function handleNotificationClick(notification: RecentActionNotification) {
		if (onNavigateToNotification) {
			onNavigateToNotification(notification);
			onClose();
		}
	}
</script>

{#if open}
	<!-- Dropdown -->
	<div
		bind:this={dropdownElement}
		in:fly={{ duration: 200, y: -10, easing: quintOut }}
		class="absolute right-0 top-full z-50 mt-2 w-[28rem] max-w-[calc(100vw-2rem)] overflow-hidden rounded-xl border border-gray-200 bg-gray-50 shadow-xl dark:border-gray-700 dark:bg-gray-950 outline-none"
		role="dialog"
		aria-label="Recent actions"
		tabindex="-1"
	>
		<!-- Header -->
		<div
			class="border-b border-gray-200 bg-gray-50 px-4 py-3 dark:border-gray-700 dark:bg-gray-950"
		>
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">Recent actions</h3>
				{#if allActions.length > 0}
					<div class="flex items-center gap-3 text-xs text-gray-400 dark:text-gray-500">
						<span class="flex items-center gap-1">
							<kbd
								class="rounded bg-gray-200 px-1.5 py-0.5 font-mono text-[10px] text-gray-600 dark:bg-gray-800 dark:text-gray-400"
								>J</kbd
							>
							<kbd
								class="rounded bg-gray-200 px-1.5 py-0.5 font-mono text-[10px] text-gray-600 dark:bg-gray-800 dark:text-gray-400"
								>K</kbd
							>
							<span class="ml-0.5">navigate</span>
						</span>
						<span class="flex items-center gap-1">
							<kbd
								class="rounded bg-gray-200 px-1.5 py-0.5 font-mono text-[10px] text-gray-600 dark:bg-gray-800 dark:text-gray-400"
								>Enter</kbd
							>
							<span class="ml-0.5">undo</span>
						</span>
						<span class="flex items-center gap-1">
							<kbd
								class="rounded bg-gray-200 px-1.5 py-0.5 font-mono text-[10px] text-gray-600 dark:bg-gray-800 dark:text-gray-400"
								>Space</kbd
							>
							<span class="ml-0.5">open</span>
						</span>
					</div>
				{/if}
			</div>
		</div>

		<!-- History list -->
		<div class="max-h-96 overflow-y-auto">
			{#if allActions.length === 0}
				<div class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
					No recent actions
				</div>
			{:else}
				<ul class="divide-y divide-gray-100 dark:divide-gray-800">
					{#each allActions as action, index (action.id)}
						{@const isBulk = action.notifications.length > 1}
						{@const isExpanded = expandedActionIds.has(action.id)}
						{@const iconConfig = getActionIconConfig(action.type)}
						{@const isFocused = focusedIndex >= 0 && index === focusedIndex}
						<li
							bind:this={actionItemElements[index]}
							class="px-3 py-3 transition-colors {isFocused
								? 'bg-gray-200 dark:bg-gray-800/50'
								: ''}"
						>
							<!-- Row 1: Action icon + type + timestamp + undo button -->
							<div class="flex items-center gap-2">
								<!-- Action type icon -->
								<svg
									class="h-4 w-4 flex-shrink-0 text-gray-600 dark:text-gray-400"
									viewBox={iconConfig.viewBox}
									fill={iconConfig.useStroke ? "none" : "currentColor"}
									stroke={iconConfig.useStroke ? "currentColor" : "none"}
									stroke-width={iconConfig.useStroke ? iconConfig.strokeWidth : undefined}
									stroke-linecap={iconConfig.useStroke ? "round" : undefined}
									stroke-linejoin={iconConfig.useStroke ? "round" : undefined}
									fill-rule={iconConfig.fillRule}
									clip-rule={iconConfig.clipRule}
								>
									{#each iconConfig.paths as path, i (i)}
										<path d={path} />
									{/each}
								</svg>
								<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
									{getActionLabel(action.type)}
								</span>
								<span class="text-xs text-gray-400 dark:text-gray-500">
									{formatRelativeTime(action.timestamp)}
								</span>
								<!-- Spacer to push undo button to the right -->
								<div class="flex-1"></div>
								<!-- Undo button -->
								<button
									type="button"
									class="flex flex-shrink-0 cursor-pointer items-center gap-1 rounded-md bg-indigo-500 px-2 py-0.5 text-xs font-medium text-white transition hover:bg-indigo-600 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-indigo-600 dark:hover:bg-indigo-700"
									on:click={() => handleUndo(action)}
									disabled={isUndoing}
								>
									{#if isUndoing}
										<svg class="h-3 w-3 animate-spin" viewBox="0 0 24 24" fill="none">
											<circle
												class="opacity-25"
												cx="12"
												cy="12"
												r="10"
												stroke="currentColor"
												stroke-width="4"
											/>
											<path
												class="opacity-75"
												fill="currentColor"
												d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
											/>
										</svg>
									{:else}
										<svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
											/>
										</svg>
									{/if}
									Undo
								</button>
							</div>

							<!-- Row 2: Notification cards -->
							<div class="mt-2">
								{#if isBulk}
									<button
										type="button"
										class="flex w-full items-center gap-2 rounded-lg border border-gray-300 bg-white px-2.5 py-2 text-left transition hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-800/60 dark:hover:bg-gray-800/40 cursor-pointer"
										on:click={() => toggleExpanded(action.id)}
									>
										<svg
											class="h-3 w-3 flex-shrink-0 text-gray-500 transition-transform dark:text-gray-400 {isExpanded
												? 'rotate-90'
												: ''}"
											fill="none"
											stroke="currentColor"
											viewBox="0 0 24 24"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M9 5l7 7-7 7"
											/>
										</svg>
										<span class="text-sm text-gray-700 dark:text-gray-300">
											{action.notifications.length} notifications
										</span>
									</button>

									{#if isExpanded}
										<div class="mt-2 space-y-1.5 pl-5">
											{#each action.notifications.slice(0, 10) as notification (notification.id)}
												{@const notifIcon = getNotificationIcon(
													notification.subjectType,
													notification.subjectRaw,
													notification.isRead,
													notification.subjectState,
													notification.subjectMerged,
													notification.subjectTitle,
													notification.reason
												)}
												<button
													type="button"
													class="flex w-full items-center gap-2 rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-left transition hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-800/60 dark:hover:bg-gray-800/40 {onNavigateToNotification
														? 'cursor-pointer'
														: ''}"
													on:click={() => handleNotificationClick(notification)}
													disabled={!onNavigateToNotification}
												>
													<svg
														class="h-3.5 w-3.5 flex-shrink-0 {notifIcon.colorClass}"
														viewBox="0 0 16 16"
														fill="currentColor"
													>
														{@html notifIcon.path}
													</svg>
													<div class="min-w-0 flex-1">
														<p class="truncate text-xs text-gray-700 dark:text-gray-300">
															{notification.subjectTitle}
														</p>
													</div>
												</button>
											{/each}
											{#if action.notifications.length > 10}
												<p class="px-2.5 text-xs text-gray-500 dark:text-gray-400">
													+{action.notifications.length - 10} more
												</p>
											{/if}
										</div>
									{/if}
								{:else}
									<!-- Single notification card -->
									{@const notification = action.notifications[0]}
									{@const notifIcon = getNotificationIcon(
										notification.subjectType,
										notification.subjectRaw,
										notification.isRead,
										notification.subjectState,
										notification.subjectMerged,
										notification.subjectTitle,
										notification.reason
									)}
									<button
										type="button"
										class="flex w-full items-center gap-2 rounded-lg border border-gray-300 bg-white px-2.5 py-2 text-left transition hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-800/60 dark:hover:bg-gray-800/40 {onNavigateToNotification
											? 'cursor-pointer'
											: ''}"
										on:click={() => handleNotificationClick(notification)}
										disabled={!onNavigateToNotification}
									>
										<svg
											class="h-4 w-4 flex-shrink-0 {notifIcon.colorClass}"
											viewBox="0 0 16 16"
											fill="currentColor"
										>
											{@html notifIcon.path}
										</svg>
										<div class="min-w-0 flex-1">
											<p class="truncate text-sm text-gray-700 dark:text-gray-300">
												{notification.subjectTitle}
											</p>
											<p class="truncate text-xs text-gray-500 dark:text-gray-400">
												{notification.repoFullName}
											</p>
										</div>
									</button>
								{/if}
							</div>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
{/if}
