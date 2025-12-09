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

import { derived, writable } from "svelte/store";
import type { Readable, Writable } from "svelte/store";

export type CommandPaletteCommandId = "view" | "shortcuts" | "bulk" | "query";

export type BulkActionId = "mark_all_read" | "mark_all_unread" | "mark_all_archive";

export interface BulkAction {
	id: BulkActionId;
	title: string;
	subtitle: string;
	keywords: string[];
}

export interface CommandPaletteCommandDefinition {
	id: CommandPaletteCommandId;
	title: string;
	subtitle?: string;
	shortcut?: string;
	keywords: string[];
}

export const BULK_ACTIONS: BulkAction[] = [
	{
		id: "mark_all_read",
		title: "Mark all as read",
		subtitle: "Mark all notifications in current view as read",
		keywords: ["mark", "read", "all", "bulk"],
	},
	{
		id: "mark_all_unread",
		title: "Mark all as unread",
		subtitle: "Mark all notifications in current view as unread",
		keywords: ["mark", "unread", "new", "all", "bulk"],
	},
	{
		id: "mark_all_archive",
		title: "Archive all",
		subtitle: "Archive all notifications in current view",
		keywords: ["archive", "done", "complete", "all", "bulk"],
	},
];

export function getMatchingBulkActions(query: string): BulkAction[] {
	const normalizedQuery = query.trim().toLowerCase();
	if (!normalizedQuery) {
		return BULK_ACTIONS;
	}

	return BULK_ACTIONS.filter((action) => {
		if (action.title.toLowerCase().includes(normalizedQuery)) {
			return true;
		}
		if (action.subtitle.toLowerCase().includes(normalizedQuery)) {
			return true;
		}
		return action.keywords.some((keyword) => keyword.toLowerCase().includes(normalizedQuery));
	});
}

interface ParsedCommandInput {
	raw: string;
	commandId: CommandPaletteCommandId | null;
	commandToken: string | null;
	commandQuery: string;
	rootQuery: string;
	isPartialCommand: boolean; // true if user is still typing the command (no space after token)
}

const COMMAND_PALETTE_DEFINITIONS: CommandPaletteCommandDefinition[] = [
	{
		id: "view",
		title: "Switch view",
		subtitle: "Jump directly to any notification view",
		shortcut: "⌘K",
		keywords: ["view", "switch", "navigate", "go to", "open view"],
	},
	{
		id: "bulk",
		title: "Bulk actions",
		subtitle: "Mark all notifications in current view",
		shortcut: "⇧V",
		keywords: ["bulk", "mark", "all", "actions"],
	},
	{
		id: "query",
		title: "Query syntax guide",
		subtitle: "Learn how to write advanced search queries",
		keywords: ["query", "syntax", "guide", "search", "filter", "help"],
	},
	{
		id: "shortcuts",
		title: "Show keyboard shortcuts",
		subtitle: "Review available shortcuts and command ideas",
		shortcut: "H",
		keywords: ["shortcuts", "help", "shortcut", "docs", "commands"],
	},
];

const COMMAND_LOOKUP = new Map<CommandPaletteCommandId, CommandPaletteCommandDefinition>(
	COMMAND_PALETTE_DEFINITIONS.map((definition) => [definition.id, definition])
);

const paletteSearchTerm = writable<string>("");
const paletteSearchTrimmed = derived(paletteSearchTerm, (term) => term.trim());
const paletteSearchActive = derived(paletteSearchTrimmed, (term) => term.length > 0);
const paletteSearchLoading = writable<boolean>(false);

export const commandPaletteSearchTerm = derived(paletteSearchTerm, (term) => term);
export const commandPaletteSearchTrimmed = paletteSearchTrimmed;
export const commandPaletteSearchActive = paletteSearchActive;
export const commandPaletteSearchLoading = derived(paletteSearchLoading, (value) => value);

