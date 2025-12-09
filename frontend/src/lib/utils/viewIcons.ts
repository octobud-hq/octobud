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

const ICON_NAME_TO_EMOJI: Record<string, string> = {
	filter: "✨",
	clock: "⏳",
	robot: "🤖",
	inbox: "📬",
	check: "✅",
	"user-circle": "👤",
};

export const DEFAULT_VIEW_ICON = "✨";

export const VIEW_ICON_SUGGESTIONS = [
	"✨",
	"📬",
	"⭐️",
	"⚡️",
	"🛠️",
	"🤖",
	"📝",
	"✅",
	"📌",
	"🔥",
	"⏳",
	"🌱",
	"📦",
] as const;

export function normalizeViewIcon(icon?: string | null): string {
	const trimmed = icon?.trim();
	if (!trimmed) {
		return DEFAULT_VIEW_ICON;
	}
	return ICON_NAME_TO_EMOJI[trimmed] ?? trimmed;
}

export function cleanIconInput(icon?: string | null): string | undefined {
	const trimmed = icon?.trim();
	return trimmed ? trimmed : undefined;
}
