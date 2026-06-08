<!--
Copyright (C) 2025 Austin Beattie

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
	import { onDestroy, onMount } from "svelte";
	import {
		fetchDiagnosticsInfo,
		fetchGitHubStatus,
		fetchLogs,
		fetchDiagnosticsBundle,
		formatBundleAsMarkdown,
		logsDownloadUrl,
		type DiagnosticsInfo,
		type GitHubStatus,
		type LogEntry,
	} from "$lib/api/diagnostics";
	import { toastStore } from "$lib/stores/toastStore";

	let info: DiagnosticsInfo | null = null;
	let githubStatus: GitHubStatus | null = null;
	let logs: LogEntry[] = [];
	let logsTruncated = false;
	let infoError = "";
	let githubError = "";
	let logsError = "";

	let levelFilter = "";
	let searchFilter = "";
	let autoRefresh = false;
	let isCopying = false;
	let lastLogsLoadedAt: Date | null = null;

	const LOG_LIMIT = 200;
	const AUTO_REFRESH_INTERVAL_MS = 5000;

	let autoRefreshTimer: ReturnType<typeof setInterval> | null = null;
	let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	onMount(async () => {
		await Promise.all([loadInfo(), loadGitHubStatus(), loadLogs()]);
	});

	onDestroy(() => {
		if (autoRefreshTimer) clearInterval(autoRefreshTimer);
		if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
	});

	async function loadInfo() {
		infoError = "";
		try {
			info = await fetchDiagnosticsInfo();
		} catch (err) {
			infoError = err instanceof Error ? err.message : "Failed to load info";
		}
	}

	async function loadGitHubStatus() {
		githubError = "";
		try {
			githubStatus = await fetchGitHubStatus();
		} catch (err) {
			githubError = err instanceof Error ? err.message : "Failed to load GitHub status";
		}
	}

	async function loadLogs() {
		logsError = "";
		try {
			const result = await fetchLogs({
				level: levelFilter || undefined,
				search: searchFilter || undefined,
				limit: LOG_LIMIT,
			});
			logs = result.entries;
			logsTruncated = result.truncated;
			lastLogsLoadedAt = new Date();
		} catch (err) {
			logsError = err instanceof Error ? err.message : "Failed to load logs";
		}
	}

	function handleLevelChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		levelFilter = target.value;
		void loadLogs();
	}

	function handleSearchInput(event: Event) {
		const target = event.target as HTMLInputElement;
		searchFilter = target.value;
		if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
		searchDebounceTimer = setTimeout(() => {
			void loadLogs();
		}, 300);
	}

	function toggleAutoRefresh() {
		autoRefresh = !autoRefresh;
		if (autoRefresh) {
			autoRefreshTimer = setInterval(() => {
				void loadLogs();
			}, AUTO_REFRESH_INTERVAL_MS);
		} else if (autoRefreshTimer) {
			clearInterval(autoRefreshTimer);
			autoRefreshTimer = null;
		}
	}

	async function handleCopyDiagnostics() {
		if (isCopying) return;
		isCopying = true;
		try {
			const bundle = await fetchDiagnosticsBundle();
			const markdown = formatBundleAsMarkdown(bundle);
			await navigator.clipboard.writeText(markdown);
			toastStore.success("Diagnostics copied to clipboard");
		} catch (err) {
			const msg = err instanceof Error ? err.message : "Failed to copy diagnostics";
			toastStore.error(msg);
		} finally {
			isCopying = false;
		}
	}

	function levelColor(level: string): string {
		switch (level.toLowerCase()) {
			case "error":
			case "dpanic":
			case "panic":
			case "fatal":
				return "text-red-700 bg-red-50 dark:text-red-300 dark:bg-red-900/30";
			case "warn":
				return "text-amber-700 bg-amber-50 dark:text-amber-300 dark:bg-amber-900/30";
			case "info":
				return "text-blue-700 bg-blue-50 dark:text-blue-300 dark:bg-blue-900/30";
			case "debug":
				return "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-800";
			default:
				return "text-gray-700 bg-gray-100 dark:text-gray-300 dark:bg-gray-800";
		}
	}

	// escapePath shell-escapes spaces so users can copy a displayed path
	// straight into a terminal (e.g. "Application Support" stays one path
	// component instead of breaking after "Application").
	function escapePath(p: string): string {
		return p.replace(/ /g, "\\ ");
	}

	function formatLogTime(t: string): string {
		const parsed = new Date(t);
		if (Number.isNaN(parsed.getTime())) return t;
		return parsed.toLocaleTimeString(undefined, {
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		});
	}

	// rateUsageStyle returns the headline + card border colors for a usage ratio.
	// Neutral up to 80% used, amber up to 95%, red beyond — chosen to mirror how
	// GitHub itself flags rate-limit risk on its dashboard.
	function rateUsageStyle(percent: number): { headline: string; card: string } {
		if (percent >= 95) {
			return {
				headline: "text-red-700 dark:text-red-300",
				card: "border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-950/30",
			};
		}
		if (percent >= 80) {
			return {
				headline: "text-amber-700 dark:text-amber-300",
				card: "border-amber-200 dark:border-amber-900/50 bg-amber-50 dark:bg-amber-950/30",
			};
		}
		return {
			headline: "text-gray-900 dark:text-gray-100",
			card: "border-gray-100 dark:border-gray-700 bg-white dark:bg-gray-800",
		};
	}

	function formatResetTime(epochSeconds: number): string {
		const ms = epochSeconds * 1000;
		const now = Date.now();
		const diffSec = Math.round((ms - now) / 1000);
		if (diffSec <= 0) return "now";
		if (diffSec < 60) return `${diffSec}s`;
		const minutes = Math.floor(diffSec / 60);
		const seconds = diffSec % 60;
		if (minutes < 60) return `${minutes}m ${seconds}s`;
		const hours = Math.floor(minutes / 60);
		return `${hours}h ${minutes % 60}m`;
	}

	$: rateBuckets = githubStatus?.rateLimit
		? [
				{ label: "Core", bucket: githubStatus.rateLimit.resources?.core },
				{ label: "GraphQL", bucket: githubStatus.rateLimit.resources?.graphql },
				{ label: "Search", bucket: githubStatus.rateLimit.resources?.search },
			]
		: [];
