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
 * Utility functions for rendering badges on the favicon.
 * Similar to how Gmail shows the number of unread emails.
 */

const FAVICON_SIZE = 32;
const BADGE_SIZE = 24;
const BADGE_FONT_SIZE = 20; // Font size with 2px padding (24 - 4 = 20)

let originalFavicon: string | null = null;
let currentBadgeCount: number | null = null;

/**
 * Get the badge text to display based on count.
 * Returns empty string for 0, the count for 1-50, or "50+" for >50.
 */
export function getBadgeText(count: number): string {
	if (count <= 0) {
		return "";
	}
	if (count <= 50) {
		return count.toString();
	}
	return "50+";
}

/**
 * Initialize the original favicon URL.
 * Should be called once on app startup.
 */
export function initializeFavicon(): void {
	if (typeof document === "undefined") return;

	const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
	if (link && link.href && !originalFavicon) {
		originalFavicon = link.href;
	}
}

/**
 * Update the favicon with a badge showing the unread count.
 * @param count - The number of unread notifications
 */
export function updateFaviconBadge(count: number): void {
	if (typeof document === "undefined") return;

	// Skip if count hasn't changed
	if (currentBadgeCount === count) {
		return;
	}
	currentBadgeCount = count;

	// Initialize original favicon if not already done
	if (!originalFavicon) {
		initializeFavicon();
	}

	const badgeText = getBadgeText(count);

	// If no badge needed, restore original favicon
	if (badgeText === "") {
		restoreOriginalFavicon();
		return;
	}

	// Create canvas for drawing
	const canvas = document.createElement("canvas");
	canvas.width = FAVICON_SIZE;
	canvas.height = FAVICON_SIZE;
	const ctx = canvas.getContext("2d");
	if (!ctx) return;

	// Load the original favicon image
	const img = new Image();
	img.crossOrigin = "anonymous";

	img.onload = () => {
		// Draw the original favicon
		ctx.drawImage(img, 0, 0, FAVICON_SIZE, FAVICON_SIZE);

		// Draw badge background (white circle)
		const badgeX = FAVICON_SIZE - BADGE_SIZE / 2;
		const badgeY = BADGE_SIZE / 2;
		ctx.fillStyle = "#ffffff"; // white
		ctx.beginPath();
		ctx.arc(badgeX, badgeY, BADGE_SIZE / 2, 0, 2 * Math.PI);
		ctx.fill();

		// Draw badge text
		ctx.fillStyle = "#000000"; // black
		ctx.font = `bold ${BADGE_FONT_SIZE}px Arial, sans-serif`;
		ctx.textAlign = "center";
		ctx.textBaseline = "middle";
		ctx.fillText(badgeText, badgeX, badgeY);

		// Update the favicon
		updateFaviconHref(canvas.toDataURL("image/png"));
	};

	img.onerror = () => {
		// Fallback: create badge without original favicon
		// Draw a simple colored background
		ctx.fillStyle = "#6366f1"; // indigo-500
		ctx.fillRect(0, 0, FAVICON_SIZE, FAVICON_SIZE);

		// Draw badge background (white circle)
		const badgeX = FAVICON_SIZE - BADGE_SIZE / 2;
		const badgeY = BADGE_SIZE / 2;
		ctx.fillStyle = "#ffffff"; // white
		ctx.beginPath();
		ctx.arc(badgeX, badgeY, BADGE_SIZE / 2, 0, 2 * Math.PI);
		ctx.fill();

		// Draw badge text
		ctx.fillStyle = "#000000"; // black
		ctx.font = `bold ${BADGE_FONT_SIZE}px Arial, sans-serif`;
		ctx.textAlign = "center";
		ctx.textBaseline = "middle";
		ctx.fillText(badgeText, badgeX, badgeY);

		// Update the favicon
		updateFaviconHref(canvas.toDataURL("image/png"));
	};

	// Start loading the image
	if (originalFavicon) {
		img.src = originalFavicon;
	}
}

/**
 * Restore the original favicon without any badge.
 */
export function restoreOriginalFavicon(): void {
	if (typeof document === "undefined") return;

	if (originalFavicon) {
		updateFaviconHref(originalFavicon);
	}
	currentBadgeCount = null;
}

/**
 * Update the favicon link element href.
 */
function updateFaviconHref(href: string): void {
	const link =
		document.querySelector<HTMLLinkElement>('link[rel="icon"]') || document.createElement("link");

	link.type = "image/png";
	link.rel = "icon";
	link.href = href;

	if (!link.parentElement) {
		document.head.appendChild(link);
	}
}

/**
 * Reset the favicon badge state (for testing).
 * @internal
 */
export function _resetFaviconState(): void {
	originalFavicon = null;
	currentBadgeCount = null;
}
