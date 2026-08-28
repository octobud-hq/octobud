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
 * Seed data for the stub server, shaped exactly like
 * backend/internal/models.Notification's JSON so the vendored
 * `fromBackendNotification` mapper works unchanged.
 *
 * The set is chosen to cover every visual branch NotificationRow can take:
 * each subject type and state, read/unread, starred, muted, archived, snoozed,
 * filtered, tagged (including more than the three the row shows inline), and a
 * title long enough to wrap in a narrow panel.
 */

const MINUTE = 60 * 1000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

const ago = (ms) => new Date(Date.now() - ms).toISOString();
const ahead = (ms) => new Date(Date.now() + ms).toISOString();

export const REPOSITORIES = {
	octobud: {
		id: 1,
		githubId: 900001,
		name: "octobud",
		fullName: "octobud-hq/octobud",
		ownerLogin: "octobud-hq",
		htmlUrl: "https://github.com/octobud-hq/octobud",
		private: false,
	},
	habitflow: {
		id: 2,
		githubId: 900002,
		name: "habitflow",
		fullName: "addonovan/habitflow",
		ownerLogin: "addonovan",
		htmlUrl: "https://github.com/addonovan/habitflow",
		private: true,
	},
	svelte: {
		id: 3,
		githubId: 900003,
		name: "svelte",
		fullName: "sveltejs/svelte",
		ownerLogin: "sveltejs",
		htmlUrl: "https://github.com/sveltejs/svelte",
		private: false,
	},
};

export const TAGS = [
	{ id: "tag-1", name: "urgent", slug: "urgent", color: "#ef4444", unreadCount: 2 },
	{ id: "tag-2", name: "backend", slug: "backend", color: "#3b82f6", unreadCount: 3 },
	{ id: "tag-3", name: "design", slug: "design", color: "#a855f7", unreadCount: 1 },
	{ id: "tag-4", name: "chore", slug: "chore", color: "#64748b", unreadCount: 0 },
	{ id: "tag-5", name: "flaky", slug: "flaky", color: "#f59e0b", unreadCount: 1 },
];

const tagBySlug = (slug) => TAGS.find((tag) => tag.slug === slug);

let nextId = 1;

function build({
	repo,
	subjectType,
	title,
	number,
	reason,
	author,
	state,
	merged = false,
	draft = false,
	isRead = false,
	starred = false,
	muted = false,
	archived = false,
	filtered = false,
	snoozedUntil = null,
	tags = [],
	updatedAt,
}) {
	const id = nextId++;
	const repository = REPOSITORIES[repo];
	const pathSegment = subjectType === "Issue" ? "issues" : "pull";
	const htmlUrl = number ? `${repository.htmlUrl}/${pathSegment}/${number}` : repository.htmlUrl;

	return {
		id,
		githubId: `stub-${String(id).padStart(4, "0")}`,
		repositoryId: repository.id,
		subjectType,
		subjectTitle: title,
		subjectUrl: number
			? `https://api.github.com/repos/${repository.fullName}/${pathSegment === "pull" ? "pulls" : "issues"}/${number}`
			: null,
		reason,
		archived,
		isRead,
		muted,
		starred,
		filtered,
		snoozedUntil,
		snoozedAt: snoozedUntil ? ago(2 * HOUR) : null,
		effectiveSortDate: updatedAt,
		githubUnread: !isRead,
		githubUpdatedAt: updatedAt,
		githubUrl: `https://api.github.com/notifications/threads/${id}`,
		importedAt: updatedAt,
		repository,
		subjectNumber: number ?? null,
		subjectState: state ?? null,
		subjectMerged: merged,
		subjectDraft: draft,
		authorLogin: author,
		tags: tags.map(tagBySlug).filter(Boolean),
		labels: [],
		assignees: [],
		reviewers: [],
		teamReviewers: [],
		subjectRaw: {
			title,
			html_url: htmlUrl,
			number: number ?? undefined,
			state: state ?? undefined,
			draft,
			merged,
			user: { login: author },
			updated_at: updatedAt,
		},
		subjectFetchedAt: updatedAt,
	};
}

