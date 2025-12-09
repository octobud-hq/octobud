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

	import { getContext, onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import CommandPalette from "../command_palette/CommandPalette.svelte";
	import type { NotificationView, Tag } from "$lib/api/types";
	import type { NotificationPageController } from "$lib/state/types";
	import { githubUsername } from "$lib/stores/authStore";
	import type { CommandPaletteComponent } from "$lib/types/components";

	// Get page controller from context
	const pageController = getContext<NotificationPageController>("notificationPageController");
	export let defaultViewSlug: string;
	export let defaultViewDisplayName: string;
	export let detailOpen: boolean;

	export let sidebarCollapsed: boolean = false; // New: sidebar collapsed state
	export let onToggleSidebar: (() => void) | null = null; // New: sidebar toggle handler
	export let builtInViews: NotificationView[];
	export let views: NotificationView[];
	export let tags: Tag[] = [];
	export let selectedViewSlug: string;
	export let onLogoClick: () => void;
	export let onSelectView: (slug: string) => void | Promise<void>;
	export let onShowMoreShortcuts: () => void = () => {};
	export let onShowQueryGuide: () => void = () => {};

	let commandPaletteComponent: CommandPaletteComponent | null = null;

	export function getCommandPalette() {
		return commandPaletteComponent;
	}

	let avatarDropdownOpen = false;
	let avatarButtonElement: HTMLButtonElement;
	let avatarDropdownElement: HTMLDivElement;

	function toggleAvatarDropdown() {
		avatarDropdownOpen = !avatarDropdownOpen;
	}

	function handleClickOutside(event: MouseEvent) {
		if (
			avatarDropdownElement &&
			!avatarDropdownElement.contains(event.target as Node) &&
			avatarButtonElement &&
			!avatarButtonElement.contains(event.target as Node)
		) {
			avatarDropdownOpen = false;
		}
	}

	onMount(() => {
		document.addEventListener("click", handleClickOutside);
		return () => {
			document.removeEventListener("click", handleClickOutside);
		};
	});

	$: username = $githubUsername;
	$: avatarUrl = username ? `https://github.com/${encodeURIComponent(username)}.png` : null;
	let avatarLoadFailed = false;

	function handleAvatarError() {
		avatarLoadFailed = true;
	}

	// Reset avatar error state when username changes
	$: {
		if (username) {
			avatarLoadFailed = false;
		}
	}

	async function handleProfileClick() {
		avatarDropdownOpen = false;
		await goto(resolve("/settings"));
		// Set hash fragment after navigation
		// Use setTimeout to ensure navigation has completed
		setTimeout(() => {
			if (typeof window !== "undefined") {
				window.location.hash = "account";
			}
		}, 0);
	}
</script>

<header
	class="sticky top-0 z-40 border-b border-gray-200 bg-gray-100/80 backdrop-blur dark:border-gray-800 dark:bg-gray-900/80 min-h-[3.5rem]"
>
	<div
		class="relative mx-auto flex w-full items-center justify-between gap-4 pl-2 pr-4 py-1.5 sm:pr-6 min-h-[3.5rem]"
	>
		<!-- Left section: Hamburger + Logo -->
		<div class="flex flex-shrink-0 items-center gap-3">
			<!-- Hamburger menu button -->
			{#if onToggleSidebar}
				<button
					type="button"
					class="flex h-9 w-9 items-center justify-center rounded-lg text-gray-700 transition hover:bg-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/70 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-gray-300 dark:hover:bg-gray-800 dark:focus-visible:ring-offset-gray-900 cursor-pointer"
					on:click={onToggleSidebar}
					title={sidebarCollapsed ? "Expand sidebar (Cmd/Ctrl+B)" : "Collapse sidebar (Cmd/Ctrl+B)"}
					aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
				>
					<svg
						class="h-5 w-5"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<line x1="3" y1="12" x2="21" y2="12" />
						<line x1="3" y1="6" x2="21" y2="6" />
						<line x1="3" y1="18" x2="21" y2="18" />
					</svg>
				</button>
			{/if}
			<button
				type="button"
				class="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 text-indigo-50 shadow-soft transition hover:scale-105 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/70 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-900 cursor-pointer"
				on:click={onLogoClick}
				title={`Go to ${defaultViewDisplayName}`}
				aria-label={`Go to ${defaultViewDisplayName}`}
			>
				<img src="/baby_octo.svg" alt="Octobud icon" class="h-6 w-6" />
				<span class="sr-only">Octobud logo</span>
			</button>
		</div>

		<!-- Right section: Avatar dropdown -->
		<div class="relative flex flex-shrink-0 items-center gap-2">
			<button
				bind:this={avatarButtonElement}
				type="button"
				on:click={toggleAvatarDropdown}
				class="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 text-sm font-medium text-white shadow-soft transition hover:scale-105 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/70 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-900 cursor-pointer overflow-hidden"
				title={username || "User menu"}
				aria-label="User menu"
			>
				{#if avatarUrl && !avatarLoadFailed}
					<img
						src={avatarUrl}
						alt={`${username}'s avatar`}
						class="h-full w-full object-cover"
						on:error={handleAvatarError}
					/>
				{:else}
					{username ? username.charAt(0).toUpperCase() : "U"}
				{/if}
			</button>

			{#if avatarDropdownOpen}
				<div
					bind:this={avatarDropdownElement}
					class="absolute right-0 top-full z-50 mt-2 w-56 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-800 dark:bg-gray-900 overflow-hidden"
					role="menu"
				>
					<button
						type="button"
						on:click={handleProfileClick}
						class="w-full px-4 py-3 border-b border-gray-200 dark:border-gray-800 flex items-center gap-3 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors cursor-pointer text-left"
					>
						<div
							class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 text-sm font-medium text-white shadow-soft overflow-hidden"
						>
							{#if avatarUrl && !avatarLoadFailed}
								<img
									src={avatarUrl}
									alt={`${username}'s avatar`}
									class="h-full w-full object-cover"
									on:error={handleAvatarError}
								/>
							{:else}
								{username ? username.charAt(0).toUpperCase() : "U"}
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<p class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
								{username || "User"}
							</p>
							<p class="text-xs text-gray-500 dark:text-gray-400">Connected via GitHub</p>
						</div>
					</button>
				</div>
			{/if}
		</div>

		<!-- Center section: Command palette (absolutely positioned to center of page) -->
		<div
			class="pointer-events-none absolute left-1/2 top-1/2 z-10 w-full max-w-[240px] sm:max-w-sm md:max-w-md lg:max-w-lg xl:max-w-xl -translate-x-1/2 -translate-y-1/2 px-4 sm:px-6 md:px-8"
		>
			<div class="pointer-events-auto max-w-full">
				<CommandPalette
					bind:this={commandPaletteComponent}
					{detailOpen}
					placeholder="Run a command"
					{builtInViews}
					{views}
					{tags}
					{selectedViewSlug}
					{defaultViewSlug}
					{onSelectView}
					onRefresh={pageController.actions.refresh}
					{onShowMoreShortcuts}
					{onShowQueryGuide}
					onMarkAllRead={pageController.actions.markAllRead}
					onMarkAllUnread={pageController.actions.markAllUnread}
					onMarkAllArchive={pageController.actions.markAllArchive}
				/>
			</div>
		</div>
	</div>
</header>
