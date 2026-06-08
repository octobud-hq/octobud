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

import { fetchWithAuth, buildApiUrl, apiErrorFromResponse } from "./fetch";

export interface DiagnosticsInfo {
	version: string;
	goVersion: string;
	os: string;
	arch: string;
	dataDir: string;
	logDir: string;
	logFile: string;
	startedAt: string;
	uptimeSeconds: number;
}

export interface LogEntry {
	time: string;
	level: string;
	logger?: string;
	caller?: string;
	msg: string;
	stacktrace?: string;
	fields?: Record<string, unknown>;
}

export interface LogsResponse {
	entries: LogEntry[];
	returned: number;
	truncated: boolean;
}

export interface RateLimitBucket {
	limit: number;
	used: number;
	remaining: number;
	reset: number;
}

export interface RateLimit {
	resources: Record<string, RateLimitBucket>;
	rate: RateLimitBucket;
}

export interface GitHubStatus {
	connected: boolean;
	githubUsername?: string;
	githubUserId?: string;
	maskedToken?: string;
	source: string;
	tokenExpiresAt?: string;
	rateLimit?: RateLimit;
	rateLimitError?: string;
}

export interface DiagnosticsBundle {
	generatedAt: string;
	info: DiagnosticsInfo;
	tokenSource?: string;
	tokenExpiresAt?: string;
}

export interface FetchLogsOptions {
	level?: string;
	since?: string;
	search?: string;
	limit?: number;
}

export async function fetchDiagnosticsInfo(): Promise<DiagnosticsInfo> {
	const response = await fetchWithAuth("/api/diagnostics/info");
	if (!response.ok) {
		throw await apiErrorFromResponse(response, `Failed to load info (${response.status})`);
	}
	return response.json();
}

export async function fetchLogs(options: FetchLogsOptions = {}): Promise<LogsResponse> {
	const params = new URLSearchParams();
	if (options.level) params.set("level", options.level);
	if (options.since) params.set("since", options.since);
	if (options.search) params.set("search", options.search);
	if (options.limit) params.set("limit", String(options.limit));
	const query = params.toString();
	const url = query ? `/api/diagnostics/logs?${query}` : "/api/diagnostics/logs";
	const response = await fetchWithAuth(url);
	if (!response.ok) {
		throw await apiErrorFromResponse(response, `Failed to load logs (${response.status})`);
	}
	return response.json();
}

export function logsDownloadUrl(): string {
	return buildApiUrl("/api/diagnostics/logs/download");
}

export async function fetchGitHubStatus(): Promise<GitHubStatus> {
	const response = await fetchWithAuth("/api/diagnostics/github/status");
	if (!response.ok) {
		throw await apiErrorFromResponse(response, `Failed to load GitHub status (${response.status})`);
	}
	return response.json();
}

export async function fetchDiagnosticsBundle(): Promise<DiagnosticsBundle> {
	const response = await fetchWithAuth("/api/diagnostics/bundle");
	if (!response.ok) {
		throw await apiErrorFromResponse(response, `Failed to load bundle (${response.status})`);
	}
	return response.json();
}

/**
 * Format a diagnostics bundle as a markdown block suitable for pasting into a
 * bug report. Intentionally minimal — only fields the backend bundle ships,
 * which are limited to non-sensitive runtime info plus token health. Anything
 * that could leak repo/org names or user activity stays out of this output;
 * users who need to share log detail copy from the in-app viewer themselves.
 */
export function formatBundleAsMarkdown(bundle: DiagnosticsBundle): string {
	const lines: string[] = [];
	lines.push("## Octobud diagnostics");
	lines.push("");
	lines.push(`- **Version:** ${bundle.info.version}`);
	lines.push(`- **Platform:** ${bundle.info.os}/${bundle.info.arch} (Go ${bundle.info.goVersion})`);
	lines.push(
		`- **Uptime:** ${Math.round(bundle.info.uptimeSeconds / 60)}m (started ${bundle.info.startedAt})`
	);
	lines.push(`- **Data dir:** \`${bundle.info.dataDir}\``);
	lines.push(`- **Log file:** \`${bundle.info.logFile}\``);
	if (bundle.tokenSource) {
		lines.push(`- **Token source:** ${bundle.tokenSource}`);
	}
	if (bundle.tokenExpiresAt) {
		lines.push(`- **Token expires:** ${bundle.tokenExpiresAt}`);
	}
	lines.push(`- **Generated:** ${bundle.generatedAt}`);
	return lines.join("\n");
}
