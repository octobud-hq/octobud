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
import {
	getBadgeText,
	initializeFavicon,
	updateFaviconBadge,
	restoreOriginalFavicon,
	_resetFaviconState,
} from "./faviconBadge";

describe("getBadgeText", () => {
	it("returns '0' for 0", () => {
		expect(getBadgeText(0)).toBe("0");
	});

	it("returns string representation for negative numbers", () => {
		expect(getBadgeText(-1)).toBe("-1");
		expect(getBadgeText(-100)).toBe("-100");
	});

	it("returns the count as string for 1", () => {
		expect(getBadgeText(1)).toBe("1");
	});

	it("returns the count as string for counts 2-49", () => {
		expect(getBadgeText(2)).toBe("2");
		expect(getBadgeText(25)).toBe("25");
		expect(getBadgeText(49)).toBe("49");
	});

	it("returns the count as string for 50", () => {
		expect(getBadgeText(50)).toBe("50");
	});

	it('returns "50+" for counts over 50', () => {
		expect(getBadgeText(51)).toBe("50+");
		expect(getBadgeText(100)).toBe("50+");
		expect(getBadgeText(1000)).toBe("50+");
		expect(getBadgeText(9999)).toBe("50+");
	});
});

describe("initializeFavicon", () => {
	beforeEach(() => {
		_resetFaviconState();
		document.head.innerHTML = "";
	});

	it("stores the original favicon href", () => {
		const link = document.createElement("link");
		link.rel = "icon";
		link.href = "/favicon.png";
		document.head.appendChild(link);

		initializeFavicon();

		// We can't directly test the internal state, but we can verify no errors occur
		expect(document.querySelector('link[rel="icon"]')).toBeTruthy();
	});

	it("does nothing when no favicon link exists", () => {
		expect(() => initializeFavicon()).not.toThrow();
	});
});

describe("updateFaviconBadge", () => {
	beforeEach(() => {
		_resetFaviconState();
		document.head.innerHTML = "";
	});

	it("initializes favicon when called", () => {
		// Set up initial favicon
		const link = document.createElement("link");
		link.rel = "icon";
		link.href = "/favicon.png";
		document.head.appendChild(link);

		initializeFavicon();

		// Calling updateFaviconBadge should not throw
		expect(() => updateFaviconBadge(5)).not.toThrow();
	});

	it("restores original favicon when count is 0", () => {
		const link = document.createElement("link");
		link.rel = "icon";
		link.href = "/favicon.png";
		document.head.appendChild(link);

		initializeFavicon();
		updateFaviconBadge(0);

		const faviconLink = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
		expect(faviconLink?.href).toContain("favicon.png");
	});
});

describe("restoreOriginalFavicon", () => {
	beforeEach(() => {
		_resetFaviconState();
		document.head.innerHTML = "";
	});

	it("restores the original favicon href", () => {
		const link = document.createElement("link");
		link.rel = "icon";
		link.href = "/favicon.png";
		document.head.appendChild(link);

		initializeFavicon();
		restoreOriginalFavicon();

		const faviconLink = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
		expect(faviconLink?.href).toContain("favicon.png");
	});

	it("does nothing when no original favicon was set", () => {
		expect(() => restoreOriginalFavicon()).not.toThrow();
	});
});
