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
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

/**
 * The extension vendors the backend API contract and the row's formatting
 * helpers from frontend/. If someone changes one side only, the panel starts
 * lying about the shape of the data it renders. Fail loudly here instead.
 *
 * See src/lib/VENDORED.md for the list and the rationale.
 */
const VENDORED_PATHS = [
	"api/types.ts",
	"api/views.ts",
	"api/notifications.ts",
	"api/tags.ts",
	"utils/time.ts",
	"utils/notificationHelpers.ts",
	"utils/notificationIcons.ts",
	"utils/viewIcons.ts",
	"utils/githubUrls.ts",
	"utils/snoozeFormat.ts",
	"utils/archiveIcons.ts",
	"utils/muteIcons.ts",
	"components/timeline/ViewDropdown.svelte",
	"components/shared/SnoozeDropdown.svelte",
	"components/shared/TagDropdown.svelte",
	"components/shared/CompactPagination.svelte",
];

function read(relative: string): string {
	return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), "utf8");
}

describe("vendored modules", () => {
	it.each(VENDORED_PATHS)("%s matches frontend/src/lib", (path) => {
		const vendored = read(`../src/lib/${path}`);
		const original = read(`../../frontend/src/lib/${path}`);

		expect(
			vendored,
			`extension/src/lib/${path} has drifted from frontend/src/lib/${path}. ` +
				`Re-copy it, or update src/lib/VENDORED.md if it should no longer be vendored.`
		).toBe(original);
	});

	it("does not vendor fetch.ts, which is deliberately adapted", () => {
		const vendored = read("../src/lib/api/fetch.ts");
		const original = read("../../frontend/src/lib/api/fetch.ts");

		expect(vendored).not.toBe(original);
		expect(vendored).toContain("getBackendUrl");
	});
});
