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

	import type { NotificationView } from "$lib/api/types";
	import { normalizeViewIcon } from "$lib/utils/viewIcons";
	import ViewDropdown from "$lib/components/timeline/ViewDropdown.svelte";

	export let views: NotificationView[] = [];
	export let selectedSlug = "";
	export let onSelectView: (slug: string) => void | Promise<void> = () => {};
	export let onOpenApp: () => void = () => {};
	export let onRefresh: () => void = () => {};
	export let busy = false;

	let isOpen = false;
	let buttonElement: HTMLButtonElement | null = null;

	// The backend flags its five synthesized views, so the dropdown's two sections
	// come straight off the API rather than from a second local list of built-ins.
	$: builtInViews = views.filter((view) => view.systemView);
	$: userViews = views.filter((view) => !view.systemView);
	$: selectedView = views.find((view) => view.slug === selectedSlug) ?? null;
	$: inboxView = views.find((view) => view.slug === "inbox");
	$: isSystemSelected = Boolean(selectedView?.systemView);

	function handleSelect(slug: string) {
		void onSelectView(slug);
	}
</script>

<header
	class="sticky top-0 z-40 flex items-center gap-1.5 border-b border-gray-200 bg-gray-100/80 px-2 py-2 backdrop-blur dark:border-gray-800 dark:bg-gray-900/80"
