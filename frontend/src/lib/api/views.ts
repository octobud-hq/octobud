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

import {
	fetchWithAuth,
	buildApiUrl,
	ApiUnreachableError,
	isNetworkError,
	isProxyConnectionError,
} from "./fetch";
import { DEFAULT_VIEW_ICON } from "$lib/utils/viewIcons";
import type { NotificationView, NotificationViewInput } from "./types";

const cloneView = (view: NotificationView): NotificationView => ({
	...view,
	slug: view.slug,
	query: view.query || "",
});

const ensureQuery = (query: string): string => {
	const trimmed = query.trim();
	if (!trimmed) {
		throw new Error("A view must include a query");
	}
	return trimmed;
};

export async function fetchViews(fetchImpl?: typeof fetch): Promise<NotificationView[]> {
	try {
		const response = await fetchWithAuth("/api/views", {}, fetchImpl);
		if (!response.ok) {
			// Check if this is a proxy connection error
			let responseText = "";
			let isValidJson = false;
			try {
				responseText = await response.text();
				JSON.parse(responseText);
				isValidJson = true;
			} catch {
				// Ignore parse errors
			}
			// Proxy errors: empty response, non-JSON response, or response with connection error keywords
			const isProxyError =
				(response.status === 500 || response.status === 502 || response.status === 503) &&
				(!responseText || !isValidJson || isProxyConnectionError(responseText));
			if (isProxyError) {
				throw new ApiUnreachableError();
			}
			throw new Error(`Failed to load views (${response.status})`);
		}
		const payload: { views: NotificationView[] } = await response.json();
		const data = (payload?.views ?? []).map(cloneView);
		return data.map(cloneView);
	} catch (error) {
		// Re-throw network errors so they can be detected at the layout level
		if (error instanceof ApiUnreachableError || isNetworkError(error)) {
			throw error;
		}
		// Return empty array if API is not available for other reasons
		console.error("Failed to fetch views:", error);
		return [];
	}
}

export async function createView(
	input: NotificationViewInput,
	fetchImpl?: typeof fetch
): Promise<NotificationView> {
	const query = ensureQuery(input.query);
	const response = await fetchWithAuth(
		"/api/views",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ ...input, query }),
		},
		fetchImpl
	);

	if (!response.ok) {
		let errorMessage = `Failed to create view (${response.status})`;
		try {
			const errorData: { error?: string } = await response.json();
			if (errorData.error) {
				errorMessage = errorData.error;
			}
		} catch {
			// If parsing fails, use the default error message
		}
		const error: any = new Error(errorMessage);
		error.statusCode = response.status;
		throw error;
	}

	const payload: { view: NotificationView } = await response.json();
	const created = cloneView(payload.view);
	return cloneView(created);
}

export interface UpdateViewInput extends NotificationViewInput {
	id: string;
}

export async function updateView(
	id: string,
	input: NotificationViewInput,
	fetchImpl?: typeof fetch
): Promise<NotificationView> {
	const query = ensureQuery(input.query);
	const response = await fetchWithAuth(
		`/api/views/${encodeURIComponent(id)}`,
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ ...input, query }),
		},
		fetchImpl
	);

	if (!response.ok) {
		let errorMessage = `Failed to update view (${response.status})`;
		try {
			const errorData: { error?: string } = await response.json();
			if (errorData.error) {
				errorMessage = errorData.error;
			}
		} catch {
			// If parsing fails, use the default error message
		}
		const error: any = new Error(errorMessage);
		error.statusCode = response.status;
		throw error;
	}

	const payload: { view: NotificationView } = await response.json();
	const updated = cloneView(payload.view);
	return cloneView(updated);
}

export async function deleteView(
	id: string,
	force: boolean = false,
	fetchImpl?: typeof fetch
): Promise<void> {
	const url = `/api/views/${encodeURIComponent(id)}${force ? "?force=true" : ""}`;
	const response = await fetchWithAuth(
		url,
		{
			method: "DELETE",
		},
		fetchImpl
	);
	if (!response.ok) {
		if (response.status === 409) {
			// Conflict - view has linked rules
			const errorData = await response.json();
			const error: any = new Error("This view has linked rules that will also be deleted");
			error.linkedRuleCount = errorData.linkedRuleCount;
			error.statusCode = 409;
			throw error;
		}
		throw new Error(`Failed to delete view (${response.status})`);
	}
}

export async function reorderViews(
	viewIds: string[],
	fetchImpl?: typeof fetch
): Promise<NotificationView[]> {
	const response = await fetchWithAuth(
		"/api/views/reorder",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ viewIds }),
		},
		fetchImpl
	);

	if (!response.ok) {
		throw new Error(`Failed to reorder views (${response.status})`);
	}

	const payload: { views: NotificationView[] } = await response.json();
	const data = (payload?.views ?? []).map(cloneView);
	return data.map(cloneView);
}
