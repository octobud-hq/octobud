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
	import { renderMarkdown } from "$lib/utils/markdown";

	type VersionSection = {
		version: string;
		date?: string;
		bodyHtml: string;
	};

	let sections: VersionSection[] = [];
	let error: string | null = null;
	let isLoading = true;

	// Matches "## [0.3.2]", "## 0.3.2", "## [0.1.3] - 2025-12-09", or "## 0.1.3 - 2025-12-09".
	// Brackets are optional so the parser tolerates format drift in either direction.
	const VERSION_HEADER = /^##\s+\[?([^\]\s]+)\]?(?:\s+-\s+(\S+))?\s*$/gm;

	function parseChangelog(md: string): VersionSection[] {
		const out: VersionSection[] = [];
		const matches = [...md.matchAll(VERSION_HEADER)];
		for (let i = 0; i < matches.length; i++) {
			const m = matches[i];
			const start = (m.index ?? 0) + m[0].length;
			const end = i + 1 < matches.length ? (matches[i + 1].index ?? md.length) : md.length;
			const body = md.slice(start, end).trim();
			out.push({
				version: m[1],
				date: m[2],
				bodyHtml: renderMarkdown(body),
			});
		}

		// Noisy fallback: if the file has `## ` headings but we parsed zero versions,
		// the format has drifted in a way the parser doesn't handle. Fail loud in the
		// console so we notice in CI/dev without breaking the UI for users.
		if (out.length === 0 && /^##\s/m.test(md)) {
			console.warn(
				"ChangelogView: found ## headings but parsed 0 version sections. " +
					"Expected `## [X.Y.Z]` or `## X.Y.Z` — check CHANGELOG.md format."
			);
		}

		return out;
	}

	onMount(async () => {
		try {
			const res = await fetch("/CHANGELOG.md");
			if (!res.ok) {
				throw new Error(`Failed to load changelog (${res.status})`);
			}
			const text = await res.text();
			sections = parseChangelog(text);
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to load changelog";
		} finally {
			isLoading = false;
		}
	});
</script>

{#if isLoading}
	<div class="text-sm text-gray-500 dark:text-gray-400">Loading release notes...</div>
{:else if error}
	<div class="rounded-md bg-red-50 p-3 text-sm text-red-800 dark:bg-red-950/50 dark:text-red-200">
		{error}
	</div>
{:else if sections.length === 0}
	<div class="text-sm text-gray-500 dark:text-gray-400">No release notes available.</div>
{:else}
	<div class="space-y-1">
		{#each sections as section, i (section.version)}
			<details
				class="changelog-version group border-b border-gray-200 py-3 last:border-b-0 dark:border-gray-800"
				open={i === 0}
			>
				<summary
					class="flex cursor-pointer list-none items-center gap-2 text-gray-900 outline-none dark:text-gray-100"
				>
					<svg
						class="h-4 w-4 flex-shrink-0 text-gray-400 transition-transform duration-150 group-open:rotate-90 dark:text-gray-500"
						viewBox="0 0 20 20"
						fill="currentColor"
						aria-hidden="true"
					>
						<path
							fill-rule="evenodd"
							d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
							clip-rule="evenodd"
						/>
					</svg>
					<span class="text-base font-semibold">{section.version}</span>
					{#if section.date}
						<span class="text-xs font-normal text-gray-500 dark:text-gray-400"
							>— {section.date}</span
						>
					{/if}
				</summary>
				<div
					class="prose prose-sm prose-link-hover mt-2 ml-6 max-w-none dark:prose-invert prose-code:rounded prose-code:bg-gray-100 prose-code:px-1.5 prose-code:py-0.5 prose-code:text-gray-800 prose-code:before:content-[''] prose-code:after:content-[''] dark:prose-code:bg-gray-900 dark:prose-code:text-gray-300"
				>
					{@html section.bodyHtml}
				</div>
			</details>
		{/each}
	</div>
{/if}

<style>
	/* Hide the default <details> marker across browsers */
	.changelog-version > summary::-webkit-details-marker {
		display: none;
	}
	.changelog-version > summary::marker {
		content: "";
	}
</style>
