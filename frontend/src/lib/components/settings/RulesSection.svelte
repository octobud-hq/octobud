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

	import type { Rule } from "$lib/api/rules";
	import type { NotificationView } from "$lib/api/types";
	import type { Tag } from "$lib/api/tags";
	import { deleteRule, reorderRules, updateRule } from "$lib/api/rules";
	import { fetchViews } from "$lib/api/views";
	import { toastStore } from "$lib/stores/toastStore";
	import { invalidateAll } from "$app/navigation";
	import RuleDialog from "$lib/components/dialogs/RuleDialog.svelte";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";
	import { onMount } from "svelte";
	import { SvelteSet } from "svelte/reactivity";
	import { archiveIconPaths } from "$lib/utils/archiveIcons";
	import { bellOffIconPath } from "$lib/utils/muteIcons";
	import { mailOpenIcon } from "$lib/utils/notificationHelpers";
	import { useDragAndDropReorder } from "$lib/composables/useDragAndDropReorder";
	import { tokenizeForSyntax, type SyntaxToken } from "$lib/utils/querySyntaxTokenizer";

	export let rules: Rule[];
	export let tags: Tag[] = [];

	let showRuleDialog = false;
	let editingRule: Rule | null = null;
	let ruleDeleteConfirmOpen = false;
	let ruleDeleting = false;
	let views: NotificationView[] = [];
	let expandedRuleIds = new SvelteSet<string>();

	// Use drag-and-drop composable
	const dragAndDrop = useDragAndDropReorder(
		rules,
		async (ruleIds: string[]) => {
			await reorderRules(ruleIds);
			await invalidateAll();
			toastStore.show("Rules reordered successfully", "success");
		},
		{
			onSaveError: (error) => {
				toastStore.show(`Failed to reorder rules: ${error}`, "error");
			},
		}
	);

	// Destructure stores for easier access
	const {
		reorderMode: dragAndDropReorderMode,
		localOrder: dragAndDropLocalOrder,
		draggedOverId: dragAndDropDraggedOverId,
		reordering: dragAndDropReordering,
	} = dragAndDrop;

	// Update items when rules change
	$: dragAndDrop.updateItems(rules);

	$: sortedRules = $dragAndDropReorderMode
		? $dragAndDropLocalOrder
		: [...rules].sort((a, b) => a.displayOrder - b.displayOrder);

	function toggleRuleExpanded(ruleId: string) {
		if (expandedRuleIds.has(ruleId)) {
			expandedRuleIds.delete(ruleId);
		} else {
			expandedRuleIds.add(ruleId);
		}
		expandedRuleIds = expandedRuleIds; // Trigger reactivity
	}

	// Create a reactive map of rule IDs to their linked views
	$: ruleViewsMap = new Map(
		sortedRules.filter((r) => r.viewId).map((r) => [r.id, views.find((v) => v.id === r.viewId)])
	);

	onMount(async () => {
		try {
			views = await fetchViews();
		} catch (error) {}
	});

	function handleNewRule() {
		editingRule = null;
		showRuleDialog = true;
	}

	function handleEditRule(rule: Rule) {
		editingRule = rule;
		showRuleDialog = true;
	}

	async function handleToggleEnabled(rule: Rule, enabled: boolean) {
		try {
			await updateRule(rule.id, { enabled });
			toastStore.show("Rule " + (enabled ? "enabled" : "disabled"), "success");
			await invalidateAll();
		} catch (error) {
			toastStore.show(`Failed to ${enabled ? "enable" : "disable"} rule: ${error}`, "error");
		}
	}

	// Rule delete handlers
	function requestRuleDelete() {
		ruleDeleteConfirmOpen = true;
	}

	function cancelRuleDelete() {
		ruleDeleteConfirmOpen = false;
	}

	async function confirmRuleDelete() {
		if (!editingRule) return;

		ruleDeleting = true;
		try {
			await deleteRule(editingRule.id);
			toastStore.show("Rule deleted successfully", "success");

			// Close dialogs
			ruleDeleteConfirmOpen = false;
			showRuleDialog = false;
			editingRule = null;

			// Refresh rules
			await invalidateAll();
		} catch (error) {
			toastStore.show(`Failed to delete rule: ${error}`, "error");
		} finally {
			ruleDeleting = false;
		}
	}

	function toggleReorderMode() {
		dragAndDrop.toggleReorderMode();
	}

	function cancelReorder() {
		dragAndDrop.cancelReorder();
	}

	function getTag(tagId: string): Tag | undefined {
		return tags.find((t) => t.id === tagId);
	}

	// Convert hex color to rgba for opacity
	function hexToRgba(hex: string, alpha: number): string {
		// Remove # if present
		const cleanHex = hex.replace("#", "");

		// Parse hex values
		const r = parseInt(cleanHex.substring(0, 2), 16);
		const g = parseInt(cleanHex.substring(2, 4), 16);
		const b = parseInt(cleanHex.substring(4, 6), 16);

		return `rgba(${r}, ${g}, ${b}, ${alpha})`;
	}

	// Get color with opacity, handling hex colors
	function getColorWithOpacity(color: string, opacity: number): string {
		// If it's a hex color (starts with #), convert to rgba
		if (color.startsWith("#") && color.length === 7) {
			return hexToRgba(color, opacity);
		}
		// For other formats (rgb, rgba, named colors), return the color as-is
		// The browser will handle it, though opacity won't be applied correctly
		// In practice, tags should use hex colors
		return color;
	}

	function getActionsChips(actions: Rule["actions"]): {
		label: string;
		iconType?: "star" | "tag" | "check" | "mail" | "archive" | "mute";
		tagId?: string;
		tagColor?: string;
	}[] {
		const chips: {
			label: string;
			iconType?: "star" | "tag" | "check" | "mail" | "archive" | "mute";
			tagId?: string;
			tagColor?: string;
		}[] = [];
		if (actions.skipInbox) chips.push({ label: "Skip inbox", iconType: "check" });
		if (actions.markRead) chips.push({ label: "Mark read", iconType: "mail" });
		if (actions.star) chips.push({ label: "Star", iconType: "star" });
		if (actions.archive) chips.push({ label: "Archive", iconType: "archive" });
		if (actions.mute) chips.push({ label: "Mute", iconType: "mute" });
		if (actions.assignTags && actions.assignTags.length > 0) {
			// assignTags now contains tag IDs, look up names and colors for display
			for (const tagId of actions.assignTags) {
				const tag = getTag(tagId);
				chips.push({
					label: tag?.name || tagId,
					iconType: "tag",
					tagId: tagId,
					tagColor: tag?.color,
				});
			}
		}
		return chips;
	}

	// Separate action chips from tag chips
	function separateActionChips(chips: ReturnType<typeof getActionsChips>): {
		actionChips: ReturnType<typeof getActionsChips>;
		tagCount: number;
	} {
		const actionChips = chips.filter((chip) => chip.iconType !== "tag");
		const tagCount = chips.filter((chip) => chip.iconType === "tag").length;
		return { actionChips, tagCount };
	}
