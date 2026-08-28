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

/**
 * Minimal cross-browser WebExtension shim.
 *
 * Firefox exposes the promise-based `browser.*` namespace; Chrome exposes
 * `chrome.*`, whose MV3 `storage` methods also return promises. The panel only
 * touches `storage.local`, so the whole compatibility story is picking the
 * namespace that exists rather than pulling in webextension-polyfill.
 */

interface StorageArea {
	get(keys: string[] | string | null): Promise<Record<string, unknown>>;
	set(items: Record<string, unknown>): Promise<void>;
	remove(keys: string[] | string): Promise<void>;
}

interface ExtensionNamespace {
	storage?: { local: StorageArea };
}

function resolveNamespace(): ExtensionNamespace | null {
	const globalScope = globalThis as typeof globalThis & {
		browser?: ExtensionNamespace;
		chrome?: ExtensionNamespace;
	};
	return globalScope.browser ?? globalScope.chrome ?? null;
}

/**
 * True when running inside an extension context. Vitest and `vite dev` are not,
 * so callers fall back to in-memory state rather than throwing.
 */
export function hasExtensionStorage(): boolean {
	return Boolean(resolveNamespace()?.storage?.local);
}

const memoryFallback = new Map<string, unknown>();

export async function storageGet<T>(key: string): Promise<T | undefined> {
	const area = resolveNamespace()?.storage?.local;
	if (!area) {
		return memoryFallback.get(key) as T | undefined;
	}
	const result = await area.get([key]);
	return result[key] as T | undefined;
}

export async function storageSet(key: string, value: unknown): Promise<void> {
	const area = resolveNamespace()?.storage?.local;
	if (!area) {
		memoryFallback.set(key, value);
		return;
	}
	await area.set({ [key]: value });
}
