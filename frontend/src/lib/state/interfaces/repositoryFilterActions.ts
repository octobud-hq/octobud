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
 * Actions for the list's repository filter (the selector above the notification list).
 * The selection is URL state (`?repos=`), so every mutation navigates and lets the route
 * loader refetch the page.
 */
export interface RepositoryFilterActions {
	/** Replace the selection. An empty list means "all repositories". */
	setRepositoryFilter: (repositoryIds: number[]) => Promise<void>;
	/** Add or remove one repository from the selection. */
	toggleRepositoryInFilter: (repositoryId: number) => Promise<void>;
	/** Select exactly this repository. */
	selectOnlyRepository: (repositoryId: number) => Promise<void>;
	/** Back to all repositories. Returns false if there was nothing to clear. */
	clearRepositoryFilter: () => Promise<boolean>;
	/** Refetch per-repository counts for the current query (without the repository filter). */
	refreshRepositoryCounts: () => Promise<void>;
	/** Open the selector dropdown (also refreshes counts). Returns false if it was already open. */
	openRepositoryFilter: () => boolean;
	closeRepositoryFilter: () => void;
	toggleRepositoryPin: (repositoryId: number) => void;
	/** Persist the current selection as the view's default (empty selection clears it). */
	setViewRepositoryDefault: () => Promise<void>;
	/** Navigate back to the view's default selection. */
	resetToViewDefault: () => Promise<void>;
}
