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

// Runs as a service worker on Chrome and an event page on Firefox. Kept as
// dependency-free classic JavaScript in public/ so one file works as both
// without asking Rollup to emit two module formats from a single build.
//
// The only job is making the toolbar button open the panel. Firefox's
// sidebar_action does that natively; Chrome needs to be told once.

(function () {
	var ext = globalThis.chrome || globalThis.browser;

	if (ext && ext.sidePanel && typeof ext.sidePanel.setPanelBehavior === "function") {
		ext.sidePanel.setPanelBehavior({ openPanelOnActionClick: true }).catch(function (error) {
			console.error("[octobud] could not bind the toolbar button to the side panel:", error);
		});
	}
})();
