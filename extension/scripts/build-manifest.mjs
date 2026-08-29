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
 * Assembles dist/chrome and dist/firefox from the Vite output in dist/build.
 *
 * The two browsers disagree about how a side panel is declared — Chrome uses
 * `side_panel` plus a service worker, Firefox uses `sidebar_action` plus an
 * event page and needs a stable extension id — so each target gets a manifest
 * merged from manifest/base.json and its own overrides. Shipping one manifest
 * containing both sets of keys would work, but each browser would warn about the
 * other's, which is noise every reviewer has to re-diagnose.
 *
 * The version comes from package.json, so that is the single place to bump.
 */

import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const root = new URL("../", import.meta.url);
const dir = (relative) => fileURLToPath(new URL(relative, root));
const readJson = (relative) => JSON.parse(readFileSync(dir(relative), "utf8"));

const TARGETS = ["chrome", "firefox"];

const { version } = readJson("package.json");
const base = readJson("manifest/base.json");

if (!/^\d+\.\d+\.\d+$/.test(version)) {
	// Chrome and Firefox both reject SemVer pre-release suffixes in a manifest
	// version, so catch it here rather than at load time.
	throw new Error(
		`package.json version "${version}" is not a plain X.Y.Z, which extension manifests require.`
	);
}

for (const target of TARGETS) {
	const outDir = dir(`dist/${target}`);
	rmSync(outDir, { recursive: true, force: true });
	mkdirSync(outDir, { recursive: true });

	cpSync(dir("dist/build"), outDir, { recursive: true });

	// Shallow merge: a key present in the target file replaces the base entirely.
	// That is deliberate for `permissions` — Chrome needs "sidePanel" for
	// setPanelBehavior, and Firefox rejects manifests listing permissions it does
	// not know, so the two lists have to differ rather than be concatenated.
	const manifest = { ...base, ...readJson(`manifest/${target}.json`), version };
	writeFileSync(`${outDir}/manifest.json`, `${JSON.stringify(manifest, null, "\t")}\n`, "utf8");

	console.log(`Built dist/${target} (manifest v${version})`);
}
