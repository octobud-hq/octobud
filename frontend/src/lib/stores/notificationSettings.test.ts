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

import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";
import {
	createNotificationSettingsStore,
	isNotificationEnabled,
	isFaviconBadgeEnabled,
	getNotificationSettingsStore,
} from "./notificationSettings";

describe("NotificationSettings Store", () => {
	beforeEach(() => {
		// Clear localStorage before each test
		localStorage.clear();
		vi.clearAllMocks();

		// Reset the singleton instance by clearing it via a new store creation
		// This ensures each test gets a fresh instance
		vi.resetModules();
	});

	describe("createNotificationSettingsStore", () => {
		it("initializes with default values when localStorage is empty", () => {
			const store = createNotificationSettingsStore();

			expect(get(store.notificationsEnabled)).toBe(true);
			expect(get(store.faviconBadgeEnabled)).toBe(true);
		});

		it("loads saved notification enabled state from localStorage", () => {
			localStorage.setItem("octobud:notifications:enabled", "false");

			const store = createNotificationSettingsStore();

			expect(get(store.notificationsEnabled)).toBe(false);
		});

		it("loads saved favicon badge enabled state from localStorage", () => {
			localStorage.setItem("octobud:notifications:faviconBadge", "false");

			const store = createNotificationSettingsStore();

			expect(get(store.faviconBadgeEnabled)).toBe(false);
		});

		it("can set notification enabled state", () => {
			const store = createNotificationSettingsStore();

			store.setEnabled(false);

			expect(get(store.notificationsEnabled)).toBe(false);

			store.setEnabled(true);

			expect(get(store.notificationsEnabled)).toBe(true);
		});

		it("can set favicon badge enabled state", () => {
			const store = createNotificationSettingsStore();

			store.setFaviconBadgeEnabled(false);

			expect(get(store.faviconBadgeEnabled)).toBe(false);

			store.setFaviconBadgeEnabled(true);

			expect(get(store.faviconBadgeEnabled)).toBe(true);
		});

		it("toggles notification enabled state", () => {
			const store = createNotificationSettingsStore();

			expect(get(store.notificationsEnabled)).toBe(true);

			store.toggle();

			expect(get(store.notificationsEnabled)).toBe(false);

			store.toggle();

			expect(get(store.notificationsEnabled)).toBe(true);
		});

		it("toggles favicon badge enabled state", () => {
			const store = createNotificationSettingsStore();

			expect(get(store.faviconBadgeEnabled)).toBe(true);

			store.toggleFaviconBadge();

			expect(get(store.faviconBadgeEnabled)).toBe(false);

			store.toggleFaviconBadge();

			expect(get(store.faviconBadgeEnabled)).toBe(true);
		});
	});

	describe("isNotificationEnabled", () => {
		it("returns true when notifications are enabled in localStorage", () => {
			localStorage.setItem("octobud:notifications:enabled", "true");

			expect(isNotificationEnabled()).toBe(true);
		});

		it("returns true by default when no setting exists", () => {
			localStorage.clear();

			expect(isNotificationEnabled()).toBe(true);
		});
	});

	describe("isFaviconBadgeEnabled", () => {
		it("returns true when favicon badge is enabled in localStorage", () => {
			localStorage.setItem("octobud:notifications:faviconBadge", "true");

			expect(isFaviconBadgeEnabled()).toBe(true);
		});

		it("returns true by default when no setting exists", () => {
			localStorage.clear();

			expect(isFaviconBadgeEnabled()).toBe(true);
		});
	});

	describe("independent settings", () => {
		it("allows notifications to be enabled while favicon badge is disabled", () => {
			const store = createNotificationSettingsStore();

			store.setEnabled(true);
			store.setFaviconBadgeEnabled(false);

			expect(get(store.notificationsEnabled)).toBe(true);
			expect(get(store.faviconBadgeEnabled)).toBe(false);
		});

		it("allows favicon badge to be enabled while notifications are disabled", () => {
			const store = createNotificationSettingsStore();

			store.setEnabled(false);
			store.setFaviconBadgeEnabled(true);

			expect(get(store.notificationsEnabled)).toBe(false);
			expect(get(store.faviconBadgeEnabled)).toBe(true);
		});

		it("allows both settings to be disabled", () => {
			const store = createNotificationSettingsStore();

			store.setEnabled(false);
			store.setFaviconBadgeEnabled(false);

			expect(get(store.notificationsEnabled)).toBe(false);
			expect(get(store.faviconBadgeEnabled)).toBe(false);
		});

		it("allows both settings to be enabled", () => {
			const store = createNotificationSettingsStore();

			store.setEnabled(true);
			store.setFaviconBadgeEnabled(true);

			expect(get(store.notificationsEnabled)).toBe(true);
			expect(get(store.faviconBadgeEnabled)).toBe(true);
		});
	});
});
