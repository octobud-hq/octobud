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

	import type { Tag } from "$lib/api/tags";
	import TagInput from "$lib/components/shared/TagInput.svelte";
	import ActionsDropdown from "$lib/components/shared/ActionsDropdown.svelte";

	export let name: string = "";
	export let description: string = "";
	export let skipInbox: boolean = true;
	export let markRead: boolean = false;
	export let star: boolean = false;
	export let archive: boolean = false;
	export let mute: boolean = false;
	export let selectedTags: string[] = [];
	export let enabled: boolean = true;
	export let applyToExisting: boolean = false;
	export let availableTags: Tag[] = [];
	export let showApplyToExisting: boolean = true;
	export let inline: boolean = false; // If true, use more compact styling for inline forms

	// Available actions for the dropdown
	const availableActions = [
		{
			id: "skipInbox",
			label: "Skip inbox",
			description: "Mark as filtered and exclude from inbox (can still appear in custom views)",
		},
		{
			id: "markRead",
			label: "Mark as read",
			description: "Automatically mark matching notifications as read",
		},
		{
			id: "star",
			label: "Star",
			description: "Add star to matching notifications",
		},
		{
			id: "archive",
			label: "Archive",
			description: "Archive matching notifications",
		},
		{
			id: "mute",
			label: "Mute",
			description: "Mute matching notifications (also archives them)",
		},
	];

	// Derive selected actions from boolean props - reactive
	$: selectedActionIds = [
		skipInbox && "skipInbox",
		markRead && "markRead",
		star && "star",
		archive && "archive",
		mute && "mute",
	].filter(Boolean) as string[];

	// Handle dropdown changes - update boolean props
	function handleActionsChange(newIds: string[]) {
		skipInbox = newIds.includes("skipInbox");
		markRead = newIds.includes("markRead");
		star = newIds.includes("star");
		archive = newIds.includes("archive");
		mute = newIds.includes("mute");
	}
</script>

<div class="space-y-5">
	<!-- Name -->
	<div class="space-y-2">
		<div>
			<label for="rule-config-name" class="text-sm font-medium text-gray-900 dark:text-gray-200">
				Rule name {#if !inline}<span class="text-red-400">*</span>{/if}
			</label>
			{#if !inline}
				<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
					Choose a descriptive name for this rule
				</p>
			{/if}
		</div>
		<input
			id="rule-config-name"
			bind:value={name}
			type="text"
			placeholder="e.g., Dependabot PRs"
			class="w-full rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 placeholder-gray-500 dark:placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30"
			required={!inline}
		/>
	</div>

	<!-- Description -->
	<div class="space-y-2">
		<div>
			<label
				for="rule-config-description"
				class="text-sm font-medium text-gray-900 dark:text-gray-200"
			>
				Description
			</label>
			{#if !inline}
				<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
					Optional description of what this rule does
				</p>
			{/if}
		</div>
		<textarea
			id="rule-config-description"
			bind:value={description}
			placeholder="Automatically filter bot-created PRs"
			rows="2"
			class="w-full rounded-lg border border-gray-300 dark:border-gray-800 bg-white dark:bg-gray-950 px-3 py-2 text-sm text-gray-900 dark:text-gray-200 placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/30 resize-none"
		></textarea>
	</div>

	<!-- Actions -->
	<div class="space-y-3">
		<div class="text-sm font-medium text-gray-900 dark:text-gray-200">Actions</div>
		<div class="space-y-3">
			<!-- Actions Dropdown -->
			<div class="space-y-2">
				<ActionsDropdown
					selectedActions={selectedActionIds}
					{availableActions}
					on:change={(e) => handleActionsChange(e.detail)}
				/>
				{#if !inline}
					<p class="text-xs text-gray-600 dark:text-gray-500">
						Select one or more actions to apply to matching notifications
					</p>
				{/if}
			</div>

			<!-- Apply Tags -->
			<div class="mt-4">
				<div class="text-sm text-gray-900 dark:text-gray-200 mb-2">Apply tags</div>
				<TagInput bind:selectedTags {availableTags} placeholder="Add tags..." />
				{#if !inline}
					<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
						Automatically apply tags to matching notifications (comma-separated)
					</p>
				{/if}
			</div>
		</div>
	</div>

	<!-- Enabled toggle -->
	<div class="flex items-center justify-between py-3 border-t border-gray-200 dark:border-gray-800">
		<div>
			<div class="text-sm font-medium text-gray-700 dark:text-gray-300">Enable rule</div>
			{#if !inline}
				<div class="text-xs text-gray-600 dark:text-gray-500">
					Rule will be applied to incoming notifications
				</div>
			{/if}
		</div>
		<label class="flex items-center cursor-pointer">
			<input type="checkbox" bind:checked={enabled} class="sr-only peer" />
			<div
				class="relative w-11 h-6 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-violet-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-violet-600"
			></div>
		</label>
	</div>

	<!-- Apply to existing (create only) -->
	{#if showApplyToExisting}
		<div
			class="flex items-center justify-between py-3 border-t border-gray-200 dark:border-gray-800"
		>
			<div>
				<div class="text-sm font-medium text-gray-700 dark:text-gray-300">
					Apply to existing notifications
				</div>
				{#if !inline}
					<div class="text-xs text-gray-600 dark:text-gray-500">
						Retroactively apply this rule to all existing notifications
					</div>
				{/if}
			</div>
			<label class="flex items-center cursor-pointer">
				<input type="checkbox" bind:checked={applyToExisting} class="sr-only peer" />
				<div
					class="relative w-11 h-6 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-violet-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-violet-600"
				></div>
			</label>
		</div>
	{/if}
</div>
