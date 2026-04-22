<!-- Copyright (C) 2025 Austin Beattie

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. -->

<script lang="ts">
	import { onMount } from "svelte";
	import { slide } from "svelte/transition";
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import { getVersion } from "$lib/api/user";

	const STORAGE_KEY = "octobud:whats-new-last-seen";

	let show = false;
	let currentVersion: string | null = null;

	// Compare two semver-like version strings.
	// Returns > 0 if a > b, < 0 if a < b, 0 if equal, NaN if either side doesn't parse.
	// Handles an optional leading "v" and prerelease suffixes (e.g. "0.4.0-beta.1").
	// Non-semver inputs (e.g. "dev") return NaN so callers can choose a fallback.
	function semverCompare(a: string, b: string): number {
		const parse = (v: string) => {
			const m = v.replace(/^v/, "").match(/^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$/);
			if (!m) return null;
			return {
				major: Number(m[1]),
				minor: Number(m[2]),
				patch: Number(m[3]),
				prerelease: m[4] ?? "",
			};
		};

		const pa = parse(a);
		const pb = parse(b);
		if (!pa || !pb) return NaN;

		if (pa.major !== pb.major) return pa.major - pb.major;
		if (pa.minor !== pb.minor) return pa.minor - pb.minor;
		if (pa.patch !== pb.patch) return pa.patch - pb.patch;

		// Per semver spec: a release version outranks a prerelease of the same X.Y.Z.
		if (pa.prerelease === pb.prerelease) return 0;
		if (!pa.prerelease) return 1;
		if (!pb.prerelease) return -1;
		return pa.prerelease < pb.prerelease ? -1 : 1;
	}

	function markSeen(version: string) {
		if (!browser) return;
		try {
			localStorage.setItem(STORAGE_KEY, version);
		} catch {
			// localStorage may be unavailable (e.g. private mode) — fail quietly
		}
	}

	function handleViewChangelog() {
		if (currentVersion) markSeen(currentVersion);
		show = false;
		void goto("/settings?scroll=release-notes#updates");
	}

	function handleDismiss() {
		if (currentVersion) markSeen(currentVersion);
		show = false;
	}

	onMount(async () => {
		if (!browser) return;
		try {
			const res = await getVersion();
			if (!res?.version) {
				console.warn("WhatsNewBanner: getVersion() returned no version, skipping banner");
				return;
			}
			currentVersion = res.version.replace(/^v/, "");

			const lastSeen = localStorage.getItem(STORAGE_KEY);

			if (!lastSeen) {
				// No baseline recorded yet — either a fresh install or (more commonly for
				// this release) an existing user upgrading from a version that predates this
				// banner. Show the banner so upgraders aren't silently skipped. Future
				// releases will compare against the baseline written by markSeen below.
				show = true;
				return;
			}

			const cmp = semverCompare(currentVersion, lastSeen);
			if (Number.isNaN(cmp)) {
				// Either side isn't valid semver (e.g. a "dev" build). Fall back to strict
				// inequality — we can't guard against downgrades here, but we also don't
				// want to silently skip the banner for real users on non-semver builds.
				if (currentVersion !== lastSeen) show = true;
			} else if (cmp > 0) {
				// Only show when current is strictly newer than what the user last saw.
				show = true;
			}
		} catch (err) {
			console.error("Failed to check what's-new banner state:", err);
		}
	});
</script>

{#if show && currentVersion}
	<div
		transition:slide={{ duration: 300 }}
		class="flex items-center justify-between gap-3 border-b border-green-200 bg-green-50 px-4 py-3 dark:border-green-800 dark:bg-green-950/50"
		role="status"
		aria-live="polite"
	>
		<div class="flex items-center gap-3 flex-1">
			<svg
				class="h-5 w-5 flex-shrink-0 text-green-900 dark:text-green-100"
				xmlns="http://www.w3.org/2000/svg"
				viewBox="0 0 24 24"
				fill="currentColor"
				aria-hidden="true"
			>
				<path
					d="M9 4.5a.75.75 0 0 1 .721.544l.813 2.846a3.75 3.75 0 0 0 2.576 2.576l2.846.813a.75.75 0 0 1 0 1.442l-2.846.813a3.75 3.75 0 0 0-2.576 2.576l-.813 2.846a.75.75 0 0 1-1.442 0l-.813-2.846a3.75 3.75 0 0 0-2.576-2.576l-2.846-.813a.75.75 0 0 1 0-1.442l2.846-.813A3.75 3.75 0 0 0 7.466 7.89l.813-2.846A.75.75 0 0 1 9 4.5ZM18 1.5a.75.75 0 0 1 .728.568l.258 1.036c.236.94.97 1.674 1.91 1.91l1.036.258a.75.75 0 0 1 0 1.456l-1.036.258a2.625 2.625 0 0 0-1.91 1.91l-.258 1.036a.75.75 0 0 1-1.456 0l-.258-1.036a2.625 2.625 0 0 0-1.91-1.91l-1.036-.258a.75.75 0 0 1 0-1.456l1.036-.258a2.625 2.625 0 0 0 1.91-1.91l.258-1.036A.75.75 0 0 1 18 1.5ZM16.5 15a.75.75 0 0 1 .712.513l.394 1.183c.15.447.5.799.948.948l1.183.395a.75.75 0 0 1 0 1.422l-1.183.395a1.5 1.5 0 0 0-.948.948l-.395 1.183a.75.75 0 0 1-1.422 0l-.395-1.183a1.5 1.5 0 0 0-.948-.948l-1.183-.395a.75.75 0 0 1 0-1.422l1.183-.395a1.5 1.5 0 0 0 .948-.948l.395-1.183A.75.75 0 0 1 16.5 15Z"
				/>
			</svg>

			<div class="flex-1">
				<p class="text-sm font-medium text-green-900 dark:text-green-100">
					Octobud updated to {currentVersion}
				</p>
				<p class="text-xs text-green-700 dark:text-green-300 mt-0.5">
					See what's new in this release.
				</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				type="button"
				on:click={handleViewChangelog}
				class="inline-flex items-center gap-2 flex-shrink-0 rounded-full bg-green-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-600 focus:ring-offset-2 dark:bg-green-500 dark:hover:bg-green-600 dark:focus:ring-offset-gray-950"
			>
				View changelog
			</button>
			<button
				type="button"
				on:click={handleDismiss}
				class="flex-shrink-0 rounded-md p-1 text-green-700 transition hover:bg-green-100 focus:outline-none focus:ring-2 focus:ring-green-600 focus:ring-offset-2 dark:text-green-300 dark:hover:bg-green-900/50 dark:focus:ring-offset-gray-950"
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