function parseCommandInput(value: string): ParsedCommandInput {
	const raw = value ?? "";
	if (!raw.startsWith(">")) {
		return {
			raw,
			commandId: null,
			commandToken: null,
			commandQuery: "",
			rootQuery: raw.trimStart(),
			isPartialCommand: false,
		};
	}

	const withoutPrefix = raw.slice(1);
	const trimmed = withoutPrefix.trimStart();
	if (trimmed.length === 0) {
		return {
			raw,
			commandId: null,
			commandToken: null,
			commandQuery: "",
			rootQuery: "",
			isPartialCommand: false,
		};
	}

	const [token, ...restParts] = trimmed.split(/\s+/);
	const commandToken = token ?? "";
	const normalizedToken = commandToken.toLowerCase() as CommandPaletteCommandId;

	const commandDefinition = COMMAND_LOOKUP.get(normalizedToken);
	const commandId = commandDefinition ? normalizedToken : null;

	// Check if there's a space after the token (user has completed typing the command)
	const hasSpaceAfterToken =
		withoutPrefix.trimStart() !== withoutPrefix.trimStart().replace(/\s+$/, "") ||
		restParts.length > 0;

	// If no command match and no space after token, treat as partial/filtering
	const isPartialCommand = !commandId && !hasSpaceAfterToken;

	const remainderSource =
		restParts.length > 0 ? trimmed.slice(commandToken.length).trimStart() : "";

	// If it's a partial command (user still typing), treat it as a root query for filtering
	if (isPartialCommand) {
		return {
			raw,
			commandId: null,
			commandToken: null,
			commandQuery: "",
			rootQuery: commandToken,
			isPartialCommand: true,
		};
	}

	return {
		raw,
		commandId,
		commandToken: commandToken.length > 0 ? commandToken : null,
		commandQuery: remainderSource,
		rootQuery: "",
		isPartialCommand: false,
	};
}

interface CommandPaletteControllerStores {
	isOpen: Writable<boolean>;
	inputValue: Writable<string>;
}

interface CommandPaletteControllerDerived {
	parsed: Readable<ParsedCommandInput>;
	activeCommandId: Readable<CommandPaletteCommandId | null>;
	commandToken: Readable<string | null>;
	commandQuery: Readable<string>;
	rootQuery: Readable<string>;
	rootCommandMatches: Readable<CommandPaletteCommandDefinition[]>;
}

interface CommandPaletteControllerActions {
	open: (commandId?: CommandPaletteCommandId) => void;
	openCommand: (commandId: CommandPaletteCommandId) => void;
	close: () => void;
	setInputValue: (value: string) => void;
	resetInput: () => void;
}

export interface CommandPaletteController {
	stores: CommandPaletteControllerStores;
	derived: CommandPaletteControllerDerived;
	actions: CommandPaletteControllerActions;
	commands: CommandPaletteCommandDefinition[];
}

function commandToInput(commandId: CommandPaletteCommandId): string {
	switch (commandId) {
		case "view":
		case "bulk":
			return `>${commandId} `;
		case "shortcuts":
		case "query":
			return `>${commandId}`;
		default:
			return "";
	}
}

export function createCommandPaletteController(): CommandPaletteController {
	const isOpen = writable<boolean>(false);
	const inputValue = writable<string>("");

	const parsed = derived(inputValue, ($inputValue) => parseCommandInput($inputValue));

	const activeCommandId = derived(
		parsed,
		($parsed) => $parsed.commandId
	) as Readable<CommandPaletteCommandId | null>;

	const commandToken = derived(parsed, ($parsed) => $parsed.commandToken);
	const commandQuery = derived(parsed, ($parsed) => $parsed.commandQuery);
	const rootQuery = derived(parsed, ($parsed) => $parsed.rootQuery);

	const rootCommandMatches = derived(rootQuery, ($rootQuery) => {
		const query = $rootQuery.trim().toLowerCase();
		if (!query) {
			return COMMAND_PALETTE_DEFINITIONS;
		}

		return COMMAND_PALETTE_DEFINITIONS.filter((command) => {
			if (command.title.toLowerCase().includes(query)) {
				return true;
			}
			if (command.subtitle?.toLowerCase().includes(query)) {
				return true;
			}
			return command.keywords.some((keyword) => keyword.toLowerCase().includes(query));
		});
	});

	function open(commandId?: CommandPaletteCommandId) {
		isOpen.set(true);
		if (commandId) {
			inputValue.set(commandToInput(commandId));
		}
	}

	function openCommand(commandId: CommandPaletteCommandId) {
		open(commandId);
	}

	function close() {
		isOpen.set(false);
		inputValue.set("");
	}

	function setInputValue(value: string) {
		inputValue.set(value);
	}

	function resetInput() {
		inputValue.set("");
	}

	return {
		stores: {
			isOpen,
			inputValue,
		},
		derived: {
			parsed,
			activeCommandId,
			commandToken,
			commandQuery,
			rootQuery,
			rootCommandMatches,
		},
		actions: {
			open,
			openCommand,
			close,
			setInputValue,
			resetInput,
		},
		commands: COMMAND_PALETTE_DEFINITIONS,
	};
}
