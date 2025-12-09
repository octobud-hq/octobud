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

import { writable, type Writable } from "svelte/store";
import { browser } from "$app/environment";

const THEME_KEY = "octobud:theme";

export type Theme = "light" | "dark";

/**
 * Theme Store
 * Manages theme preferences persisted to localStorage
 */
export function createThemeStore() {
	// Load from localStorage, default to 'dark'
	const getStoredTheme = (): Theme => {
		if (typeof window === "undefined") return "dark";
		const stored = localStorage.getItem(THEME_KEY);
		if (stored === "light" || stored === "dark") {
			return stored as Theme;
		}
		return "dark";
	};

	const theme = writable<Theme>(getStoredTheme());

	// Apply theme class to html element and persist to localStorage
	if (browser) {
		const applyTheme = (newTheme: Theme) => {
			const htmlElement = document.documentElement;
			if (newTheme === "dark") {
				htmlElement.classList.add("dark");
			} else {
				htmlElement.classList.remove("dark");
			}
			localStorage.setItem(THEME_KEY, newTheme);
		};

		// Apply initial theme immediately to prevent flash
		const initialTheme = getStoredTheme();
		applyTheme(initialTheme);
		theme.set(initialTheme);

		// Subscribe to changes and apply theme
		theme.subscribe((value) => {
			applyTheme(value);
		});
	}

	return {
		theme: theme as Writable<Theme>,
		setTheme: (newTheme: Theme) => {
			theme.set(newTheme);
		},
		toggle: () => {
			theme.update((current) => (current === "dark" ? "light" : "dark"));
		},
	};
}

// Singleton instance
let themeStoreInstance: ReturnType<typeof createThemeStore> | null = null;

export function getThemeStore() {
	if (!themeStoreInstance) {
		themeStoreInstance = createThemeStore();
	}
	return themeStoreInstance;
}
