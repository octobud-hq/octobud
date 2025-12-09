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
// along with this program.  If not, see https://www.gnu.org/licenses/.

/**
 * Types for syntax highlighting tokens
 */
export type TokenType =
	| "key"
	| "negation"
	| "colon"
	| "value"
	| "whitespace"
	| "text"
	| "operator"
	| "paren";

export interface SyntaxToken {
	type: TokenType;
	text: string;
}

/**
 * Tokenize a query string for syntax highlighting.
 * Parses the input into tokens that can be used to apply syntax highlighting
 * to query input fields.
 *
 * @param input - The query string to tokenize
 * @returns Array of syntax tokens
 */
export function tokenizeForSyntax(input: string): SyntaxToken[] {
	const tokens: SyntaxToken[] = [];
	let i = 0;

	while (i < input.length) {
		// Check for parentheses
		if (input[i] === "(" || input[i] === ")") {
			tokens.push({ type: "paren", text: input[i] });
			i++;
			continue;
		}

		// Check for negation operator at the start or after whitespace
		if (input[i] === "-" && (i === 0 || /\s/.test(input[i - 1]))) {
			const start = i;
			i++;
			// Check if this is followed by a key:value pattern
			while (i < input.length && /[a-zA-Z0-9_]/.test(input[i])) {
				i++;
			}
			if (i < input.length && input[i] === ":") {
				// It's a negation operator
				tokens.push({ type: "negation", text: "-" });
				i = start + 1;
				continue;
			}
			// Not a negation, just a dash
			i = start;
		}

		// Check for operators (AND, OR, NOT - all caps only)
		if (/[A-Z]/.test(input[i])) {
			const start = i;
			while (i < input.length && /[A-Z]/.test(input[i])) {
				i++;
			}
			const word = input.substring(start, i);
			if (word === "AND" || word === "OR" || word === "NOT") {
				tokens.push({ type: "operator", text: word });
				continue;
			}
			// Not an operator, continue as key or text
			i = start;
		}

		// Check for key:value pattern
		if (/[a-zA-Z_]/.test(input[i])) {
			const start = i;
			while (i < input.length && /[a-zA-Z0-9_]/.test(input[i])) {
				i++;
			}
			if (i < input.length && input[i] === ":") {
				// It's a key
				tokens.push({
					type: "key",
					text: input.substring(start, i),
				});
				tokens.push({ type: "colon", text: ":" });
				i++;
				// Now capture the value (until whitespace, parentheses, or end)
				const valueStart = i;
				while (i < input.length && !/[\s()]/.test(input[i])) {
					i++;
				}
				if (i > valueStart) {
					tokens.push({
						type: "value",
						text: input.substring(valueStart, i),
					});
				}
				continue;
			}
			// Not a key, just regular text
			tokens.push({ type: "text", text: input.substring(start, i) });
			continue;
		}

		// Whitespace
		if (/\s/.test(input[i])) {
			const start = i;
			while (i < input.length && /\s/.test(input[i])) {
				i++;
			}
			tokens.push({
				type: "whitespace",
				text: input.substring(start, i),
			});
			continue;
		}

		// Other characters
		tokens.push({ type: "text", text: input[i] });
		i++;
	}

	return tokens;
}
