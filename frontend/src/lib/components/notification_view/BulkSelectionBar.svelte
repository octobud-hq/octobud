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

	import { getContext } from "svelte";
	import type { NotificationPageController } from "$lib/state/types";
	import type { Tag } from "$lib/api/types";
	import SelectAllDropdown from "./SelectAllDropdown.svelte";
	import SnoozeDropdown from "$lib/components/shared/SnoozeDropdown.svelte";
	import BulkTagDropdown from "$lib/components/shared/BulkTagDropdown.svelte";
	import { mailOpenIcon, mailClosedIcon } from "$lib/utils/notificationHelpers";
	import { archiveIconPaths, unarchiveIconPath } from "$lib/utils/archiveIcons";
	import { bellOffIconPath, bellOnIconPath } from "$lib/utils/muteIcons";
	import type {
		SnoozeDropdownComponent,
		TagDropdownComponent,
		BulkTagDropdownComponent,
	} from "$lib/types/components";

	export let tags: Tag[] = [];

	const pageController = getContext<NotificationPageController>("notificationPageController");

	// Access stores from page controller
	const { multiselectMode, selectedIds, bulkUpdating, selectAllMode, pageData } =
		pageController.stores;

	// Derive reactive values
	$: selectedCount = $selectedIds.size;
	$: pageCount = $pageData.items.length;
	$: totalCount = $pageData.total;

	let snoozeButtonElement: HTMLButtonElement | null = null;
	let snoozeDropdownComponent: SnoozeDropdownComponent | null = null;
	let isSnoozeDropdownOpen = false;
	let tagButtonElement: HTMLButtonElement | null = null;
	let tagDropdownComponent: BulkTagDropdownComponent | null = null;
	let isTagDropdownOpen = false;

	$: availableTags = tags;

	// Export a method to get the snooze dropdown state
	export function isSnoozeDropdownOpenState(): boolean {
		return isSnoozeDropdownOpen;
	}

	// Export methods to open/close the snooze dropdown
	export function openSnoozeDropdown() {
		isSnoozeDropdownOpen = true;
	}

	export function closeSnoozeDropdown() {
		isSnoozeDropdownOpen = false;
	}

	// Export a method to get the tag dropdown state
	export function isTagDropdownOpenState(): boolean {
		return isTagDropdownOpen;
	}

	// Export methods to open/close the tag dropdown
	export function openTagDropdown() {
		isTagDropdownOpen = true;
	}

	export function closeTagDropdown() {
		isTagDropdownOpen = false;
	}

	$: displayText = (() => {
		if ($selectAllMode === "all") {
			return `All ${totalCount} selected across all pages`;
		} else if ($selectAllMode === "page") {
			return `${selectedCount} selected on this page`;
		} else if (selectedCount > 0) {
			return `${selectedCount} selected`;
		}
		return "Bulk actions";
	})();

	$: hasSelection = selectedCount > 0 || $selectAllMode !== "none";

	// Pass button element to dropdown when it changes
	$: if (snoozeDropdownComponent && snoozeButtonElement) {
		snoozeDropdownComponent.setButtonElement(snoozeButtonElement);
	}

	$: if (tagDropdownComponent && tagButtonElement) {
		tagDropdownComponent.setButtonElement(tagButtonElement);
	}

	function handleSnoozeClick() {
		isSnoozeDropdownOpen = !isSnoozeDropdownOpen;
	}

	function handleSnoozeDropdownClose() {
		isSnoozeDropdownOpen = false;
	}

	function handleSnoozeAction(until: string) {
		void pageController.actions.bulkSnooze(until);
		handleSnoozeDropdownClose();
	}

	function handleUnsnoozeAction() {
		void pageController.actions.bulkUnsnooze();
		handleSnoozeDropdownClose();
	}

	// Create a dummy notification object for the snooze dropdown
	// (not snoozed since we're bulk snoozing)
	const dummyNotification = {
		id: "",
		githubId: "",
		snoozedUntil: null,
		snoozedAt: null,
	} as any;

	function handleTagClick() {
		isTagDropdownOpen = !isTagDropdownOpen;
	}

	function handleTagDropdownClose() {
		isTagDropdownOpen = false;
	}

	async function handleAssignTag(tagId: string) {
		await pageController.actions.bulkAssignTag(tagId);
		handleTagDropdownClose();
	}

	async function handleRemoveTag(tagId: string) {
		await pageController.actions.bulkRemoveTag(tagId);
		handleTagDropdownClose();
	}
</script>

