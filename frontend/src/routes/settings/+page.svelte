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

	import { onMount, onDestroy, getContext } from "svelte";
	import { browser } from "$app/environment";
	import type { PageData } from "./$types";
	import SettingsView from "$lib/components/settings/SettingsView.svelte";
	import GitHubSettingsSection from "$lib/components/settings/GitHubSettingsSection.svelte";
	import RulesSection from "$lib/components/settings/RulesSection.svelte";
	import NotificationSettingsSection from "$lib/components/settings/NotificationSettingsSection.svelte";
	import ThemeSettingsSection from "$lib/components/settings/ThemeSettingsSection.svelte";
	import SyncOlderNotificationsSection from "$lib/components/settings/SyncOlderNotificationsSection.svelte";
	import StorageSettingsSection from "$lib/components/settings/StorageSettingsSection.svelte";
	import UpdateSettingsSection from "$lib/components/settings/UpdateSettingsSection.svelte";
	import { registerListShortcuts } from "$lib/keyboard/listShortcuts";
	import { registerCommand } from "$lib/keyboard/commandRegistry";

	export let data: PageData;

	// Valid section IDs
	const validSections = ["account", "appearance", "notifications", "rules", "data", "updates"];

	// Active section state
	let activeSection = "account";

	// Get layout context to access command palette
	const layoutContext = getContext("layoutFunctions") as any;

	// Parse hash from URL
	function getHashSection(): string {
		if (!browser) return "account";
		const hash = window.location.hash.slice(1); // Remove the #
		return validSections.includes(hash) ? hash : "account";
	}

	// Handle hash changes
	function handleHashChange() {
		activeSection = getHashSection();
	}

	// Command palette handlers for shortcuts
	function openCommandPaletteForView(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.openWithCommand("view");
		return true;
	}

	function openCommandPaletteEmpty(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.focusInput();
		return true;
	}

	function toggleShortcutsModal(): boolean {
		return layoutContext?.toggleShortcutsModal() ?? false;
	}

	function isAnyDialogOpen(): boolean {
		return layoutContext?.isAnyDialogOpen() ?? false;
	}

	let unregisterShortcuts: (() => void) | null = null;

	onMount(() => {
		// Initialize section from hash
		activeSection = getHashSection();

		// Listen for hash changes
		window.addEventListener("hashchange", handleHashChange);

		// Register commands for shortcuts
		registerCommand("openPaletteView", () => openCommandPaletteForView());
		registerCommand("openPaletteEmpty", () => openCommandPaletteEmpty());
		registerCommand("toggleShortcutsModal", () => toggleShortcutsModal());

		// Register shortcuts - only Cmd+K and Cmd+Shift+K are needed for settings
		unregisterShortcuts = registerListShortcuts({
			getDetailOpen: () => false,
			isAnyDialogOpen,
			isFilterDropdownOpen: () => false,
		});

		return () => {
			window.removeEventListener("hashchange", handleHashChange);
			if (unregisterShortcuts) {
				unregisterShortcuts();
			}
		};
	});

	onDestroy(() => {
		if (browser) {
			window.removeEventListener("hashchange", handleHashChange);
		}
		if (unregisterShortcuts) {
			unregisterShortcuts();
		}
	});

	// Section titles and descriptions
	const sectionMeta: Record<string, { title: string; description: string }> = {
		account: {
			title: "Account",
			description: "Manage your GitHub connection",
		},
		appearance: {
			title: "Appearance",
			description: "Customize how Octobud looks",
		},
		notifications: {
			title: "Notifications",
			description: "Configure desktop notification preferences",
		},
		rules: {
			title: "Rules",
			description: "Automatically organize notifications",
		},
		data: {
			title: "Data",
			description: "Manage sync and storage settings",
		},
		updates: {
			title: "Updates",
			description: "Configure automatic update checking",
		},
	};

	$: currentMeta = sectionMeta[activeSection] || sectionMeta.account;
</script>

<SettingsView {activeSection}>
	<!-- Section Header -->
	<div class="mb-8">
		<h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">
			{currentMeta.title}
		</h1>
		<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
			{currentMeta.description}
		</p>
	</div>

	<!-- Section Content -->
	{#if activeSection === "account"}
		<GitHubSettingsSection />
	{:else if activeSection === "appearance"}
		<ThemeSettingsSection />
	{:else if activeSection === "notifications"}
		<NotificationSettingsSection />
	{:else if activeSection === "rules"}
		<RulesSection rules={data.rules} tags={data.tags} />
	{:else if activeSection === "data"}
		<div class="space-y-8">
			<SyncOlderNotificationsSection />
			<div class="border-t border-gray-200 dark:border-gray-800 pt-8">
				<StorageSettingsSection />
			</div>
		</div>
	{:else if activeSection === "updates"}
		<UpdateSettingsSection />
	{/if}
</SettingsView>
