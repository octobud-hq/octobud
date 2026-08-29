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

// Chrome's MV3 service worker. Its only job is making the toolbar button open
// the side panel, which Chrome does not do by default.
//
// Firefox does not load this: `sidebar_action` wires its toolbar button to the
// panel natively, so the Firefox manifest omits `background` entirely rather
// than shipping an event page that would do nothing.
//
// Kept as dependency-free classic JavaScript in public/ so it is copied through
// verbatim instead of asking Rollup to emit a second module format.

(function () {
	var ext = globalThis.chrome || globalThis.browser;

	if (ext && ext.sidePanel && typeof ext.sidePanel.setPanelBehavior === "function") {
		ext.sidePanel.setPanelBehavior({ openPanelOnActionClick: true }).catch(function (error) {
			console.error("[octobud] could not bind the toolbar button to the side panel:", error);
		});
	}
})();
