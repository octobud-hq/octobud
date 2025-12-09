<!-- Copyright (C) 2025 Austin Beattie

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A GENERAL PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. -->

<script lang="ts">
	import { slide } from "svelte/transition";
	import { getUpdateStore } from "$lib/stores/updateStore";
	import { browser } from "$app/environment";

	const updateStore = getUpdateStore();
	
	// Access individual stores from the updateStore object
	const { updateInfo, updateAvailable, settings } = updateStore;
	
	// Check if banner should be shown (update available and not dismissed)
	$: showBanner = $updateAvailable && $updateInfo !== null;
	
	// Check if dismissed
	$: currentSettings = $settings;
	$: dismissedUntil = currentSettings?.dismissedUntil;
	$: isDismissed = dismissedUntil 
		? new Date(dismissedUntil) > new Date() 
		: false;
	
	$: shouldShow = showBanner && !isDismissed;
	$: currentUpdateInfo = $updateInfo;

	let isDownloading = false;

	async function handleDownload() {
		const info = $updateInfo;
		if (!info?.downloadUrl) {
			return;
		}

		isDownloading = true;
		try {
			// Create a temporary anchor element to trigger download
			// This ensures the download starts directly without opening a new tab
			if (browser) {
				const link = document.createElement("a");
				link.href = info.downloadUrl;
				link.download = info.assetName || "Octobud-update.pkg";
				link.target = "_blank";
				document.body.appendChild(link);
				link.click();
				document.body.removeChild(link);
				
				// Dismiss the banner after starting download
				await updateStore.dismissUpdate(24); // Dismiss for 24 hours
			}
		} catch (err) {
			console.error("Failed to download update:", err);
		} finally {
			isDownloading = false;
		}
	}

	async function handleDismiss() {
		await updateStore.dismissUpdate(24); // Dismiss for 24 hours
	}
</script>

