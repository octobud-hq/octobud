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

import { storageGet, storageSet } from "./browser";

const STORAGE_KEY = "octobud:backendUrl";

/**
 * Ports Octobud is reachable on. 8808 is the packaged binary's default; 8080 is
 * what `make backend-dev` uses. Probed in that order.
 */
export const CANDIDATE_BASE_URLS = ["http://localhost:8808", "http://localhost:8080"] as const;

export const DEFAULT_BASE_URL = CANDIDATE_BASE_URLS[0];

const PROBE_TIMEOUT_MS = 1500;

let activeBaseUrl: string = DEFAULT_BASE_URL;

export function normalizeBaseUrl(raw: string): string {
	return raw.trim().replace(/\/+$/, "");
}

/**
 * Synchronous accessor used by `buildApiUrl`. Reflects whatever the last
 * `initBackendUrl`/`setBackendUrl` call resolved to.
 */
export function getBackendUrl(): string {
	return activeBaseUrl;
}

export async function setBackendUrl(raw: string): Promise<string> {
	activeBaseUrl = normalizeBaseUrl(raw) || DEFAULT_BASE_URL;
	await storageSet(STORAGE_KEY, activeBaseUrl);
	return activeBaseUrl;
}

/**
 * Cheap liveness check against `/healthz`, which is the one unauthenticated
 * endpoint the backend exposes.
 *
 * Worth doing before the first real request even when the URL is already known:
 * a connection to a port nothing is listening on can hang for as long as the OS
 * allows, and the request timeout that backstops that is deliberately generous.
 * Probing first turns "Octobud isn't running" into an answer in under two
 * seconds instead of ten.
 */
export async function isBackendReachable(baseUrl: string = activeBaseUrl): Promise<boolean> {
	return isReachable(baseUrl);
}

async function isReachable(baseUrl: string): Promise<boolean> {
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS);
	try {
		const response = await fetch(`${baseUrl}/healthz`, { signal: controller.signal });
		return response.ok;
	} catch {
		return false;
	} finally {
		clearTimeout(timer);
	}
}

/**
 * Resolves the backend base URL once at panel startup.
 *
 * A stored value always wins so an explicit override is never silently
 * overwritten by a probe. Otherwise the candidates are tried in order, and the
 * first one answering `/healthz` is persisted. When nothing answers we keep the
 * default so the panel renders its "backend unreachable" state against a
 * sensible URL instead of an empty one.
 */
export async function initBackendUrl(): Promise<string> {
	const stored = await storageGet<string>(STORAGE_KEY);
	if (typeof stored === "string" && stored.trim().length > 0) {
		activeBaseUrl = normalizeBaseUrl(stored);
		return activeBaseUrl;
	}

	for (const candidate of CANDIDATE_BASE_URLS) {
		if (await isReachable(candidate)) {
			return setBackendUrl(candidate);
		}
	}

	activeBaseUrl = DEFAULT_BASE_URL;
	return activeBaseUrl;
}
