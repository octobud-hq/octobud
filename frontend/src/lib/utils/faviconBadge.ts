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
const BADGE_HEIGHT = 20;
const BADGE_FONT_SIZE = 18; // Font size with 2px padding (24 - 4 = 20)
const BADGE_TEXT_PADDING = 2;

let originalFavicon: string | null = null;
let currentBadgeCount: number | null = null;

/**
 * Get the badge text to display based on count.
 * Returns "0" for 0, the count as a string for 1-50, or "50+" for >50.
 */
export function getBadgeText(count: number): string {
	if (count <= 50) {
		return count.toString();
	}
	return "50+";
}

/**
 * Draw a rounded rectangle on the canvas context.
 * Uses roundRect() if available (Chrome 99+, Firefox 112+, Safari 16+),
 * otherwise falls back to path-based drawing for older browsers.
 */
function drawRoundedRect(
	ctx: CanvasRenderingContext2D,
	x: number,
	y: number,
	width: number,
	height: number,
	radius: number
): void {
	if (typeof ctx.roundRect === "function") {
		// Modern browsers: use native roundRect (supported in Chrome 99+, Firefox 112+, Safari 16+)
		ctx.roundRect(x, y, width, height, radius);
	} else {
		// Fallback for older browsers: draw rounded rectangle using path API
		ctx.beginPath();
		ctx.moveTo(x + radius, y);
		ctx.lineTo(x + width - radius, y);
		ctx.arcTo(x + width, y, x + width, y + radius, radius);
		ctx.lineTo(x + width, y + height - radius);
		ctx.arcTo(x + width, y + height, x + width - radius, y + height, radius);
		ctx.lineTo(x + radius, y + height);
		ctx.arcTo(x, y + height, x, y + height - radius, radius);
		ctx.lineTo(x, y + radius);
		ctx.arcTo(x, y, x + radius, y, radius);
		ctx.closePath();
	}
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

	// If no favicon was found, we can't draw a badge
	if (!originalFavicon) {
		return;
	}

	const badgeText = getBadgeText(count);

	const bodyElement = document.getElementsByTagName("body")[0];
	const fontFamily = bodyElement
		? window.getComputedStyle(bodyElement)["fontFamily"]
		: "system-ui, -apple-system, sans-serif";

	// Create canvas for drawing
	const canvas = document.createElement("canvas");
	canvas.width = FAVICON_SIZE;
	canvas.height = FAVICON_SIZE;
	const ctx = canvas.getContext("2d");
	if (!ctx) return;

	// Set font properties before measuring text
	ctx.font = `${BADGE_FONT_SIZE}px ${fontFamily}`;
	ctx.textAlign = "center";
	ctx.textBaseline = "middle";
	ctx.letterSpacing = "-1px";

	// Find the badge width as the minimum of (text width + padding) and the favicon size
	const badgeWidth = Math.min(
		FAVICON_SIZE,
		Math.ceil(ctx.measureText(badgeText).width + BADGE_TEXT_PADDING)
	);

	// Load the original favicon image
	const img = new Image();
	img.crossOrigin = "anonymous";

	img.onload = () => {
		// Draw the original favicon
		ctx.drawImage(img, 0, 0, FAVICON_SIZE, FAVICON_SIZE);

		// Draw badge background (white rounded rectangle)
		const badgeX = FAVICON_SIZE - badgeWidth;
		const badgeY = FAVICON_SIZE - BADGE_HEIGHT;
		ctx.fillStyle = "#ffffff"; // white
		drawRoundedRect(ctx, badgeX, badgeY, badgeWidth, BADGE_HEIGHT, 5);
		ctx.fill();

		// Draw badge text (centered in the badge)
		ctx.fillStyle = "#000000"; // black
		// With textAlign="center" and textBaseline="middle", text is centered at (textX, textY)
		const textX = badgeX + badgeWidth / 2;
		const textY = badgeY + BADGE_HEIGHT / 2;
		ctx.fillText(badgeText, textX, textY);

		// Update the favicon
		updateFaviconHref(canvas.toDataURL("image/png"));
	};

	// Fallback: restore the original favicon
	img.onerror = () => restoreOriginalFavicon();

	// Start loading the image
	img.src = originalFavicon;
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