export function createNotifications() {
	nextId = 1;
	return [
		build({
			repo: "octobud",
			subjectType: "PullRequest",
			title: "Add a browser side panel extension",
			number: 121,
			reason: "review_requested",
			author: "addonovan",
			state: "open",
			updatedAt: ago(18 * MINUTE),
			tags: ["urgent", "backend"],
		}),
		build({
			repo: "octobud",
			subjectType: "PullRequest",
			title: "Fix the flaky notification sync integration test that intermittently fails on CI",
			number: 119,
			reason: "author",
			author: "addonovan",
			state: "open",
			updatedAt: ago(2 * HOUR),
			tags: ["flaky"],
		}),
		build({
			repo: "octobud",
			subjectType: "PullRequest",
			title: "Bump modernc.org/sqlite from 1.50.0 to 1.50.1",
			number: 118,
			reason: "subscribed",
			author: "dependabot",
			state: "closed",
			merged: true,
			isRead: true,
			updatedAt: ago(6 * HOUR),
			tags: ["chore"],
		}),
		build({
			repo: "octobud",
			subjectType: "PullRequest",
			title: "Draft: rework the query parser error messages",
			number: 117,
			reason: "mention",
			author: "kbeattie",
			state: "open",
			draft: true,
			updatedAt: ago(9 * HOUR),
		}),
		build({
			repo: "octobud",
			subjectType: "Issue",
			title: "Views with an empty query should fall back to inbox semantics",
			number: 116,
			reason: "mention",
			author: "octocat",
			state: "open",
			updatedAt: ago(1 * DAY),
			tags: ["backend"],
		}),
		build({
			repo: "habitflow",
			subjectType: "Issue",
			title: "Streak resets an hour early in non-UTC time zones",
			number: 42,
			reason: "assign",
			author: "addonovan",
			state: "closed",
			isRead: true,
			updatedAt: ago(2 * DAY),
		}),
		build({
			repo: "habitflow",
			subjectType: "PullRequest",
			title: "Weekly review charts",
			number: 91,
			reason: "review_requested",
			author: "addonovan",
			state: "open",
			starred: true,
			updatedAt: ago(40 * MINUTE),
			tags: ["design"],
		}),
		build({
			repo: "habitflow",
			subjectType: "PullRequest",
			title: "Auto-miss overdue occurrences via Solid Queue",
			number: 88,
			reason: "subscribed",
			author: "addonovan",
			state: "closed",
			merged: true,
			archived: true,
			isRead: true,
			updatedAt: ago(3 * DAY),
		}),
		build({
			repo: "habitflow",
			subjectType: "PullRequest",
			title: "Push notification egress proxy",
			number: 100,
			reason: "author",
			author: "addonovan",
			state: "open",
			snoozedUntil: ahead(20 * HOUR),
			updatedAt: ago(5 * HOUR),
			tags: ["backend", "urgent"],
		}),
		build({
			repo: "svelte",
			subjectType: "Release",
			title: "svelte@5.55.7",
			number: null,
			reason: "subscribed",
			author: "sveltejs",
			updatedAt: ago(11 * HOUR),
		}),
		build({
			repo: "svelte",
			subjectType: "Discussion",
			title: "RFC: attachment syntax for actions",
			number: 14002,
			reason: "subscribed",
			author: "rich-harris",
			isRead: true,
			updatedAt: ago(4 * DAY),
		}),
		build({
			repo: "svelte",
			subjectType: "Issue",
			title: "Legacy component API deprecation warning is too noisy",
			number: 13990,
			reason: "subscribed",
			author: "octocat",
			state: "open",
			muted: true,
			updatedAt: ago(6 * DAY),
		}),
		build({
			repo: "octobud",
			subjectType: "CheckSuite",
			title: "CI workflow failed on main",
			number: null,
			reason: "ci_activity",
			author: "github-actions",
			state: "failure",
			updatedAt: ago(30 * MINUTE),
			tags: ["urgent"],
		}),
		build({
			repo: "octobud",
			subjectType: "PullRequest",
			title: "Tidy up the sidebar collapse animation",
			number: 112,
			reason: "team_mention",
			author: "kbeattie",
			state: "open",
			filtered: true,
			updatedAt: ago(3 * DAY),
			tags: ["design", "chore"],
		}),
		build({
			repo: "octobud",
			subjectType: "PullRequest",
			title: "Query engine: support parenthesised OR groups",
			number: 108,
			reason: "review_requested",
			author: "octocat",
			state: "open",
			isRead: true,
			updatedAt: ago(5 * DAY),
			tags: ["backend", "urgent", "design", "chore"],
		}),
		build({
			repo: "habitflow",
			subjectType: "Commit",
			title: "Fix N+1 in the agenda serializer",
			number: null,
			reason: "comment",
			author: "addonovan",
			updatedAt: ago(7 * HOUR),
		}),
	];
}

/**
 * Custom views alongside the five the backend synthesizes. Queries use the same
 * syntax the real engine accepts, within the stub evaluator's subset.
 */
export const SYSTEM_VIEWS = [
	{ slug: "inbox", name: "Inbox", icon: "inbox", query: "in:inbox", isDefault: true },
	{ slug: "everything", name: "Everything", icon: "infinity", query: "in:anywhere" },
	{ slug: "starred", name: "Starred", icon: "star", query: "is:starred in:anywhere" },
	{ slug: "snoozed", name: "Snoozed", icon: "snooze", query: "in:snoozed" },
	{ slug: "archive", name: "Archive", icon: "archive", query: "in:archive" },
];

export const USER_VIEWS = [
	{
		slug: "review-requests",
		name: "Needs my review",
		icon: "👀",
		description: "Open PRs waiting on me",
		query: "reason:review_requested is:unread",
	},
	{
		slug: "my-prs",
		name: "My pull requests",
		icon: "✨",
		query: "type:PullRequest author:addonovan",
	},
	{
		slug: "urgent",
		name: "Urgent",
		icon: "🔥",
		description: "Anything tagged urgent",
		query: "tags:urgent in:anywhere",
	},
	{
		slug: "ci",
		name: "CI activity",
		icon: "🤖",
		query: "reason:ci_activity in:anywhere",
	},
];
