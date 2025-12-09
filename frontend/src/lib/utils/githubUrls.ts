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
 * Constructs GitHub web URLs from API URLs or subject metadata as a fallback
 * when html_url is not available in the subject data.
 */

/**
 * Constructs a GitHub HTML URL from available notification data.
 *
 * @param repoFullName - Full repository name (e.g., "owner/repo")
 * @param subjectType - Type of the notification subject (e.g., "PullRequest", "Issue")
 * @param subjectUrl - Optional API URL from the notification subject
 * @param number - Optional PR/Issue number
 * @returns The constructed GitHub web URL, or null if unable to construct
 */
export function constructGitHubHtmlUrl(
	repoFullName: string,
	subjectType: string,
	subjectUrl?: string,
	number?: number
): string | null {
	if (!repoFullName) {
		return null;
	}

	// Normalize subject type to handle variations
	const normalizedType = subjectType.toLowerCase();

	// Try to extract information from API URL if provided
	if (subjectUrl) {
		const extractedUrl = extractFromApiUrl(subjectUrl, repoFullName);
		if (extractedUrl) {
			return extractedUrl;
		}
	}

	// If we have a number, construct URL for Issue or PullRequest
	if (number !== undefined) {
		if (normalizedType === "pullrequest" || normalizedType === "pull_request") {
			return `https://github.com/${repoFullName}/pull/${number}`;
		}
		if (normalizedType === "issue") {
			return `https://github.com/${repoFullName}/issues/${number}`;
		}
		if (normalizedType === "discussion") {
			return `https://github.com/${repoFullName}/discussions/${number}`;
		}
	}

	// For other types, try to construct based on type and API URL patterns
	if (subjectUrl) {
		return constructFromApiUrlPattern(repoFullName, normalizedType, subjectUrl);
	}

	// Handle types that don't need a subjectUrl to construct a URL
	// Check runs/suites -> actions page
	if (
		normalizedType === "checkrun" ||
		normalizedType === "check_run" ||
		normalizedType === "checksuite" ||
		normalizedType === "check_suite"
	) {
		return `https://github.com/${repoFullName}/actions`;
	}

	// Vulnerability alerts -> security/dependabot page
	if (
		normalizedType === "repositoryvulnerabilityalert" ||
		normalizedType === "repository_vulnerability_alert"
	) {
		return `https://github.com/${repoFullName}/security/dependabot`;
	}

	return null;
}

/**
 * Attempts to extract and construct a GitHub web URL from an API URL.
 */
function extractFromApiUrl(apiUrl: string, repoFullName: string): string | null {
	try {
		// Pattern: https://api.github.com/repos/{owner}/{repo}/issues/{number}
		const issueMatch = apiUrl.match(/\/repos\/[^\/]+\/[^\/]+\/issues\/(\d+)/);
		if (issueMatch) {
			return `https://github.com/${repoFullName}/issues/${issueMatch[1]}`;
		}

		// Pattern: https://api.github.com/repos/{owner}/{repo}/pulls/{number}
		const pullMatch = apiUrl.match(/\/repos\/[^\/]+\/[^\/]+\/pulls\/(\d+)/);
		if (pullMatch) {
			return `https://github.com/${repoFullName}/pull/${pullMatch[1]}`;
		}

		// Pattern: https://api.github.com/repos/{owner}/{repo}/commits/{sha}
		const commitMatch = apiUrl.match(/\/repos\/[^\/]+\/[^\/]+\/commits\/([a-f0-9]+)/);
		if (commitMatch) {
			return `https://github.com/${repoFullName}/commit/${commitMatch[1]}`;
		}

		// Pattern: https://api.github.com/repos/{owner}/{repo}/releases/{id}
		const releaseMatch = apiUrl.match(/\/repos\/[^\/]+\/[^\/]+\/releases\/(\d+)/);
		if (releaseMatch) {
			// For releases, we need the tag name, not the ID
			// This is a limitation - we'd need to fetch the release to get the tag
			// For now, just link to releases page
			return `https://github.com/${repoFullName}/releases`;
		}

		// Pattern: https://api.github.com/repos/{owner}/{repo}/discussions/{number}
		const discussionMatch = apiUrl.match(/\/repos\/[^\/]+\/[^\/]+\/discussions\/(\d+)/);
		if (discussionMatch) {
			return `https://github.com/${repoFullName}/discussions/${discussionMatch[1]}`;
		}
	} catch (error) {
		return null;
	}

	return null;
}

/**
 * Constructs a URL based on subject type and API URL patterns.
 */
function constructFromApiUrlPattern(
	repoFullName: string,
	subjectType: string,
	apiUrl: string
): string | null {
	// For commit types, try to extract SHA
	if (subjectType === "commit") {
		const shaMatch = apiUrl.match(/\/commits\/([a-f0-9]{40}|[a-f0-9]{7,})/);
		if (shaMatch) {
			return `https://github.com/${repoFullName}/commit/${shaMatch[1]}`;
		}
	}

	// For release types
	if (subjectType === "release") {
		// Try to extract tag from API URL
		const tagMatch = apiUrl.match(/\/releases\/tags\/([^\/]+)/);
		if (tagMatch) {
			return `https://github.com/${repoFullName}/releases/tag/${tagMatch[1]}`;
		}
		// Fallback to releases page
		return `https://github.com/${repoFullName}/releases`;
	}

	// For vulnerability alerts
	if (
		subjectType === "repositoryvulnerabilityalert" ||
		subjectType === "repository_vulnerability_alert"
	) {
		return `https://github.com/${repoFullName}/security/dependabot`;
	}

	// For check runs/suites
	if (
		subjectType === "checkrun" ||
		subjectType === "check_run" ||
		subjectType === "checksuite" ||
		subjectType === "check_suite"
	) {
		return `https://github.com/${repoFullName}/actions`;
	}

	return null;
}
