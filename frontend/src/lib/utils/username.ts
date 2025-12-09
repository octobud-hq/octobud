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

// Username validation constants - must match backend/internal/core/auth/username.go
export const MIN_USERNAME_LENGTH = 3;
export const MAX_USERNAME_LENGTH = 64;

export interface UsernameValidationResult {
	valid: boolean;
	error?: string;
}

/**
 * Validates username format
 * Returns validation result with error message if invalid
 */
export function validateUsername(username: string): UsernameValidationResult {
	const length = [...username].length; // Count Unicode characters, not bytes

	if (length === 0) {
		return {
			valid: false,
			error: "Username cannot be empty",
		};
	}

	if (length < MIN_USERNAME_LENGTH) {
		return {
			valid: false,
			error: `Username must be at least ${MIN_USERNAME_LENGTH} characters long (got ${length})`,
		};
	}

	if (length > MAX_USERNAME_LENGTH) {
		return {
			valid: false,
			error: `Username must be no more than ${MAX_USERNAME_LENGTH} characters long (got ${length})`,
		};
	}

	// Pattern: alphanumeric (including Unicode), underscores, dots, hyphens, and @
	// Must start with letter/number, can contain @ but not at start/end
	// Simplified regex for frontend (full validation happens on backend)
	const pattern =
		/^[\p{L}\p{N}]([\p{L}\p{N}._-]*[\p{L}\p{N}])?$|^[\p{L}\p{N}][\p{L}\p{N}._-]*@[\p{L}\p{N}]([\p{L}\p{N}.-]*[\p{L}\p{N}])?$/u;

	if (!pattern.test(username)) {
		return {
			valid: false,
			error:
				"Username can only contain letters, numbers, underscores, dots, hyphens, and @ symbols",
		};
	}

	return { valid: true };
}