>
	<button
		type="button"
		class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 text-indigo-50 shadow-soft transition hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40"
		title="Open Octobud"
		aria-label="Open Octobud"
		on:click={onOpenApp}
	>
		<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
			<path
				d="M3 12V15.8C3 16.9201 3 17.4802 3.21799 17.908C3.40973 18.2843 3.71569 18.5903 4.09202 18.782C4.51984 19 5.0799 19 6.2 19H17.8C18.9201 19 19.4802 19 19.908 18.782C20.2843 18.5903 20.5903 18.2843 20.782 17.908C21 17.4802 21 16.9201 21 15.8V12M3 12H6.67452C7.16369 12 7.40829 12 7.63846 12.0553C7.84254 12.1043 8.03763 12.1851 8.21657 12.2947C8.4184 12.4184 8.59136 12.5914 8.93726 12.9373L9.06274 13.0627C9.40865 13.4086 9.5816 13.5816 9.78343 13.7053C9.96237 13.8149 10.1575 13.8957 10.3615 13.9447C10.5917 14 10.8363 14 11.3255 14H12.6745C13.1637 14 13.4083 14 13.6385 13.9447C13.8425 13.8957 14.0376 13.8149 14.2166 13.7053C14.4184 13.5816 14.5914 13.4086 14.9373 13.0627L15.0627 12.9373C15.4086 12.5914 15.5816 12.4184 15.7834 12.2947C15.9624 12.1851 16.1575 12.1043 16.3615 12.0553C16.5917 12 16.8363 12 17.3255 12H21M3 12L5.32639 6.83025C5.78752 5.8055 6.0181 5.29312 6.38026 4.91755C6.70041 4.58556 7.09278 4.33186 7.52691 4.17615C8.01802 4 8.57988 4 9.70361 4H14.2964C15.4201 4 15.982 4 16.4731 4.17615C16.9072 4.33186 17.2996 4.58556 17.6197 4.91755C17.9819 5.29312 18.2125 5.8055 18.6736 6.83025L21 12"
				stroke="currentColor"
				stroke-width="1.5"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		</svg>
	</button>

	<!-- Reuses the sidebar nav-item idiom so the trigger reads as an Octobud view row. -->
	<button
		bind:this={buttonElement}
		type="button"
		class="group flex min-w-0 flex-1 items-center gap-2 rounded-md border border-transparent px-2 py-1.5 text-left transition hover:bg-gray-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 dark:hover:bg-gray-800/50"
		aria-haspopup="menu"
		aria-expanded={isOpen}
		on:click={() => (isOpen = !isOpen)}
	>
		<span class="flex flex-shrink-0 items-center justify-center text-base" aria-hidden="true">
			{#if isSystemSelected}
				<svg
					class="h-[18px] w-[18px] text-gray-700 dark:text-gray-300"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<path
						d="M3 12V15.8C3 16.9201 3 17.4802 3.21799 17.908C3.40973 18.2843 3.71569 18.5903 4.09202 18.782C4.51984 19 5.0799 19 6.2 19H17.8C18.9201 19 19.4802 19 19.908 18.782C20.2843 18.5903 20.5903 18.2843 20.782 17.908C21 17.4802 21 16.9201 21 15.8V12M3 12H6.67452C7.16369 12 7.40829 12 7.63846 12.0553C7.84254 12.1043 8.03763 12.1851 8.21657 12.2947C8.4184 12.4184 8.59136 12.5914 8.93726 12.9373L9.06274 13.0627C9.40865 13.4086 9.5816 13.5816 9.78343 13.7053C9.96237 13.8149 10.1575 13.8957 10.3615 13.9447C10.5917 14 10.8363 14 11.3255 14H12.6745C13.1637 14 13.4083 14 13.6385 13.9447C13.8425 13.8957 14.0376 13.8149 14.2166 13.7053C14.4184 13.5816 14.5914 13.4086 14.9373 13.0627L15.0627 12.9373C15.4086 12.5914 15.5816 12.4184 15.7834 12.2947C15.9624 12.1851 16.1575 12.1043 16.3615 12.0553C16.5917 12 16.8363 12 17.3255 12H21M3 12L5.32639 6.83025C5.78752 5.8055 6.0181 5.29312 6.38026 4.91755C6.70041 4.58556 7.09278 4.33186 7.52691 4.17615C8.01802 4 8.57988 4 9.70361 4H14.2964C15.4201 4 15.982 4 16.4731 4.17615C16.9072 4.33186 17.2996 4.58556 17.6197 4.91755C17.9819 5.29312 18.2125 5.8055 18.6736 6.83025L21 12"
					/>
				</svg>
			{:else}
				{normalizeViewIcon(selectedView?.icon)}
			{/if}
		</span>

		<span class="min-w-0 flex-1 truncate text-[14px] font-normal text-gray-900 dark:text-gray-100">
			{selectedView?.name ?? "Loading…"}
		</span>

		{#if selectedView && selectedView.unreadCount > 0}
			<span
				class="flex h-6 flex-shrink-0 items-center justify-center rounded-full bg-gray-200 px-2 text-[12px] text-gray-900 dark:bg-gray-800/60 dark:text-gray-100"
				aria-label={`${selectedView.unreadCount} unread`}
			>
				{selectedView.unreadCount}
			</span>
		{/if}

		<svg
			class="h-4 w-4 flex-shrink-0 text-gray-500 transition dark:text-gray-400 {isOpen
				? 'rotate-180'
				: ''}"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			<polyline points="6 9 12 15 18 9"></polyline>
		</svg>
	</button>

	<button
		type="button"
		class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg text-gray-600 transition hover:bg-gray-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/40 disabled:opacity-40 dark:text-gray-400 dark:hover:bg-gray-800/50"
		title="Refresh"
		aria-label="Refresh"
		disabled={busy}
		on:click={onRefresh}
	>
		<svg
			class="h-4 w-4 {busy ? 'animate-spin' : ''}"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			<polyline points="23 4 23 10 17 10"></polyline>
			<polyline points="1 20 1 14 7 14"></polyline>
			<path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
		</svg>
	</button>
</header>

<ViewDropdown
	{isOpen}
	{buttonElement}
	{builtInViews}
	{userViews}
	{inboxView}
	selectedViewId={selectedView?.id ?? ""}
	selectedViewSlug={selectedSlug}
	onSelectView={handleSelect}
	onNewView={onOpenApp}
	onClose={() => (isOpen = false)}
/>