</script>

<div>
	<div class="flex items-center justify-between mb-6">
		<div>
			<h2 class="text-md font-medium text-gray-900 dark:text-gray-100">Rules</h2>
			<p class="text-xs text-gray-600 dark:text-gray-400 mt-1">
				Automatically organize notifications based on custom queries
			</p>
		</div>
		<button
			on:click={handleNewRule}
			class="rounded-full bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-indigo-700 cursor-pointer"
		>
			New Rule
		</button>
	</div>

	{#if sortedRules.length === 0}
		<div
			class="text-center py-12 bg-gray-50 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800"
		>
			<svg
				class="w-12 h-12 mx-auto text-gray-500 dark:text-gray-600 mb-4"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"
				/>
			</svg>
			<h3 class="text-md font-medium text-gray-700 dark:text-gray-300 mb-2">No rules yet</h3>
			<p class="text-xs text-gray-600 dark:text-gray-500 mb-4">
				Create your first rule to automatically organize notifications
			</p>
			<button
				on:click={handleNewRule}
				class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-full transition-colors cursor-pointer"
			>
				Create Rule
			</button>
		</div>
	{:else}
		<div class="space-y-2">
			{#each sortedRules as rule (rule.id)}
				{@const actionChips = getActionsChips(rule.actions)}
				{@const { actionChips: previewActionChips, tagCount } = separateActionChips(actionChips)}
				{@const syntaxTokens = !rule.viewId ? tokenizeForSyntax(rule.query) : []}
				{@const linkedView = ruleViewsMap.get(rule.id)}
				{@const isExpanded = expandedRuleIds.has(rule.id)}
				<div
					class="bg-gray-50 dark:bg-gray-900/60 border border-gray-200 dark:border-gray-800 rounded-lg transition-colors {$dragAndDropReorderMode
						? 'cursor-move hover:border-gray-300 dark:hover:border-gray-700'
						: ''} {$dragAndDropDraggedOverId === rule.id ? 'border-blue-600' : ''}"
					draggable={$dragAndDropReorderMode}
					role={$dragAndDropReorderMode ? "button" : undefined}
					aria-label={$dragAndDropReorderMode ? `Reorder rule ${rule.name}` : undefined}
					on:dragstart={(e) => dragAndDrop.handleDragStart(rule.id, e)}
					on:dragover={(e) => dragAndDrop.handleDragOver(rule.id, e)}
					on:drop={(e) => dragAndDrop.handleDrop(rule.id, e)}
					on:dragend={dragAndDrop.handleDragEnd}
				>
					<!-- Collapsed header: All in one row -->
					<button
						type="button"
						class="flex items-center justify-between gap-3 px-3 py-3 w-full text-left {$dragAndDropReorderMode
							? 'cursor-move'
							: 'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800/30 transition-colors'}"
						on:click={() => {
							if (!$dragAndDropReorderMode) {
								toggleRuleExpanded(rule.id);
							}
						}}
						disabled={$dragAndDropReorderMode}
					>
						<div class="flex items-center gap-3 flex-1 min-w-0">
							{#if $dragAndDropReorderMode}
								<svg
									class="w-4 h-4 text-gray-500 dark:text-gray-600 flex-shrink-0"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M4 8h16M4 16h16"
									/>
								</svg>
							{/if}
							<!-- Enabled/Disabled indicator dot -->
							<div
								class="w-2 h-2 rounded-full flex-shrink-0 {rule.enabled
									? 'bg-green-500'
									: 'bg-gray-500'}"
								title={rule.enabled ? "Enabled" : "Disabled"}
							></div>
							<!-- Rule name -->
							<h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100 truncate min-w-0">
								{rule.name}
							</h3>
							<!-- Type badge -->
							{#if rule.viewId}
								<span
									class="inline-flex items-center px-2 py-0.5 rounded-md bg-indigo-100 dark:bg-indigo-950/30 border border-indigo-300 dark:border-indigo-800/40 text-xs text-indigo-700 dark:text-indigo-200 font-medium flex-shrink-0"
								>
									View-based
								</span>
							{:else}
								<span
									class="inline-flex items-center px-2 py-0.5 rounded-md bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-700 text-xs text-gray-700 dark:text-gray-300 font-medium flex-shrink-0"
								>
									Query-based
								</span>
							{/if}
							<!-- Action mini chips -->
							{#if previewActionChips.length > 0 || tagCount > 0}
								<div class="flex items-center gap-1.5 flex-shrink-0">
									{#each previewActionChips as chip, i (chip.label + i)}
										<span
											class="inline-flex items-center justify-center w-5 h-5 rounded-full border bg-gray-100 dark:bg-gray-800 border-gray-300 dark:border-gray-700 text-gray-700 dark:text-gray-300"
											title={chip.label}
										>
											{#if chip.iconType === "star"}
												<svg
													class="w-3 h-3 text-indigo-400"
													viewBox="0 0 24 24"
													fill="currentColor"
													stroke="none"
												>
													<path
														d="M11.245 4.174C11.4765 3.50808 11.5922 3.17513 11.7634 3.08285C11.9115 3.00298 12.0898 3.00298 12.238 3.08285C12.4091 3.17513 12.5248 3.50808 12.7563 4.174L14.2866 8.57639C14.3525 8.76592 14.3854 8.86068 14.4448 8.93125C14.4972 8.99359 14.5641 9.04218 14.6396 9.07278C14.725 9.10743 14.8253 9.10947 15.0259 9.11356L19.6857 9.20852C20.3906 9.22288 20.743 9.23007 20.8837 9.36432C21.0054 9.48051 21.0605 9.65014 21.0303 9.81569C20.9955 10.007 20.7146 10.2199 20.1528 10.6459L16.4387 13.4616C16.2788 13.5829 16.1989 13.6435 16.1501 13.7217C16.107 13.7909 16.0815 13.8695 16.0757 13.9507C16.0692 14.0427 16.0982 14.1387 16.1563 14.3308L17.506 18.7919C17.7101 19.4667 17.8122 19.8041 17.728 19.9793C17.6551 20.131 17.5108 20.2358 17.344 20.2583C17.1513 20.2842 16.862 20.0829 16.2833 19.6802L12.4576 17.0181C12.2929 16.9035 12.2106 16.8462 12.1211 16.8239C12.042 16.8043 11.9593 16.8043 11.8803 16.8239C11.7908 16.8462 11.7084 16.9035 11.5437 17.0181L7.71805 19.6802C7.13937 20.0829 6.85003 20.2842 6.65733 20.2583C6.49055 20.2358 6.34626 20.131 6.27337 19.9793C6.18915 19.8041 6.29123 19.4667 6.49538 18.7919L7.84503 14.3308C7.90313 14.1387 7.93218 14.0427 7.92564 13.9507C7.91986 13.8695 7.89432 13.7909 7.85123 13.7217C7.80246 13.6435 7.72251 13.5829 7.56262 13.4616L3.84858 10.6459C3.28678 10.2199 3.00588 10.007 2.97101 9.81569C2.94082 9.65014 2.99594 9.48051 3.11767 9.36432C3.25831 9.23007 3.61074 9.22289 4.31559 9.20852L8.9754 9.11356C9.176 9.10947 9.27631 9.10743 9.36177 9.07278C9.43724 9.04218 9.50414 8.99359 9.55645 8.93125C9.61593 8.86068 9.64887 8.76592 9.71475 8.57639L11.245 4.174Z"
													/>
												</svg>
											{:else if chip.iconType === "check"}
												<svg
													class="w-3 h-3"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
													stroke-linecap="round"
													stroke-linejoin="round"
												>
													<path d="m4.5 12.75 4.5 4.5 10.5-10.5" />
												</svg>
											{:else if chip.iconType === "mail"}
												<svg class="w-3 h-3" viewBox="0 0 24 24" fill="currentColor">
													<path
														d={mailOpenIcon.path}
														fill-rule={mailOpenIcon.fillRule}
														clip-rule={mailOpenIcon.clipRule}
														stroke="currentColor"
														stroke-width=".3"
														stroke-linecap="round"
														stroke-linejoin="round"
													/>
												</svg>
											{:else if chip.iconType === "archive"}
												<svg
													class="w-3 h-3"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
													stroke-linecap="round"
													stroke-linejoin="round"
												>
													{#each archiveIconPaths as path (path)}
														<path d={path} />
													{/each}
												</svg>
											{:else if chip.iconType === "mute"}
												<svg
													class="w-3 h-3"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"
													stroke-linecap="round"
													stroke-linejoin="round"
												>
													<path d={bellOffIconPath} />
												</svg>
											{/if}
										</span>
									{/each}
									{#if tagCount > 0}
										<span
											class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full border bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border-blue-300 dark:border-blue-800/40 text-[10px] font-medium"
										>
											<svg class="w-2.5 h-2.5" viewBox="0 0 24 24" fill="currentColor">
												<path
													d="M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16z"
												/>
											</svg>
											<span>({tagCount} tag{tagCount !== 1 ? "s" : ""})</span>
										</span>
									{/if}
								</div>
							{/if}
						</div>
						{#if !$dragAndDropReorderMode}
							<svg
								class="w-4 h-4 text-gray-600 dark:text-gray-400 flex-shrink-0 transition-transform {isExpanded
									? 'rotate-180'
									: ''}"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M19 9l-7 7-7-7"
								/>
							</svg>
						{/if}
					</button>

					<!-- Expanded details -->
					{#if isExpanded && !$dragAndDropReorderMode}
						<div class="border-t border-gray-200 dark:border-gray-800 pt-5 px-4 pb-2 space-y-3">
							<!-- Query and Actions in lighter container -->
							<div class="bg-gray-100/40 dark:bg-gray-800/40 rounded-lg p-4 space-y-5">
								<!-- Query or View -->
								<div class="space-y-2">
									<div class="flex items-start justify-between gap-4">
										<div class="flex-1">
											<div class="text-sm font-medium text-gray-900 dark:text-gray-200">Query</div>
											<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
												{#if rule.viewId}
													The rule automatically uses this view's current query
												{:else}
													Use the same query syntax as custom views to filter notifications
												{/if}
											</p>
										</div>
										<!-- Enabled/Disabled Toggle -->
										<label class="flex items-center gap-2 cursor-pointer flex-shrink-0">
											<span class="text-sm font-medium text-gray-700 dark:text-gray-300">
												Enabled
											</span>
											<input
												type="checkbox"
												checked={rule.enabled}
												on:change={(e) => handleToggleEnabled(rule, e.currentTarget.checked)}
												class="sr-only peer"
											/>
											<div
												class="relative w-9 h-5 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"
											></div>
										</label>
									</div>
									{#if rule.viewId}
										{#if linkedView}
											<div
												class="flex items-center gap-2 px-3 py-1.5 bg-indigo-100 dark:bg-indigo-950/30 border border-indigo-300 dark:border-indigo-800/40 rounded-md"
											>
												<span class="text-lg flex-shrink-0">{linkedView.icon}</span>
												<span class="text-sm text-indigo-700 dark:text-indigo-200 font-medium"
													>{linkedView.name}</span
												>
												<span class="text-xs text-indigo-600/80 dark:text-indigo-400/60 ml-auto"
													>View-based rule</span
												>
											</div>
										{:else}
											<div
												class="flex items-center gap-2 px-3 py-1.5 bg-indigo-100 dark:bg-indigo-950/30 border border-indigo-300 dark:border-indigo-800/40 rounded-md"
											>
												<span class="text-sm text-indigo-700 dark:text-indigo-200 font-medium"
													>Linked to view (loading...)</span
												>
												<span class="text-xs text-indigo-600/80 dark:text-indigo-400/60 ml-auto"
													>View-based rule</span
												>
											</div>
										{/if}
									{:else}
										<div
											class="px-3 py-1.5 bg-gray-50 dark:bg-gray-900/90 border border-gray-300 dark:border-gray-700 rounded-md font-mono text-xs leading-relaxed overflow-x-auto"
										>
											<div class="whitespace-pre">
												{#each syntaxTokens as token, i (token.text + i)}<!--
												-->{#if token.type === "key"}<!--
													--><span
															class="text-blue-600 dark:text-blue-400">{token.text}</span
														><!--
												-->{:else if token.type === "negation"}<!--
													--><span
															class="text-red-600 dark:text-red-400">{token.text}</span
														><!--
												-->{:else if token.type === "colon"}<!--
													--><span
															class="text-gray-600 dark:text-gray-500">{token.text}</span
														><!--
												-->{:else if token.type === "value"}<!--
													--><span
															class="text-blue-500 dark:text-blue-300">{token.text}</span
														><!--
												-->{:else if token.type === "operator"}<!--
													--><span
															class="text-purple-600 dark:text-purple-400">{token.text}</span
														><!--
												-->{:else if token.type === "paren"}<!--
													--><span
															class="text-yellow-600 dark:text-yellow-400">{token.text}</span
														><!--
												-->{:else if token.type === "whitespace"}<!--
													--><span
															>{token.text}</span
														><!--
												-->{:else}<!--
													--><span class="text-gray-800 dark:text-gray-300"
															>{token.text}</span
														><!--
												-->{/if}<!--
											-->{/each}
											</div>
										</div>
									{/if}
								</div>

								<!-- All action chips -->
								<div class="space-y-2">
									<div>
										<div class="text-sm font-medium text-gray-900 dark:text-gray-200">Actions</div>
										<p class="text-xs text-gray-600 dark:text-gray-500 mt-1">
											Select one or more actions to apply to matching notifications
										</p>
									</div>
									<!-- Action chips -->
									<div class="flex items-center gap-2 flex-wrap">
										{#if actionChips.length > 0}
											{#each actionChips as chip, i (chip.label + i)}
												{@const isTagChip = chip.iconType === "tag"}
												{@const hasTagColor = isTagChip && chip.tagColor}
												{@const isActionChip = !isTagChip}
												{@const tagBgColor =
													hasTagColor && chip.tagColor
														? getColorWithOpacity(chip.tagColor, 0.2)
														: undefined}
												<span
													class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-xs font-semibold {isActionChip
														? 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-200 border-gray-300 dark:border-gray-700'
														: hasTagColor
															? ''
															: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border-blue-300 dark:border-blue-800/40'}"
													style={hasTagColor && chip.tagColor
														? `background-color: ${tagBgColor}; color: ${chip.tagColor}; border-color: ${chip.tagColor};`
														: undefined}
												>
													{#if chip.iconType === "star"}
														<svg
															class="w-3.5 h-3.5 text-indigo-400"
															viewBox="0 0 24 24"
															fill="currentColor"
															stroke="none"
														>
															<path
																d="M11.245 4.174C11.4765 3.50808 11.5922 3.17513 11.7634 3.08285C11.9115 3.00298 12.0898 3.00298 12.238 3.08285C12.4091 3.17513 12.5248 3.50808 12.7563 4.174L14.2866 8.57639C14.3525 8.76592 14.3854 8.86068 14.4448 8.93125C14.4972 8.99359 14.5641 9.04218 14.6396 9.07278C14.725 9.10743 14.8253 9.10947 15.0259 9.11356L19.6857 9.20852C20.3906 9.22288 20.743 9.23007 20.8837 9.36432C21.0054 9.48051 21.0605 9.65014 21.0303 9.81569C20.9955 10.007 20.7146 10.2199 20.1528 10.6459L16.4387 13.4616C16.2788 13.5829 16.1989 13.6435 16.1501 13.7217C16.107 13.7909 16.0815 13.8695 16.0757 13.9507C16.0692 14.0427 16.0982 14.1387 16.1563 14.3308L17.506 18.7919C17.7101 19.4667 17.8122 19.8041 17.728 19.9793C17.6551 20.131 17.5108 20.2358 17.344 20.2583C17.1513 20.2842 16.862 20.0829 16.2833 19.6802L12.4576 17.0181C12.2929 16.9035 12.2106 16.8462 12.1211 16.8239C12.042 16.8043 11.9593 16.8043 11.8803 16.8239C11.7908 16.8462 11.7084 16.9035 11.5437 17.0181L7.71805 19.6802C7.13937 20.0829 6.85003 20.2842 6.65733 20.2583C6.49055 20.2358 6.34626 20.131 6.27337 19.9793C6.18915 19.8041 6.29123 19.4667 6.49538 18.7919L7.84503 14.3308C7.90313 14.1387 7.93218 14.0427 7.92564 13.9507C7.91986 13.8695 7.89432 13.7909 7.85123 13.7217C7.80246 13.6435 7.72251 13.5829 7.56262 13.4616L3.84858 10.6459C3.28678 10.2199 3.00588 10.007 2.97101 9.81569C2.94082 9.65014 2.99594 9.48051 3.11767 9.36432C3.25831 9.23007 3.61074 9.22289 4.31559 9.20852L8.9754 9.11356C9.176 9.10947 9.27631 9.10743 9.36177 9.07278C9.43724 9.04218 9.50414 8.99359 9.55645 8.93125C9.61593 8.86068 9.64887 8.76592 9.71475 8.57639L11.245 4.174Z"
															/>
														</svg>
													{:else if chip.iconType === "check"}
														<svg
															class="w-3.5 h-3.5"
															viewBox="0 0 24 24"
															fill="none"
															stroke="currentColor"
															stroke-width="2"
															stroke-linecap="round"
															stroke-linejoin="round"
														>
															<path d="m4.5 12.75 4.5 4.5 10.5-10.5" />
														</svg>
													{:else if chip.iconType === "mail"}
														<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor">
															<path
																d={mailOpenIcon.path}
																fill-rule={mailOpenIcon.fillRule}
																clip-rule={mailOpenIcon.clipRule}
																stroke="currentColor"
																stroke-width=".3"
																stroke-linecap="round"
																stroke-linejoin="round"
															/>
														</svg>
													{:else if chip.iconType === "archive"}
														<svg
															class="w-3.5 h-3.5"
															viewBox="0 0 24 24"
															fill="none"
															stroke="currentColor"
															stroke-width="2"
															stroke-linecap="round"
															stroke-linejoin="round"
														>
															{#each archiveIconPaths as path (path)}
																<path d={path} />
															{/each}
														</svg>
													{:else if chip.iconType === "mute"}
														<svg
															class="w-3.5 h-3.5"
															viewBox="0 0 24 24"
															fill="none"
															stroke="currentColor"
															stroke-width="2"
															stroke-linecap="round"
															stroke-linejoin="round"
														>
															<path d={bellOffIconPath} />
														</svg>
													{:else if chip.iconType === "tag"}
														<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor">
															<path
																d="M17.63 5.84C17.27 5.33 16.67 5 16 5L5 5.01C3.9 5.01 3 5.9 3 7v10c0 1.1.9 1.99 2 1.99L16 19c.67 0 1.27-.33 1.63-.84L22 12l-4.37-6.16z"
															/>
														</svg>
													{/if}
													{chip.label}
												</span>
											{/each}
										{:else}
											<span class="text-xs text-gray-600 dark:text-gray-500 italic"
												>No actions configured</span
											>
										{/if}
									</div>
								</div>
							</div>

							<!-- Edit/Delete buttons -->
							<div class="flex items-center justify-end gap-2 pt-1">
								<button
									on:click|stopPropagation={() => handleEditRule(rule)}
									class="px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors cursor-pointer"
								>
									Edit
								</button>
								<button
									on:click|stopPropagation={() => {
										editingRule = rule;
										requestRuleDelete();
									}}
									class="px-3 py-1.5 text-sm font-medium text-red-400 hover:text-red-300 hover:bg-red-950/30 rounded-lg transition-colors cursor-pointer"
								>
									Delete
								</button>
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Reorder controls -->
		<div class="flex items-center justify-between mt-4">
			<div class="text-xs text-gray-500 dark:text-gray-500">
				{sortedRules.length} rule{sortedRules.length !== 1 ? "s" : ""}
			</div>
			<div class="flex gap-2">
				{#if $dragAndDropReorderMode}
					<button
						on:click={cancelReorder}
						disabled={$dragAndDropReordering}
						class="px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 disabled:opacity-50 cursor-pointer"
					>
						Cancel
					</button>
					<button
						on:click={toggleReorderMode}
						disabled={$dragAndDropReordering}
						class="px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 disabled:opacity-50 cursor-pointer"
					>
						{$dragAndDropReordering ? "Saving..." : "Done"}
					</button>
				{:else}
					<button
						on:click={toggleReorderMode}
						class="px-1 py-1 text-xs font-semibold text-gray-500 dark:text-gray-500 transition hover:text-gray-800 dark:hover:text-gray-300 cursor-pointer"
					>
						Reorder
					</button>
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Rule Dialog -->
<RuleDialog
	bind:open={showRuleDialog}
	rule={editingRule}
	onDelete={editingRule ? requestRuleDelete : null}
/>

<ConfirmDialog
	open={ruleDeleteConfirmOpen}
	title="Delete rule"
	body="Are you sure you want to delete this rule? This action cannot be undone."
	confirmLabel="Delete"
	cancelLabel="Cancel"
	confirmTone="danger"
	confirming={ruleDeleting}
	onCancel={cancelRuleDelete}
	onConfirm={confirmRuleDelete}
/>
