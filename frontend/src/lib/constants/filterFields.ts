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
 * Filter field configuration for notification queries
 */

export interface FilterFieldConfig {
	value: string;
	description: string;
	valueSuggestions?: string[];
}

/**
 * All available filter fields for notifications
 */
export const FILTER_FIELDS: FilterFieldConfig[] = [
	{
		value: "in",
		description: "View context (inbox, archive, snoozed, filtered, anywhere)",
		valueSuggestions: ["inbox", "archive", "snoozed", "filtered", "anywhere"],
	},
	{
		value: "is",
		description: "Special status flag",
		valueSuggestions: ["read", "unread", "muted"],
	},
	{
		value: "reason",
		description: "Notification reason",
		valueSuggestions: [
			"assign",
			"author",
			"comment",
			"invitation",
			"manual",
			"mention",
			"review_requested",
			"security_alert",
			"state_change",
			"subscribed",
			"team_mention",
		],
	},
	{
		value: "type",
		description: "Subject type (issue, pullrequest, etc.)",
		valueSuggestions: ["issue", "pullrequest", "release", "discussion", "commit"],
	},
	{
		value: "repo",
		description: "Repository full name",
	},
	{
		value: "state",
		description: "Issue or PR state (open, closed)",
		valueSuggestions: ["open", "closed"],
	},
	{
		value: "author",
		description: "Author login",
	},
	{
		value: "org",
		description: "Organization name",
	},
	{
		value: "tags",
		description: "Tag slug (supports partial matching)",
	},
];

/**
 * Valid filter field names (for validation)
 */
export const VALID_FILTER_FIELDS = FILTER_FIELDS.map((f) => f.value);

/**
 * Get value suggestions for a specific field
 */
export function getValueSuggestionsForField(field: string): string[] {
	const fieldConfig = FILTER_FIELDS.find((f) => f.value === field.toLowerCase());
	return fieldConfig?.valueSuggestions ?? [];
}

/**
 * Check if a field name is valid
 */
export function isValidFilterField(field: string): boolean {
	return VALID_FILTER_FIELDS.includes(field.toLowerCase());
}
