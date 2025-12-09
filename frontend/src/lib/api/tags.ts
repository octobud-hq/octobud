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

export interface Tag {
	id: string;
	name: string;
	slug: string;
	color?: string;
	description?: string;
	unreadCount?: number;
}

export interface TagInput {
	name: string;
	color?: string;
	description?: string;
}

interface TagsResponse {
	tags: Tag[];
}

interface TagEnvelope {
	tag: Tag;
}

import { fetchWithAuth, buildApiUrl, ApiUnreachableError, isProxyConnectionError } from "./fetch";

export async function fetchTags(fetchImpl: typeof fetch = fetch): Promise<Tag[]> {
	const response = await fetchWithAuth("/api/tags", {}, fetchImpl);
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
		throw new Error(`Failed to fetch tags: ${response.statusText}`);
	}
	const data: TagsResponse = await response.json();
	return data.tags;
}

export async function createTag(input: TagInput, fetchImpl: typeof fetch = fetch): Promise<Tag> {
	const response = await fetchWithAuth(
		"/api/tags",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(input),
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to create tag: ${response.statusText}`);
	}
	const data: TagEnvelope = await response.json();
	return data.tag;
}

export async function updateTag(
	id: string,
	input: TagInput,
	fetchImpl: typeof fetch = fetch
): Promise<Tag> {
	const response = await fetchWithAuth(
		`/api/tags/${id}`,
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(input),
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to update tag: ${response.statusText}`);
	}
	const data: TagEnvelope = await response.json();
	return data.tag;
}

export async function deleteTag(id: string, fetchImpl: typeof fetch = fetch): Promise<void> {
	const response = await fetchWithAuth(
		`/api/tags/${id}`,
		{
			method: "DELETE",
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to delete tag: ${response.statusText}`);
	}
}

export async function assignTagToNotification(
	githubId: string,
	tagId: string,
	fetchImpl: typeof fetch = fetch
): Promise<import("./types").Notification> {
	const encodedGithubId = encodeURIComponent(githubId);
	const response = await fetchWithAuth(
		`/api/notifications/${encodedGithubId}/tags`,
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ tagId: tagId }),
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to assign tag: ${response.statusText}`);
	}
	const data = await response.json();
	return data.notification;
}

export async function assignTagToNotificationByName(
	githubId: string,
	tagName: string,
	fetchImpl: typeof fetch = fetch
): Promise<import("./types").Notification> {
	const encodedGithubId = encodeURIComponent(githubId);
	const response = await fetchWithAuth(
		`/api/notifications/${encodedGithubId}/tags-by-name`,
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ tagName }),
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to assign tag: ${response.statusText}`);
	}
	const data = await response.json();
	return data.notification;
}

export async function removeTagFromNotification(
	githubId: string,
	tagId: string,
	fetchImpl: typeof fetch = fetch
): Promise<import("./types").Notification> {
	const encodedGithubId = encodeURIComponent(githubId);
	const response = await fetchWithAuth(
		`/api/notifications/${encodedGithubId}/tags/${tagId}`,
		{
			method: "DELETE",
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to remove tag: ${response.statusText}`);
	}
	const data = await response.json();
	return data.notification;
}

export async function reorderTags(
	tagIds: string[],
	fetchImpl: typeof fetch = fetch
): Promise<Tag[]> {
	const response = await fetchWithAuth(
		"/api/tags/reorder",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ tagIds }),
		},
		fetchImpl
	);
	if (!response.ok) {
		throw new Error(`Failed to reorder tags: ${response.statusText}`);
	}
	const data: TagsResponse = await response.json();
	return data.tags;
}
