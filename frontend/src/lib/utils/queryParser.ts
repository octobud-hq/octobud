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

// Query parser for GitHub-style filter syntax

import { isValidFilterField as isValidField } from "$lib/constants/filterFields";

export { isValidField };

export interface QueryTerm {
	field: string;
	values: string[];
	negated: boolean;
	startPos: number;
	endPos: number;
}

export interface ParsedQuery {
	terms: QueryTerm[];
	freeText: string;
	errors: string[];
	isValid: boolean;
}

export interface ContextAtCursor {
	type: "field" | "value" | "none";
	field?: string;
	partial: string;
	termIndex?: number;
}

/**
 * Parse a query string into structured terms and free text
 * Syntax: field:value field:value1,value2 -field:value
 * Free text (no colons) is collected and returned separately
 */
export function parseQuery(query: string): ParsedQuery {
	const trimmed = query.trim();
	if (!trimmed) {
		return { terms: [], freeText: "", errors: [], isValid: true };
	}

	const terms: QueryTerm[] = [];
	const freeTextTokens: string[] = [];
	const errors: string[] = [];

	// Split on spaces while respecting quotes
	const parts = splitQuery(trimmed);

	for (const part of parts) {
		// Collect parts that don't contain a colon as free text
		if (!part.text.includes(":")) {
			freeTextTokens.push(part.text);
			continue;
		}

		try {
			const term = parseTerm(part.text, part.start);
			if (term) {
				terms.push(term);
			}
		} catch (error) {
			errors.push(error instanceof Error ? error.message : String(error));
		}
	}

	return {
		terms,
		freeText: freeTextTokens.join(" "),
		errors,
		isValid: errors.length === 0,
	};
}

interface QueryPart {
	text: string;
	start: number;
}

/**
 * Split query string on spaces, respecting quotes
 */
function splitQuery(query: string): QueryPart[] {
	const parts: QueryPart[] = [];
	let current = "";
	let startPos = 0;
	let inQuotes = false;
	let currentStart = 0;

	for (let i = 0; i < query.length; i++) {
		const ch = query[i];

		if (ch === '"') {
			inQuotes = !inQuotes;
			continue;
		}

		if (ch === " " && !inQuotes) {
			if (current.length > 0) {
				parts.push({ text: current, start: currentStart });
				current = "";
			}
			startPos = i + 1;
			currentStart = startPos;
			continue;
		}

		if (current.length === 0) {
			currentStart = i;
		}
		current += ch;
	}

	if (current.length > 0) {
		parts.push({ text: current, start: currentStart });
	}

	return parts;
}

/**
 * Parse a single term like "field:value1,value2" or "-field:value"
 */
function parseTerm(text: string, startPos: number): QueryTerm | null {
	const negated = text.startsWith("-");
	const cleaned = negated ? text.slice(1) : text;

	const colonIndex = cleaned.indexOf(":");
	if (colonIndex === -1) {
		return null;
	}

	const field = cleaned.slice(0, colonIndex).trim().toLowerCase();
	const valuesPart = cleaned.slice(colonIndex + 1).trim();

	if (!field || !valuesPart) {
		throw new Error(`Invalid term format: "${text}"`);
	}

	// Split on commas to get multiple values
	const values = valuesPart
		.split(",")
		.map((v) => v.trim())
		.filter(Boolean);

	if (values.length === 0) {
		throw new Error(`No values provided for field "${field}"`);
	}

	return {
		field,
		values,
		negated,
		startPos,
		endPos: startPos + text.length,
	};
}

/**
 * Get context information at the cursor position
 */
export function getContextAtCursor(query: string, cursorPos: number): ContextAtCursor {
	if (!query || cursorPos === 0) {
		return { type: "none", partial: "" };
	}

	const beforeCursor = query.slice(0, cursorPos);
	const afterCursor = query.slice(cursorPos);

	// Find the start of the current token
	let tokenStart = beforeCursor.lastIndexOf(" ") + 1;

	// Find the end of the current token
	let tokenEnd = afterCursor.indexOf(" ");
	if (tokenEnd === -1) tokenEnd = afterCursor.length;
	tokenEnd = cursorPos + tokenEnd;

	const currentToken = query.slice(tokenStart, tokenEnd);

	// Check if we're in a negated term
	const isNegated = currentToken.startsWith("-");
	const tokenWithoutNegation = isNegated ? currentToken.slice(1) : currentToken;

	const colonIndex = tokenWithoutNegation.indexOf(":");

	if (colonIndex === -1) {
		// No colon yet - we're in field context
		return {
			type: "field",
			partial: tokenWithoutNegation,
		};
	}

	// We're after the colon - value context
	const field = tokenWithoutNegation.slice(0, colonIndex).trim().toLowerCase();
	const valuesPart = tokenWithoutNegation.slice(colonIndex + 1);

	// Find which value we're currently in (if comma-separated)
	const beforeColon = tokenStart + (isNegated ? 1 : 0) + colonIndex + 1;
	const relativePos = cursorPos - beforeColon;

	// Get the partial value up to cursor
	const partialValue = valuesPart.slice(0, relativePos);
	const lastComma = partialValue.lastIndexOf(",");
	const partial = lastComma === -1 ? partialValue : partialValue.slice(lastComma + 1);

	return {
		type: "value",
		field,
		partial: partial.trim(),
	};
}
