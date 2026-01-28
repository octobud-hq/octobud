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

	import { onMount, onDestroy } from "svelte";
	import { getNotificationSettingsStore } from "$lib/stores/notificationSettings";
	import {
		requestNotificationPermission,
		getNotificationPermission,
		sendNotificationSettingToSW,
	} from "$lib/utils/serviceWorkerRegistration";
	import { toastStore } from "$lib/stores/toastStore";
	import { browser } from "$app/environment";
	import { getMuteStatus, setMute, clearMute, type MuteStatus } from "$lib/api/user";

	const settingsStore = getNotificationSettingsStore();
	const { notificationsEnabled, faviconBadgeEnabled } = settingsStore;

	let isSupported = false;
	let permission: NotificationPermission = "default";
	let isToggling = false;
	let muteStatus: MuteStatus | null = null;
	let isMuteLoading = false;
	let isMuteDropdownOpen = false;
	let dropdownElement: HTMLDivElement;
	let buttonElement: HTMLButtonElement;

	onMount(() => {
		if (browser) {
			isSupported = "Notification" in window && "serviceWorker" in navigator;
			permission = getNotificationPermission();
			loadMuteStatus();
		}
	});

	async function loadMuteStatus() {
		try {
			muteStatus = await getMuteStatus();
		} catch (error) {
			console.error("Failed to load mute status:", error);
		}
	}

	async function handleSetMute(duration: "30m" | "1h" | "rest_of_day") {
		if (isMuteLoading) return;

		isMuteLoading = true;
		try {
			muteStatus = await setMute(duration);
			isMuteDropdownOpen = false;
			const durationText =
				duration === "30m" ? "30 minutes" : duration === "1h" ? "1 hour" : "rest of day";
			toastStore.success(`Notifications muted for ${durationText}`);
		} catch (error) {
			console.error("Failed to set mute:", error);
			toastStore.error("Failed to mute notifications");
		} finally {
			isMuteLoading = false;
		}
	}

	async function handleClearMute() {
		if (isMuteLoading) return;

		isMuteLoading = true;
		try {
			muteStatus = await clearMute();
			toastStore.success("Notifications unmuted");
		} catch (error) {
			console.error("Failed to clear mute:", error);
			toastStore.error("Failed to unmute notifications");
		} finally {
			isMuteLoading = false;
		}
	}

	function formatMuteTime(mutedUntil: string | null): string {
		if (!mutedUntil) return "";
		const date = new Date(mutedUntil);
		const now = new Date();
		const diffMs = date.getTime() - now.getTime();
		const diffMins = Math.floor(diffMs / 60000);

		if (diffMins < 1) return "Less than a minute";
		if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? "" : "s"}`;
		const hours = Math.floor(diffMins / 60);
		const mins = diffMins % 60;
		if (mins === 0) return `${hours} hour${hours === 1 ? "" : "s"}`;
		return `${hours}h ${mins}m`;
	}

	function handleClickOutside(event: MouseEvent) {
		if (
			dropdownElement &&
			!dropdownElement.contains(event.target as Node) &&
			buttonElement &&
			!buttonElement.contains(event.target as Node)
		) {
			isMuteDropdownOpen = false;
		}
	}

	onMount(() => {
		if (browser) {
			document.addEventListener("click", handleClickOutside);
		}
	});

	onDestroy(() => {
		if (browser) {
			document.removeEventListener("click", handleClickOutside);
		}
	});

	async function handleToggle(enabled: boolean) {
		if (isToggling) return;

		isToggling = true;

		try {
			if (enabled) {
				// Turning notifications ON - request permission if needed
				const currentPermission = getNotificationPermission();

				if (currentPermission === "denied") {
					toastStore.error(
						"Notification permission is denied. Please enable it in your browser settings."
					);
					// Don't enable the toggle if permission is denied
					settingsStore.setEnabled(false);
					return;
				}

				if (currentPermission === "default") {
					// Request permission
					const newPermission = await requestNotificationPermission();
					permission = newPermission; // Update local state

					if (newPermission !== "granted") {
						toastStore.error(
							"Notification permission is required to enable desktop notifications."
						);
						settingsStore.setEnabled(false);
						return;
					}

					toastStore.success("Desktop notifications enabled");
				} else {
					// Permission already granted
					permission = currentPermission; // Update local state
					toastStore.success("Desktop notifications enabled");
				}

				// Set enabled and notify service worker
				settingsStore.setEnabled(true);
				await sendNotificationSettingToSW(true);
			} else {
				// Turning notifications OFF
				settingsStore.setEnabled(false);
				await sendNotificationSettingToSW(false);
				toastStore.success("Desktop notifications disabled");
			}
		} catch (error) {
			console.error("Failed to toggle notifications:", error);
			toastStore.error("Failed to update notification settings");
		} finally {
			isToggling = false;
		}
	}

	$: canEnable = permission !== "denied";

	function handleFaviconBadgeToggle(enabled: boolean) {
		settingsStore.setFaviconBadgeEnabled(enabled);
		if (enabled) {
			toastStore.success("Unread count badge in tab header enabled");
		} else {
			toastStore.success("Unread count badge in tab header disabled");
		}
	}
</script>

<div class="space-y-4">
	<!-- Notification Toggle -->
	<div class="flex items-center justify-between py-3">
		<div class="flex-1">
			<div class="flex items-center gap-2">
				<span class="text-md font-medium text-gray-900 dark:text-gray-100">
					Enable Desktop Notifications
				</span>
				{#if !isSupported}
					<span class="text-xs text-gray-600 dark:text-gray-500"
						>(Not supported in this browser)</span
					>
				{/if}
			</div>
			<p class="text-xs text-gray-600 dark:text-gray-400 mt-1">
				Notifications are shown for any new items in the inbox. Rules that filter, archive, or mute
				will also prevent desktop notifications.
				{#if permission === "denied"}
					<br />
					<span class="text-amber-600 dark:text-amber-500">
						Permission is denied. Enable notifications in your browser settings.
					</span>
				{/if}
			</p>
		</div>
		<label
			class="relative flex items-center cursor-pointer {!isSupported || isToggling
				? 'opacity-50 cursor-not-allowed'
				: ''}"
		>
			<input
				type="checkbox"
				checked={$notificationsEnabled}
				disabled={!isSupported || isToggling || !canEnable}
				on:change={(e) => handleToggle(e.currentTarget.checked)}
				class="sr-only peer"
			/>
			<div
				class="relative w-9 h-5 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"
			></div>
		</label>
	</div>

	<!-- Favicon Badge Toggle -->
	<div class="flex items-center justify-between py-3 border-t border-gray-200 dark:border-gray-800">
		<div class="flex-1">
			<div class="flex items-center gap-2">
				<span class="text-md font-medium text-gray-900 dark:text-gray-100">
					Show unread notification count in tab header
				</span>
			</div>
			<p class="text-xs text-gray-600 dark:text-gray-400 mt-1">
				Display a badge on the browser tab favicon showing the number of unread notifications.
			</p>
		</div>
		<label class="relative flex items-center cursor-pointer">
			<input
				type="checkbox"
				checked={$faviconBadgeEnabled}
				on:change={(e) => handleFaviconBadgeToggle(e.currentTarget.checked)}
				class="sr-only peer"
			/>
			<div
				class="relative w-9 h-5 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"
			></div>
		</label>
	</div>

	<!-- Mute Notifications -->
	<div class="space-y-3 pt-4 border-t border-gray-200 dark:border-gray-800">
		<div class="flex items-center justify-between">
			<div class="flex-1">
				<div class="flex items-center gap-2">
					<span class="text-md font-medium text-gray-900 dark:text-gray-100">
						Mute Notifications
					</span>
				</div>
				<p class="text-xs text-gray-600 dark:text-gray-400 mt-1">
					Temporarily mute all desktop notifications. Notifications will resume automatically when
					the mute period expires.
				</p>
			</div>
		</div>

		{#if muteStatus}
			{#if muteStatus.isMuted && muteStatus.mutedUntil}
				<div
					class="flex items-center justify-between p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg"
				>
					<div class="flex-1">
						<div class="text-sm font-medium text-amber-900 dark:text-amber-200">
							Notifications muted
						</div>
						<div class="text-xs text-amber-700 dark:text-amber-300 mt-1">
							Until: {formatMuteTime(muteStatus.mutedUntil)} remaining
						</div>
					</div>
					<button
						type="button"
						disabled={isMuteLoading}
						on:click={handleClearMute}
						class="px-3 py-1.5 text-sm font-medium text-amber-900 dark:text-amber-200 bg-amber-100 dark:bg-amber-900/40 hover:bg-amber-200 dark:hover:bg-amber-900/60 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						Unmute
					</button>
				</div>
			{:else}
				<div class="relative">
					<button
						type="button"
						bind:this={buttonElement}
						disabled={isMuteLoading}
						on:click={() => (isMuteDropdownOpen = !isMuteDropdownOpen)}
						class="w-full px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-between"
					>
						<span>Mute for...</span>
						<svg
							class="w-4 h-4 text-gray-600 dark:text-gray-400 transition-transform {isMuteDropdownOpen
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
					</button>

					{#if isMuteDropdownOpen}
						<div
							bind:this={dropdownElement}
							class="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg"
							role="menu"
						>
							<button
								type="button"
								disabled={isMuteLoading}
								on:click={() => handleSetMute("30m")}
								class="w-full px-4 py-2 text-left text-sm text-gray-900 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
								role="menuitem"
							>
								30 minutes
							</button>
							<button
								type="button"
								disabled={isMuteLoading}
								on:click={() => handleSetMute("1h")}
								class="w-full px-4 py-2 text-left text-sm text-gray-900 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
								role="menuitem"
							>
								1 hour
							</button>
							<button
								type="button"
								disabled={isMuteLoading}
								on:click={() => handleSetMute("rest_of_day")}
								class="w-full px-4 py-2 text-left text-sm text-gray-900 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
								role="menuitem"
							>
								Rest of day
							</button>
						</div>
					{/if}
				</div>
			{/if}
		{/if}
	</div>
</div>