{#if multiselectMode}
	<div
		class="flex flex-wrap items-center gap-3 rounded-2xl border border-indigo-600/40 bg-gray-50/80 dark:bg-gray-950/60 pl-4 pr-3 py-3 text-sm text-indigo-700 dark:text-indigo-200"
	>
		<SelectAllDropdown
			currentMode={$selectAllMode}
			{pageCount}
			{totalCount}
			onSelectPage={pageController.actions.selectAllPage}
			onSelectAll={pageController.actions.selectAllAcrossPages}
			onClearSelection={pageController.actions.clearSelection}
		/>

		<span class="text-xs">{displayText}</span>

		<div class="ml-auto flex flex-wrap items-center gap-2">
			<!-- All views show read, unread, archive, unarchive, mute, unmute, snooze, and unsnooze -->
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkMarkRead()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Mark read"
				title="Mark read (R)"
			>
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
					<path
						d={mailOpenIcon.path}
						fill-rule={mailOpenIcon.fillRule}
						clip-rule={mailOpenIcon.clipRule}
					/>
				</svg>
				<span class="text-gray-500 dark:text-gray-400">R</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkMarkUnread()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Mark unread"
				title="Mark unread (Shift+R)"
			>
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
					<path
						d={mailClosedIcon.path}
						fill-rule={mailClosedIcon.fillRule}
						clip-rule={mailClosedIcon.clipRule}
					/>
				</svg>
				<span class="text-gray-500 dark:text-gray-400">⇧R</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkArchive()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Archive"
				title="Archive (E)"
			>
				<svg
					class="h-4 w-4"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					{#each archiveIconPaths as path (path)}
						<path d={path} />
					{/each}
				</svg>
				<span class="text-gray-500 dark:text-gray-400">E</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkUnarchive()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Unarchive"
				title="Unarchive (Shift+E)"
			>
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
					<path d={unarchiveIconPath} />
				</svg>
				<span class="text-gray-500 dark:text-gray-400">⇧E</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkUnfilter()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Move to inbox"
				title="Move to inbox (I)"
			>
				<svg
					class="h-4 w-4"
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
				<span class="text-gray-500 dark:text-gray-400">I</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkMute()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Mute"
				title="Mute (M)"
			>
				<svg
					class="h-4 w-4"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<path d={bellOffIconPath} />
				</svg>
				<span class="text-gray-500 dark:text-gray-400">M</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkUnmute()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Unmute"
				title="Unmute (Shift+M)"
			>
				<svg
					class="h-4 w-4"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<path d={bellOnIconPath} />
				</svg>
				<span class="text-gray-500 dark:text-gray-400">⇧M</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkStar()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Star"
				title="Star (S)"
			>
				<svg
					class="h-4 w-4 text-violet-500"
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
				<span class="text-gray-500 dark:text-gray-400">S</span>
			</button>
			<button
				class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => void pageController.actions.bulkUnstar()}
				disabled={$bulkUpdating || !hasSelection}
				aria-label="Unstar"
				title="Unstar (Shift+S)"
			>
				<svg
					class="h-4 w-4"
					viewBox="0 0 24 24"
					fill="none"
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
				<span class="text-gray-500 dark:text-gray-400">⇧S</span>
			</button>
			<div class="relative">
				<button
					bind:this={snoozeButtonElement}
					class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
					type="button"
					on:click={handleSnoozeClick}
					disabled={$bulkUpdating || !hasSelection}
					aria-label="Snooze"
					title="Snooze (Z)"
				>
					<svg
						class="h-4 w-4"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path
							d="M12 6V12L16 14M22 12C22 17.5228 17.5228 22 12 22C6.47715 22 2 17.5228 2 12C2 6.47715 6.47715 2 12 2C17.5228 2 22 6.47715 22 12Z"
						/>
					</svg>
					<span class="text-gray-500 dark:text-gray-400">Z</span>
				</button>
				<SnoozeDropdown
					bind:this={snoozeDropdownComponent}
					notification={dummyNotification}
					isOpen={isSnoozeDropdownOpen}
					onSnooze={handleSnoozeAction}
					onUnsnooze={handleUnsnoozeAction}
					onOpenCustomDialog={() => {}}
					onClose={handleSnoozeDropdownClose}
					forceShowUnsnooze={true}
				/>
			</div>
			<div class="relative">
				<button
					bind:this={tagButtonElement}
					class="flex items-center gap-2 rounded-full border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-1.5 text-xs font-semibold text-gray-700 dark:text-white transition hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 cursor-pointer"
					type="button"
					on:click={handleTagClick}
					disabled={$bulkUpdating || !hasSelection}
					aria-label="Manage tags"
					title="Manage tags (T)"
				>
					<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
						<path
							d="M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16z"
						/>
					</svg>
					<span class="text-gray-500 dark:text-gray-400">Tags</span>
				</button>
				<BulkTagDropdown
					bind:this={tagDropdownComponent}
					isOpen={isTagDropdownOpen}
					{availableTags}
					onAssignTag={handleAssignTag}
					onRemoveTag={handleRemoveTag}
					onClose={handleTagDropdownClose}
				/>
			</div>
			<button
				class="rounded-full border border-transparent px-3 py-1 text-xs font-medium text-indigo-700 dark:text-indigo-200 transition hover:bg-gray-100 dark:hover:bg-gray-900/60 disabled:opacity-50 cursor-pointer"
				type="button"
				on:click={() => pageController.actions.clearSelection()}
				disabled={$bulkUpdating || !hasSelection}
			>
				Clear
			</button>
		</div>
	</div>
{/if}
