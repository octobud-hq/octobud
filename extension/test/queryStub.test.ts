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

import { describe, expect, it } from "vitest";
// @ts-expect-error -- plain JS dev tooling, deliberately untyped
import { evaluateQuery, parseQuery } from "../dev/stub-server/query.mjs";
// @ts-expect-error -- plain JS dev tooling, deliberately untyped
import { createNotifications, SYSTEM_VIEWS, USER_VIEWS } from "../dev/stub-server/fixtures.mjs";

interface StubNotification {
	githubId: string;
	subjectTitle: string;
	subjectType: string;
	isRead: boolean;
	starred: boolean;
	muted: boolean;
	archived: boolean;
	filtered: boolean;
	snoozedUntil: string | null;
	reason: string;
	authorLogin: string;
	repository: { fullName: string };
	tags: { slug: string }[];
}

const items = createNotifications() as StubNotification[];
const run = (query: string) => evaluateQuery(items, query) as StubNotification[];

describe("parseQuery", () => {
	it("splits fields, values and negation", () => {
		expect(parseQuery("-is:read type:PullRequest,Issue octobud")).toEqual([
			{ negated: true, field: "is", values: ["read"] },
			{ negated: false, field: "type", values: ["PullRequest", "Issue"] },
			{ negated: false, field: null, values: ["octobud"] },
		]);
	});
});

describe("evaluateQuery", () => {
	it("treats an empty query as the inbox and hides muted items", () => {
		const result = run("");
		expect(result.length).toBeGreaterThan(0);
		expect(result.every((item) => !item.archived && !item.snoozedUntil && !item.filtered)).toBe(
			true
		);
		expect(result.some((item) => item.muted)).toBe(false);
	});

	it("does not add implicit defaults when the query scopes itself with in:", () => {
		expect(run("in:anywhere")).toHaveLength(items.length);
	});

	it("filters on read state", () => {
		expect(run("is:unread in:anywhere").every((item) => !item.isRead)).toBe(true);
		expect(run("is:read in:anywhere").every((item) => item.isRead)).toBe(true);
	});

	it("scopes in:archive and in:snoozed to their own buckets", () => {
		expect(run("in:archive").every((item) => item.archived)).toBe(true);
		expect(run("in:snoozed").every((item) => Boolean(item.snoozedUntil))).toBe(true);
	});

	it("normalizes PullRequest against the wire's subject type", () => {
		const byType = run("type:PullRequest in:anywhere");
		expect(byType.length).toBeGreaterThan(0);
		expect(byType.every((item) => item.subjectType === "PullRequest")).toBe(true);
		expect(run("type:pull_request in:anywhere")).toHaveLength(byType.length);
	});

	it("ORs comma-separated values within one field", () => {
		const combined = run("type:Issue,Release in:anywhere");
		expect(combined.length).toBe(
			run("type:Issue in:anywhere").length + run("type:Release in:anywhere").length
		);
	});

	it("negates with a leading dash", () => {
		expect(run("in:anywhere -type:PullRequest").every((i) => i.subjectType !== "PullRequest")).toBe(
			true
		);
	});

	it("matches tags by slug", () => {
		const urgent = run("tags:urgent in:anywhere");
		expect(urgent.length).toBeGreaterThan(0);
		expect(urgent.every((item) => item.tags.some((tag) => tag.slug === "urgent"))).toBe(true);
	});

	it("accepts the starred:true form as well as is:starred", () => {
		expect(run("starred:true in:anywhere")).toEqual(run("is:starred in:anywhere"));
	});

	it("matches free text against title, repo and author", () => {
		expect(run("flaky in:anywhere").some((i) => i.subjectTitle.includes("flaky"))).toBe(true);
		expect(
			run("habitflow in:anywhere").every((i) => i.repository.fullName.includes("habitflow"))
		).toBe(true);
	});

	it("gives every seeded view a non-empty, distinct result set", () => {
		const views = [...SYSTEM_VIEWS, ...USER_VIEWS] as { slug: string; query: string }[];
		const sizes = new Map<string, number>();

		for (const view of views) {
			const result = run(view.query);
			expect(result.length, `view "${view.slug}" returned nothing`).toBeGreaterThan(0);
			sizes.set(view.slug, result.length);
		}

		// The panel is only interesting if switching views visibly changes things.
		expect(new Set(sizes.values()).size).toBeGreaterThan(1);
	});
});
