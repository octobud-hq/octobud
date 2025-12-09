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

// Utility to replace GitHub user-attachment images with a helpful message
export function replaceGitHubAttachmentImages(html: string): string {
	// Only run in browser context
	if (typeof window === "undefined") {
		return html;
	}

	const parser = new DOMParser();
	const doc = parser.parseFromString(html, "text/html");

	// Find all img tags
	const images = doc.querySelectorAll("img");

	images.forEach((img) => {
		const src = img.getAttribute("src");
		if (src && isGitHubUserAttachmentUrl(src)) {
			// Replace with a badge/message
			const badge = doc.createElement("span");
			badge.className =
				"inline-flex items-center gap-1.5 px-2.5 py-1 my-2 rounded-md bg-blue-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-300 text-sm border border-gray-200 dark:border-gray-700";
			badge.innerHTML = `
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
				<span>Open in GitHub to see attached image</span>
			`;
			img.replaceWith(badge);
		}
	});

	return doc.body.innerHTML;
}

function isGitHubUserAttachmentUrl(url: string): boolean {
	try {
		const parsedUrl = new URL(url);
		const host = parsedUrl.hostname.toLowerCase();

		// Check if it's a user-attachments URL or private-user-images URL
		// Check for github-user-content.com subdomains (like camo.github-user-content.com) - these are GitHub's image proxies for user attachments
		// Also check for githubusercontent.com subdomains with user-attachments in the path
		return (
			(host === "github.com" && parsedUrl.pathname.includes("/user-attachments/")) ||
			host.endsWith(".githubusercontent.com")
		);
	} catch {
		return false;
	}
}
