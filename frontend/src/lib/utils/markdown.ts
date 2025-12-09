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
 * Utility for rendering markdown content with syntax highlighting
 */

import { marked } from "marked";
import DOMPurify from "dompurify";
import hljs from "highlight.js";
import "./markdown-styles.css"; // Custom styles with light/dark theme support
import { replaceGitHubAttachmentImages } from "./imageProxy";

// Configure marked for GitHub Flavored Markdown
marked.setOptions({
	gfm: true,
	breaks: true,
});

/**
 * Render markdown content to HTML with syntax highlighting for code blocks.
 * Also handles image proxy replacement and sanitization.
 *
 * Note: highlight.js auto-detection works with markdown triple backtick code blocks (```).
 * Marked parses the markdown first, then calls our renderer.code() function with the
 * extracted code and optional language parameter.
 *
 * @param content - Raw markdown content string
 * @returns Sanitized HTML string ready for rendering
 */
export function renderMarkdown(content: string): string {
	try {
		// Create a renderer with custom code block handling for syntax highlighting
		const renderer = new marked.Renderer();

		renderer.code = ({ text, lang }: { text: string; lang?: string; escaped?: boolean }) => {
			const codeString = String(text || "");
			const normalizedLang =
				lang && typeof lang === "string" ? lang.trim().toLowerCase() : undefined;

			// Try to highlight with specified language
			if (normalizedLang && hljs.getLanguage(normalizedLang)) {
				try {
					const highlightResult = hljs.highlight(codeString, { language: normalizedLang });
					if (
						highlightResult &&
						typeof highlightResult === "object" &&
						"value" in highlightResult
					) {
						const htmlValue = (highlightResult as { value: string }).value;
						if (typeof htmlValue === "string") {
							return `<pre><code class="hljs language-${normalizedLang}">${htmlValue}</code></pre>`;
						}
					}
				} catch (err) {
					// Fall through to auto-detect
				}
			}

			// Auto-detect language (works with triple backtick blocks without language)
			try {
				const highlightResult = hljs.highlightAuto(codeString);
				if (highlightResult && typeof highlightResult === "object" && "value" in highlightResult) {
					const htmlValue = (highlightResult as { value: string }).value;
					const detectedLang = (highlightResult as { language?: string }).language || "plaintext";
					if (typeof htmlValue === "string") {
						return `<pre><code class="hljs language-${detectedLang}">${htmlValue}</code></pre>`;
					}
				}
			} catch (err) {
				// Fall back to plain code block
			}

			// Fall back to plain code block
			// DOMPurify will handle escaping text content when sanitizing
			return `<pre><code class="hljs">${codeString}</code></pre>`;
		};

		// Render markdown to HTML
		const rawHtml = marked(content, { renderer }) as string;

		// Replace GitHub attachment images
		const withImageProxies = replaceGitHubAttachmentImages(rawHtml);

		// Sanitize HTML - use DOMPurify defaults (works fine for markdown + highlight.js)
		const sanitized = DOMPurify.sanitize(withImageProxies);

		return sanitized;
	} catch (error) {
		// Fallback to sanitized plain HTML if markdown parsing fails
		return DOMPurify.sanitize(content);
	}
}
