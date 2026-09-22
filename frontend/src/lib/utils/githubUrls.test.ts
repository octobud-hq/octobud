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

import { describe, it, expect } from "vitest";
import { constructGitHubHtmlUrl } from "./githubUrls";

describe("constructGitHubHtmlUrl", () => {
	it("builds pull request and issue URLs from numbers", () => {
		expect(constructGitHubHtmlUrl("cli/cli", "PullRequest", undefined, 42)).toBe(
			"https://github.com/cli/cli/pull/42"
		);
		expect(constructGitHubHtmlUrl("cli/cli", "Issue", undefined, 7)).toBe(
			"https://github.com/cli/cli/issues/7"
		);
	});

	it("links advisory credits to the repository's advisories page (no subject URL)", () => {
		expect(constructGitHubHtmlUrl("cli/cli", "AdvisoryCredit")).toBe(
			"https://github.com/cli/cli/security/advisories"
		);
	});

	it("links repository advisories to the advisory when the subject URL carries a GHSA id", () => {
		expect(
			constructGitHubHtmlUrl(
				"cli/cli",
				"RepositoryAdvisory",
				"https://api.github.com/repos/cli/cli/security-advisories/GHSA-ABCD-ef12-gh34"
			)
		).toBe("https://github.com/cli/cli/security/advisories/ghsa-abcd-ef12-gh34");
		expect(constructGitHubHtmlUrl("cli/cli", "repository_advisory")).toBe(
			"https://github.com/cli/cli/security/advisories"
		);
	});

	it("returns null without a repository", () => {
		expect(constructGitHubHtmlUrl("", "AdvisoryCredit")).toBeNull();
	});
});