</script>

<div class="space-y-8">
	<!-- App Info -->
	<section>
		<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">App info</h3>
		<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">Version, paths, and runtime details</p>

		<div
			class="mt-3 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
		>
			{#if infoError}
				<p class="text-sm text-red-600 dark:text-red-400">{infoError}</p>
			{:else if !info}
				<p class="text-sm text-gray-500 dark:text-gray-400">Loading...</p>
			{:else}
				<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-sm">
					<div>
						<dt class="text-gray-500 dark:text-gray-400">Version</dt>
						<dd class="font-mono text-gray-900 dark:text-gray-100">{info.version}</dd>
					</div>
					<div>
						<dt class="text-gray-500 dark:text-gray-400">Platform</dt>
						<dd class="font-mono text-gray-900 dark:text-gray-100">
							{info.os}/{info.arch} · {info.goVersion}
						</dd>
					</div>
					<div class="sm:col-span-2">
						<dt class="text-gray-500 dark:text-gray-400">Data directory</dt>
						<dd class="font-mono text-xs text-gray-900 dark:text-gray-100 break-all">
							{escapePath(info.dataDir)}
						</dd>
					</div>
					<div class="sm:col-span-2">
						<dt class="text-gray-500 dark:text-gray-400">Log file</dt>
						<dd class="font-mono text-xs text-gray-900 dark:text-gray-100 break-all">
							{escapePath(info.logFile)}
						</dd>
					</div>
				</dl>
			{/if}
		</div>
	</section>

	<!-- GitHub Status -->
	<section>
		<div class="flex items-center justify-between">
			<div>
				<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">GitHub status</h3>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Connection and rate limit (fetched live)
				</p>
			</div>
			<button
				type="button"
				on:click={loadGitHubStatus}
				class="text-xs text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300 cursor-pointer"
			>
				Refresh
			</button>
		</div>

		<div
			class="mt-3 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
		>
			{#if githubError}
				<p class="text-sm text-red-600 dark:text-red-400">{githubError}</p>
			{:else if !githubStatus}
				<p class="text-sm text-gray-500 dark:text-gray-400">Loading...</p>
			{:else if !githubStatus.connected}
				<p class="text-sm text-gray-500 dark:text-gray-400">Not connected to GitHub.</p>
			{:else}
				<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-sm">
					<div>
						<dt class="text-gray-500 dark:text-gray-400">Account</dt>
						<dd class="font-mono text-gray-900 dark:text-gray-100">
							{githubStatus.githubUsername ?? "—"}
						</dd>
					</div>
					<div>
						<dt class="text-gray-500 dark:text-gray-400">Source</dt>
						<dd class="font-mono text-gray-900 dark:text-gray-100">{githubStatus.source}</dd>
					</div>
					{#if githubStatus.tokenExpiresAt}
						<div class="sm:col-span-2">
							<dt class="text-gray-500 dark:text-gray-400">Token expires</dt>
							<dd class="font-mono text-gray-900 dark:text-gray-100">
								{new Date(githubStatus.tokenExpiresAt).toLocaleString()}
							</dd>
						</div>
					{/if}
				</dl>

				{#if githubStatus.rateLimitError}
					<p class="mt-3 text-xs text-amber-600 dark:text-amber-400">
						Rate limit fetch failed: {githubStatus.rateLimitError}
					</p>
				{:else if githubStatus.rateLimit}
					<div class="mt-4">
						<h4 class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase mb-2">
							Rate limits
						</h4>
						<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
							{#each rateBuckets as item (item.label)}
								{#if item.bucket}
									{@const percent =
										item.bucket.limit > 0
											? Math.round((item.bucket.used / item.bucket.limit) * 100)
											: 0}
									{@const style = rateUsageStyle(percent)}
									<div class="rounded-lg p-3 border {style.card}">
										<div class="text-xs text-gray-500 dark:text-gray-400">{item.label}</div>
										<div class="text-lg font-semibold {style.headline}">{percent}% used</div>
										<div class="text-xs text-gray-500 dark:text-gray-400">
											{item.bucket.used.toLocaleString()} / {item.bucket.limit.toLocaleString()}
										</div>
										<div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
											Resets in {formatResetTime(item.bucket.reset)}
										</div>
									</div>
								{/if}
							{/each}
						</div>
					</div>
				{/if}
			{/if}
		</div>
	</section>

	<!-- Log Viewer -->
	<section>
		<div class="flex items-center justify-between">
			<div>
				<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">Recent logs</h3>
				<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
					Newest first. Tokens are redacted; other content like repo names, PR titles, and URLs is
					shown as-is - only share in private channels.
				</p>
			</div>
			{#if lastLogsLoadedAt}
				<span class="text-xs text-gray-500 dark:text-gray-400">
					Updated {lastLogsLoadedAt.toLocaleTimeString()}
				</span>
			{/if}
		</div>

		<div class="mt-3 flex flex-wrap gap-2 items-center">
			<select
				value={levelFilter}
				on:change={handleLevelChange}
				class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 dark:border-gray-700 dark:bg-gray-800 dark:text-white"
			>
				<option value="">All levels</option>
				<option value="debug">Debug+</option>
				<option value="info">Info+</option>
				<option value="warn">Warn+</option>
				<option value="error">Error+</option>
			</select>

			<input
				type="text"
				placeholder="Search..."
				value={searchFilter}
				on:input={handleSearchInput}
				class="flex-1 min-w-[10rem] rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 dark:border-gray-700 dark:bg-gray-800 dark:text-white"
			/>

			<button
				type="button"
				on:click={loadLogs}
				class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700 cursor-pointer"
			>
				Reload
			</button>

			<label
				class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 cursor-pointer"
			>
				<input
					type="checkbox"
					checked={autoRefresh}
					on:change={toggleAutoRefresh}
					class="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 cursor-pointer"
				/>
				Auto-refresh
			</label>
		</div>

		{#if logsError}
			<p class="mt-3 text-sm text-red-600 dark:text-red-400">{logsError}</p>
		{:else if logs.length === 0}
			<p class="mt-3 text-sm text-gray-500 dark:text-gray-400">No matching log entries.</p>
		{:else}
			{#if logsTruncated}
				<p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
					Showing the most recent {LOG_LIMIT} matches — older entries are in the log file.
				</p>
			{/if}
			<div
				class="mt-3 rounded-lg border border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900/60 max-h-[28rem] overflow-y-auto"
			>
				<ul class="divide-y divide-gray-200 dark:divide-gray-800">
					{#each logs as entry, i (i)}
						<li class="px-3 py-2 text-xs font-mono">
							<div class="flex items-start gap-2">
								<span
									class="inline-block px-1.5 py-0.5 rounded text-[10px] uppercase font-semibold {levelColor(
										entry.level
									)}"
								>
									{entry.level}
								</span>
								<span class="text-gray-500 dark:text-gray-400 whitespace-nowrap"
									>{formatLogTime(entry.time)}</span
								>
								<span class="text-gray-900 dark:text-gray-100 break-all">{entry.msg}</span>
							</div>
							{#if entry.fields && Object.keys(entry.fields).length > 0}
								<details class="mt-1 ml-12">
									<summary
										class="text-[11px] text-gray-500 dark:text-gray-400 cursor-pointer hover:text-gray-700 dark:hover:text-gray-300"
									>
										{Object.keys(entry.fields).length} field{Object.keys(entry.fields).length === 1
											? ""
											: "s"}
									</summary>
									<pre
										class="mt-1 text-[11px] text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-all">{JSON.stringify(
											entry.fields,
											null,
											2
										)}</pre>
								</details>
							{/if}
							{#if entry.stacktrace}
								<details class="mt-1 ml-12">
									<summary
										class="text-[11px] text-red-600 dark:text-red-400 cursor-pointer hover:text-red-700 dark:hover:text-red-300"
									>
										Stack trace
									</summary>
									<pre
										class="mt-1 text-[11px] text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-all">{entry.stacktrace}</pre>
								</details>
							{/if}
						</li>
					{/each}
				</ul>
			</div>
		{/if}
	</section>

	<!-- Diagnostics Actions -->
	<section>
		<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">Diagnostics</h3>

		<div class="mt-3 space-y-4">
			<div>
				<button
					type="button"
					on:click={handleCopyDiagnostics}
					disabled={isCopying}
					class="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
				>
					{isCopying ? "Copying..." : "Copy diagnostics"}
				</button>
				<p class="mt-1.5 text-xs text-gray-600 dark:text-gray-400">
					Version, paths, and token health only. Safe to paste into a public bug report.
				</p>
			</div>

			<div>
				<a
					href={logsDownloadUrl()}
					class="inline-flex items-center gap-2 rounded-full border border-gray-300 bg-white px-4 py-2 text-xs font-semibold text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700 cursor-pointer"
				>
					Download logs (zip)
				</a>
				<p class="mt-1.5 text-xs text-gray-600 dark:text-gray-400">
					Raw log files. No redaction is applied. Entries may include repo names, PR titles, URLs,
					and other operational detail. Share only in private channels.
				</p>
			</div>
		</div>
	</section>
</div>
