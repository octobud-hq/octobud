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

	import Modal from "../shared/Modal.svelte";

	export let open: boolean = false;
	export let onClose: () => void;

	type Shortcut = {
		keys: string[];
		description: string;
		related?: {
			keys: string[];
			description: string;
		}[];
	};

	type ShortcutGroup = {
		title: string;
		shortcuts: Shortcut[];
	};

	const shortcutGroups: ShortcutGroup[] = [
		{
			title: "Navigation",
			shortcuts: [
				{ keys: ["J", "K"], description: "Navigate notification list" },
				{ keys: ["Space"], description: "Toggle notification detail" },
				{ keys: ["⇧P"], description: "Toggle split view" },
				{ keys: ["GG"], description: "Jump to first notification" },
				{ keys: ["⇧G"], description: "Jump to last notification" },
				{ keys: ["[", "]"], description: "Navigate pages" },
				{ keys: ["⇧J", "⇧K"], description: "Navigate views" },

				{
					keys: ["/"],
					description: "Focus query input",
				},
				{ keys: ["⌘B"], description: "Toggle sidebar" },
			],
		},
		{
			title: "Command Palette",
			shortcuts: [
				{ keys: ["⇧⌘K"], description: "Open command palette" },
				{ keys: ["⌘K"], description: "Switch views" },

				{ keys: ["⇧V"], description: "Bulk actions" },
			],
		},
		{
			title: "Actions",
			shortcuts: [
				{
					keys: ["S"],
					description: "Toggle star/unstar notification",
				},
				{
					keys: ["R"],
					description: "Toggle read/unread notification",
				},
				{
					keys: ["E"],
					description: "Archive/unarchive notification",
				},
				{
					keys: ["Z"],
					description: "Snooze/unsnooze notification",
				},
				{
					keys: ["M"],
					description: "Toggle mute/unmute notification",
				},
				{ keys: ["T"], description: "Tag or untag notification" },
				{
					keys: ["I"],
					description: "Allow back into inbox (if skipped inbox due to rule)",
				},
				{ keys: ["O"], description: "Open in GitHub" },
				{ keys: ["H"], description: "Toggle keyboard shortcuts" },
			],
		},
		{
			title: "Multiselect",
			shortcuts: [
				{ keys: ["V"], description: "Toggle multiselect mode" },
				{
					keys: ["X"],
					description: "Toggle selection of focused item",
				},
				{
					keys: ["A"],
					description: "Cycle select all (page → all → none)",
				},
				{ keys: ["S"], description: "Star selected" },
				{ keys: ["⇧S"], description: "Unstar selected" },
				{ keys: ["R"], description: "Mark selected as read" },
				{ keys: ["⇧R"], description: "Mark selected as unread" },
				{ keys: ["E"], description: "Archive selected" },
				{ keys: ["⇧E"], description: "Unarchive selected" },
				{ keys: ["M"], description: "Mute selected" },
				{ keys: ["⇧M"], description: "Unmute selected" },
				{ keys: ["Z"], description: "Open bulk snooze dropdown" },
				{ keys: ["T"], description: "Tag or untag selected" },
				{
					keys: ["I"],
					description: "Allow back into inbox (if skipped inbox due to rule)",
				},
			],
		},
	];
</script>

<Modal {open} {onClose} title="Keyboard Shortcuts">
	<div class="space-y-5 pb-4">
		{#each shortcutGroups as group (group.title)}
			<section class="space-y-3">
				<header
					class="text-xs font-semibold uppercase tracking-wide text-gray-600 dark:text-gray-500"
				>
					{group.title}
				</header>
				<ul class="space-y-1.5">
					{#each group.shortcuts as shortcut, i (shortcut.description + i)}
						<li class="space-y-1.5">
							<div
								class="flex items-center justify-between gap-4 rounded-xl bg-gray-100 dark:bg-gray-800 px-3 py-1.5"
							>
								<span class="text-sm text-gray-700 dark:text-gray-300">{shortcut.description}</span>
								<span class="text-xs text-gray-600 dark:text-gray-400">
									{#each shortcut.keys as key, j (key + j)}
										{#if j > 0}
											<span> / </span>
										{/if}{key}{/each}
								</span>
							</div>
							{#if shortcut.related && shortcut.related.length > 0}
								<ul class="ml-4 space-y-1.5 border-l-2 border-gray-300 dark:border-gray-700 pl-3">
									{#each shortcut.related as relatedShortcut, k (relatedShortcut.description + k)}
										<li
											class="flex items-center justify-between gap-4 rounded-xl bg-gray-100 dark:bg-gray-800 px-3 py-1.5"
										>
											<span class="text-sm text-gray-700 dark:text-gray-300"
												>{relatedShortcut.description}</span
											>
											<span class="text-xs text-gray-600 dark:text-gray-400">
												{#each relatedShortcut.keys as key, l (key + l)}
													{#if l > 0}
														<span> / </span>
													{/if}{key}{/each}
											</span>
										</li>
									{/each}
								</ul>
							{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</div>
</Modal>