{#if shouldShow && currentUpdateInfo}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-blue-200 bg-blue-50 px-4 py-3 dark:border-blue-800 dark:bg-blue-950/50"
		role="status"
		aria-live="polite"
	>
		<div class="flex items-center gap-3 flex-1">
			<!-- Update icon -->
			<svg
				class="h-5 w-5 flex-shrink-0 text-blue-900 dark:text-blue-100"
				fill="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					d="M12.5535 16.5061C12.4114 16.6615 12.2106 16.75 12 16.75C11.7894 16.75 11.5886 16.6615 11.4465 16.5061L7.44648 12.1311C7.16698 11.8254 7.18822 11.351 7.49392 11.0715C7.79963 10.792 8.27402 10.8132 8.55352 11.1189L11.25 14.0682V3C11.25 2.58579 11.5858 2.25 12 2.25C12.4142 2.25 12.75 2.58579 12.75 3V14.0682L15.4465 11.1189C15.726 10.8132 16.2004 10.792 16.5061 11.0715C16.8118 11.351 16.833 11.8254 16.5535 12.1311L12.5535 16.5061Z"
				/>
				<path
					d="M3.75 15C3.75 14.5858 3.41422 14.25 3 14.25C2.58579 14.25 2.25 14.5858 2.25 15V15.0549C2.24998 16.4225 2.24996 17.5248 2.36652 18.3918C2.48754 19.2919 2.74643 20.0497 3.34835 20.6516C3.95027 21.2536 4.70814 21.5125 5.60825 21.6335C6.47522 21.75 7.57754 21.75 8.94513 21.75H15.0549C16.4225 21.75 17.5248 21.75 18.3918 21.6335C19.2919 21.5125 20.0497 21.2536 20.6517 20.6516C21.2536 20.0497 21.5125 19.2919 21.6335 18.3918C21.75 17.5248 21.75 16.4225 21.75 15.0549V15C21.75 14.5858 21.4142 14.25 21 14.25C20.5858 14.25 20.25 14.5858 20.25 15C20.25 16.4354 20.2484 17.4365 20.1469 18.1919C20.0482 18.9257 19.8678 19.3142 19.591 19.591C19.3142 19.8678 18.9257 20.0482 18.1919 20.1469C17.4365 20.2484 16.4354 20.25 15 20.25H9C7.56459 20.25 6.56347 20.2484 5.80812 20.1469C5.07435 20.0482 4.68577 19.8678 4.40901 19.591C4.13225 19.3142 3.9518 18.9257 3.85315 18.1919C3.75159 17.4365 3.75 16.4354 3.75 15Z"
				/>
			</svg>

			<div class="flex-1">
				<p class="text-sm font-medium text-blue-900 dark:text-blue-100">
					Update available: {currentUpdateInfo.latestVersion}
				</p>
				{#if currentUpdateInfo.currentVersion}
					<p class="text-xs text-blue-700 dark:text-blue-300 mt-0.5">
						You're currently on {currentUpdateInfo.currentVersion}
					</p>
				{/if}
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				type="button"
				on:click={handleDownload}
				disabled={isDownloading}
				class="inline-flex items-center gap-2 flex-shrink-0 rounded-full bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-600 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-indigo-500 dark:hover:bg-indigo-600 dark:focus:ring-offset-gray-950"
			>
				{#if isDownloading}
					<svg
						class="h-3 w-3 animate-spin"
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
					>
						<circle
							class="opacity-25"
							cx="12"
							cy="12"
							r="10"
							stroke="currentColor"
							stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
				{:else}
					<svg
						class="h-3 w-3"
						fill="currentColor"
						viewBox="0 0 24 24"
						xmlns="http://www.w3.org/2000/svg"
					>
						<path
							d="M12.5535 16.5061C12.4114 16.6615 12.2106 16.75 12 16.75C11.7894 16.75 11.5886 16.6615 11.4465 16.5061L7.44648 12.1311C7.16698 11.8254 7.18822 11.351 7.49392 11.0715C7.79963 10.792 8.27402 10.8132 8.55352 11.1189L11.25 14.0682V3C11.25 2.58579 11.5858 2.25 12 2.25C12.4142 2.25 12.75 2.58579 12.75 3V14.0682L15.4465 11.1189C15.726 10.8132 16.2004 10.792 16.5061 11.0715C16.8118 11.351 16.833 11.8254 16.5535 12.1311L12.5535 16.5061Z"
						/>
						<path
							d="M3.75 15C3.75 14.5858 3.41422 14.25 3 14.25C2.58579 14.25 2.25 14.5858 2.25 15V15.0549C2.24998 16.4225 2.24996 17.5248 2.36652 18.3918C2.48754 19.2919 2.74643 20.0497 3.34835 20.6516C3.95027 21.2536 4.70814 21.5125 5.60825 21.6335C6.47522 21.75 7.57754 21.75 8.94513 21.75H15.0549C16.4225 21.75 17.5248 21.75 18.3918 21.6335C19.2919 21.5125 20.0497 21.2536 20.6517 20.6516C21.2536 20.0497 21.5125 19.2919 21.6335 18.3918C21.75 17.5248 21.75 16.4225 21.75 15.0549V15C21.75 14.5858 21.4142 14.25 21 14.25C20.5858 14.25 20.25 14.5858 20.25 15C20.25 16.4354 20.2484 17.4365 20.1469 18.1919C20.0482 18.9257 19.8678 19.3142 19.591 19.591C19.3142 19.8678 18.9257 20.0482 18.1919 20.1469C17.4365 20.2484 16.4354 20.25 15 20.25H9C7.56459 20.25 6.56347 20.2484 5.80812 20.1469C5.07435 20.0482 4.68577 19.8678 4.40901 19.591C4.13225 19.3142 3.9518 18.9257 3.85315 18.1919C3.75159 17.4365 3.75 16.4354 3.75 15Z"
						/>
					</svg>
				{/if}
				{isDownloading ? "Downloading..." : "Download"}
			</button>
			<button
				type="button"
				on:click={handleDismiss}
				class="flex-shrink-0 rounded-md p-1 text-blue-700 transition hover:bg-blue-100 focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2 dark:text-blue-300 dark:hover:bg-blue-900/50 dark:focus:ring-offset-gray-950"
				aria-label="Dismiss"
			>
				<svg
					class="h-5 w-5"
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<line x1="18" y1="6" x2="6" y2="18"></line>
					<line x1="6" y1="6" x2="18" y2="18"></line>
				</svg>
			</button>
		</div>
	</div>
{/if}

