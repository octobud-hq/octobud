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

export interface RuleActions {
	skipInbox: boolean;
	markRead?: boolean;
	star?: boolean;
	archive?: boolean;
	mute?: boolean;
	assignTags?: string[]; // Tag IDs as strings
	removeTags?: string[]; // Tag IDs as strings
}

export interface Rule {
	id: string;
	name: string;
	description?: string;
	query: string;
	viewId?: string;
	actions: RuleActions;
	enabled: boolean;
	displayOrder: number;
	createdAt: string;
	updatedAt: string;
}

interface RulesResponse {
	rules: Rule[];
}

interface RuleResponse {
	rule: Rule;
}

interface CreateRuleRequest {
	name: string;
	description?: string;
	query?: string; // Either query or viewId must be provided (not both)
	viewId?: string;
	actions: RuleActions;
	enabled?: boolean;
	applyToExisting?: boolean;
}

interface UpdateRuleRequest {
	name?: string;
	description?: string;
	query?: string;
	viewId?: string;
	actions?: RuleActions;
	enabled?: boolean;
}

import { fetchWithAuth, buildApiUrl } from "./fetch";

export async function fetchRules(fetchImpl: typeof fetch = fetch): Promise<Rule[]> {
	const response = await fetchWithAuth("/api/rules", {}, fetchImpl);
	if (!response.ok) {
		throw new Error(`Failed to fetch rules: ${response.statusText}`);
	}
	const data: RulesResponse = await response.json();
	return data.rules;
}

export async function createRule(
	data: CreateRuleRequest,
	fetchImpl: typeof fetch = fetch
): Promise<Rule> {
	const response = await fetchWithAuth(
		"/api/rules",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(data),
		},
		fetchImpl
	);
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(`Failed to create rule: ${errorText || response.statusText}`);
	}
	const result: RuleResponse = await response.json();
	return result.rule;
}

export async function updateRule(
	id: string,
	data: UpdateRuleRequest,
	fetchImpl: typeof fetch = fetch
): Promise<Rule> {
	const response = await fetchWithAuth(
		`/api/rules/${id}`,
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(data),
		},
		fetchImpl
	);
	if (!response.ok) {
		const errorText = await response.text();
		throw new Error(`Failed to update rule: ${errorText || response.statusText}`);
	}
	const result: RuleResponse = await response.json();
	return result.rule;
}

export async function deleteRule(id: string, fetchImpl: typeof fetch = fetch): Promise<void> {
	const response = await fetchWithAuth(
		`/api/rules/${id}`,
		{
			method: "DELETE",
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to delete rule: ${response.statusText}`);
	}
}

export async function reorderRules(
	ruleIds: string[],
	fetchImpl: typeof fetch = fetch
): Promise<Rule[]> {
	const response = await fetchWithAuth(
		"/api/rules/reorder",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ ruleIds }),
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to reorder rules: ${response.statusText}`);
	}
	const data: RulesResponse = await response.json();
	return data.rules;
}
