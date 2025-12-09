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
 * Command Registry Pattern for Keyboard Shortcuts
 *
 * This provides a centralized way to register and execute commands by name,
 * replacing the previous callback-based system with a cleaner registry pattern.
 */

export type CommandHandler = (context?: CommandContext) => boolean | Promise<boolean>;

export interface CommandContext {
	// State getters
	getDetailOpen: () => boolean;
	getSplitViewMode?: () => boolean;
	isAnyDialogOpen: () => boolean;
	isFilterDropdownOpen: () => boolean;
	getMultiselectActive?: () => boolean;
	getHasSelection?: () => boolean;
	getSnoozeDropdownOpen?: () => boolean;
	getTagDropdownOpen?: () => boolean;
	getBulkSnoozeDropdownOpen?: () => boolean;
	getBulkTagDropdownOpen?: () => boolean;
	// For special handlers that need parameters
	selectSnoozeOption?: (optionIndex: number) => boolean;
}

// Global command registry
const commandRegistry = new Map<string, CommandHandler>();

/**
 * Register a command handler
 */
export function registerCommand(name: string, handler: CommandHandler): void {
	commandRegistry.set(name, handler);
}

/**
 * Unregister a command handler
 */
export function unregisterCommand(name: string): void {
	commandRegistry.delete(name);
}

/**
 * Execute a command by name
 */
export function executeCommand(name: string, context: CommandContext): boolean | Promise<boolean> {
	const handler = commandRegistry.get(name);
	if (!handler) {
		console.warn(`Command "${name}" not found in registry`);
		return false;
	}
	return handler(context);
}

/**
 * Check if a command is registered
 */
export function hasCommand(name: string): boolean {
	return commandRegistry.has(name);
}
