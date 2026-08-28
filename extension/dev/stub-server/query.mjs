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
 * A deliberately small stand-in for backend/internal/query.
 *
 * The real engine is a lexer -> parser -> AST -> SQL builder supporting
 * parentheses, explicit AND/OR/NOT and free-text search across several columns.
 * This handles only what the panel's views actually need: space-separated terms
 * ANDed together, comma-separated values ORed within one field, and a leading
 * `-` for negation.
 *
 * It exists so the seeded views return visibly different result sets. It is not
 * a reimplementation, and it should not grow into one — if a test needs richer
 * query semantics, run the real backend instead.
 */

const BOOLEAN_FIELDS = new Set(["starred", "archived", "snoozed", "muted", "filtered", "read"]);

function normalizeType(value) {
	const lowered = String(value ?? "").toLowerCase();
	if (lowered === "pullrequest" || lowered === "pull_request") return "pull_request";
	return lowered;
}

function contains(haystack, needle) {
	return String(haystack ?? "")
		.toLowerCase()
		.includes(String(needle ?? "").toLowerCase());
}

function isInInbox(item) {
	return !item.archived && !item.snoozedUntil && !item.filtered;
}

/** Evaluates a single `field:value` pair (already lowercased field). */
function matchesPair(item, field, value) {
	const lowered = String(value).toLowerCase();

	switch (field) {
		case "is":
			switch (lowered) {
				case "read":
					return item.isRead;
				case "unread":
					return !item.isRead;
				case "starred":
					return item.starred;
				case "snoozed":
					return Boolean(item.snoozedUntil);
				case "archived":
					return item.archived;
				case "muted":
					return item.muted;
				case "filtered":
					return item.filtered;
				default:
					return false;
			}

		case "in":
			switch (lowered) {
				case "inbox":
					return isInInbox(item);
				case "archive":
					return item.archived;
				case "snoozed":
					return Boolean(item.snoozedUntil);
				case "filtered":
					return item.filtered;
				case "anywhere":
					return true;
				default:
					return false;
			}

		case "type":
			return normalizeType(item.subjectType) === normalizeType(lowered);

		case "repo":
			return contains(item.repository?.fullName, lowered);

		case "reason":
			return contains(item.reason, lowered);

		case "author":
			return contains(item.authorLogin, lowered);

		case "tags":
			return (item.tags ?? []).some(
				(tag) => tag.slug.toLowerCase() === lowered || contains(tag.name, lowered)
			);

		// `starred:true` style. The frontend's built-in view constants use this
		// form while the backend's system views use `is:`, so both are accepted.
		case "starred":
		case "archived":
		case "muted":
		case "filtered":
		case "read": {
			const actual =
				field === "read" ? item.isRead : field === "starred" ? item.starred : item[field];
			return lowered === "true" ? Boolean(actual) : !actual;
		}
		case "snoozed":
			return lowered === "true" ? Boolean(item.snoozedUntil) : !item.snoozedUntil;

		default:
			return false;
	}
}

/** Free text matches title, repo or author, mirroring the real engine's spirit. */
function matchesFreeText(item, text) {
	return (
		contains(item.subjectTitle, text) ||
		contains(item.repository?.fullName, text) ||
		contains(item.authorLogin, text)
	);
}

export function parseQuery(queryString) {
	return String(queryString ?? "")
		.split(/\s+/)
		.map((raw) => raw.trim())
		.filter(Boolean)
		.map((raw) => {
			const negated = raw.startsWith("-");
			const body = negated ? raw.slice(1) : raw;
			const separator = body.indexOf(":");

			if (separator === -1) {
				return { negated, field: null, values: [body] };
			}

			return {
				negated,
				field: body.slice(0, separator).toLowerCase(),
				values: body
					.slice(separator + 1)
					.split(",")
					.filter(Boolean),
			};
		});
}

function matchesTerm(item, term) {
	const matched = term.field
		? term.values.some((value) => matchesPair(item, term.field, value))
		: term.values.some((value) => matchesFreeText(item, value));
	return term.negated ? !matched : matched;
}

/**
 * Mirrors the implicit defaults in backend/internal/query/query.go: an empty
 * query means the inbox, a query that scopes itself with `in:` gets nothing
 * added, and anything else just hides muted items.
 */
function applyImplicitDefaults(item, terms) {
	if (terms.length === 0) {
		return isInInbox(item) && !item.muted;
	}
	if (terms.some((term) => term.field === "in")) {
		return true;
	}
	return !item.muted;
}

export function evaluateQuery(items, queryString) {
	const terms = parseQuery(queryString);
	return items.filter(
		(item) => applyImplicitDefaults(item, terms) && terms.every((term) => matchesTerm(item, term))
	);
}
