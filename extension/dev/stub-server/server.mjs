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
 * Stub of the Octobud backend's HTTP API, for developing the panel without a
 * running Go binary (and without a GitHub account).
 *
 * State is in-memory and mutable, so actions taken in the panel really do change
 * what subsequent requests return. Restart the process to reset.
 *
 * Scope and limits are documented in ./README.md.
 */

import { createServer } from "node:http";
import { createNotifications, SYSTEM_VIEWS, TAGS, USER_VIEWS } from "./fixtures.mjs";
import { evaluateQuery } from "./query.mjs";

const PORT = Number(process.env.PORT ?? 8808);
const DEFAULT_PAGE_SIZE = 30;

let notifications = createNotifications();

/**
 * The real server allows `http://localhost:*` and `http://127.0.0.1:*` only, and
 * relies on extension host permissions for extension origins. The stub echoes
 * whatever Origin it is given so that a panel loaded from either a
 * chrome-extension:// or moz-extension:// origin can talk to it even if host
 * permissions are misconfigured — that keeps a CORS mistake from being
 * misdiagnosed as a bug in the panel.
 */
function corsHeaders(request) {
	return {
		"Access-Control-Allow-Origin": request.headers.origin ?? "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		"Access-Control-Allow-Headers": "Accept, Authorization, Content-Type, X-CSRF-Token",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Max-Age": "300",
		Vary: "Origin",
	};
}

function send(request, response, status, body) {
	const payload = body === undefined ? "" : JSON.stringify(body);
	response.writeHead(status, {
		"Content-Type": "application/json",
		"Content-Length": Buffer.byteLength(payload),
		...corsHeaders(request),
	});
	response.end(payload);
}

async function readJsonBody(request) {
	const chunks = [];
	for await (const chunk of request) chunks.push(chunk);
	if (chunks.length === 0) return {};
	try {
		return JSON.parse(Buffer.concat(chunks).toString("utf8"));
	} catch {
		return {};
	}
}

function unreadCountFor(query) {
	return evaluateQuery(notifications, query).filter((item) => !item.isRead).length;
}

function buildViews() {
	const system = SYSTEM_VIEWS.map((view, index) => ({
		id: view.slug,
		name: view.name,
		slug: view.slug,
		icon: view.icon,
		description: view.description,
		isDefault: Boolean(view.isDefault),
		systemView: true,
		query: view.query,
		unreadCount: unreadCountFor(view.query),
		displayOrder: index,
	}));

	const user = USER_VIEWS.map((view, index) => ({
		id: `view-${view.slug}`,
		name: view.name,
		slug: view.slug,
		icon: view.icon,
		description: view.description,
		isDefault: false,
		systemView: false,
		query: view.query,
		unreadCount: unreadCountFor(view.query),
		displayOrder: index,
	}));

	return [...system, ...user];
}

function findByGithubId(githubId) {
	return notifications.find((item) => item.githubId === githubId);
}

/** Field mutations keyed by the action segment of the URL. */
const ACTIONS = {
	"mark-read": (item) => ({ isRead: true, githubUnread: false }),
	"mark-unread": (item) => ({ isRead: false, githubUnread: true }),
	archive: (item) => ({ archived: true }),
	unarchive: (item) => ({ archived: false }),
	mute: (item) => ({ muted: true }),
	unmute: (item) => ({ muted: false }),
	star: (item) => ({ starred: true }),
	unstar: (item) => ({ starred: false }),
	unfilter: (item) => ({ filtered: false }),
	snooze: (item, body) => ({
		snoozedUntil: body.snoozedUntil ?? null,
		snoozedAt: new Date().toISOString(),
	}),
	unsnooze: (item) => ({ snoozedUntil: null, snoozedAt: null }),
};

function applyTagChange(githubIds, tagId, add) {
	const tag = TAGS.find((candidate) => candidate.id === tagId);
	if (!tag) return 0;

	let count = 0;
	for (const githubId of githubIds) {
		const item = findByGithubId(githubId);
		if (!item) continue;
		const has = item.tags.some((candidate) => candidate.id === tagId);
		if (add && !has) {
			item.tags = [...item.tags, tag];
			count += 1;
		} else if (!add && has) {
			item.tags = item.tags.filter((candidate) => candidate.id !== tagId);
			count += 1;
		}
	}
	return count;
}

async function handle(request, response, url) {
	const { pathname } = url;

	if (pathname === "/healthz") {
		return send(request, response, 200, { status: "ok" });
	}

	if (pathname === "/api/views" && request.method === "GET") {
		return send(request, response, 200, { views: buildViews() });
	}

	if (pathname === "/api/tags" && request.method === "GET") {
		return send(request, response, 200, { tags: TAGS });
	}

	if (pathname === "/api/notifications" && request.method === "GET") {
		const query = url.searchParams.get("query") ?? "";
		const page = Math.max(1, Number(url.searchParams.get("page") ?? 1));
		const pageSize = Math.max(1, Number(url.searchParams.get("pageSize") ?? DEFAULT_PAGE_SIZE));

		const matched = evaluateQuery(notifications, query).sort(
			(a, b) => Date.parse(b.effectiveSortDate) - Date.parse(a.effectiveSortDate)
		);
		const start = (page - 1) * pageSize;

		return send(request, response, 200, {
			notifications: matched.slice(start, start + pageSize),
			total: matched.length,
			page,
			pageSize,
		});
	}

	const bulkTag = pathname.match(/^\/api\/notifications\/bulk\/(assign-tag|remove-tag)$/);
	if (bulkTag && request.method === "POST") {
		const body = await readJsonBody(request);
		const count = applyTagChange(body.githubIds ?? [], body.tagId, bulkTag[1] === "assign-tag");
		return send(request, response, 200, { count });
	}

	const action = pathname.match(/^\/api\/notifications\/([^/]+)\/([a-z-]+)$/);
	if (action && request.method === "POST") {
		const [, rawGithubId, actionName] = action;
		const mutate = ACTIONS[actionName];
		if (!mutate) {
			return send(request, response, 404, { error: `Unknown action: ${actionName}` });
		}

		const item = findByGithubId(decodeURIComponent(rawGithubId));
		if (!item) {
			return send(request, response, 404, { error: "Notification not found" });
		}

		const body = await readJsonBody(request);
		Object.assign(item, mutate(item, body));
		return send(request, response, 200, { notification: item });
	}

	return send(request, response, 404, { error: `No stub route for ${request.method} ${pathname}` });
}

const server = createServer((request, response) => {
	const url = new URL(request.url, `http://localhost:${PORT}`);

	if (request.method === "OPTIONS") {
		response.writeHead(204, corsHeaders(request));
		return response.end();
	}

	handle(request, response, url).catch((error) => {
		console.error(`[stub] ${request.method} ${url.pathname} failed:`, error);
		send(request, response, 500, { error: String(error) });
	});
});

server.listen(PORT, "127.0.0.1", () => {
	console.log(`[stub] Octobud API stub listening on http://localhost:${PORT}`);
	console.log(`[stub] ${notifications.length} notifications, ${buildViews().length} views`);
});
