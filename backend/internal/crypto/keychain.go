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

// Package crypto provides encryption utilities for sensitive data storage.
package crypto

// Keychain provides an interface for storing and retrieving tokens securely.
// On macOS, this uses the system keychain. On other platforms, it falls back
// to encrypted storage in SQLite.
type Keychain interface {
	// StoreToken stores a token in the keychain using the given service and account.
	StoreToken(service, account, token string) error

	// GetToken retrieves a token from the keychain using the given service and account.
	GetToken(service, account string) (string, error)

	// DeleteToken removes a token from the keychain using the given service and account.
	DeleteToken(service, account string) error

	// TokenExists checks if a token exists in the keychain for the given service and account.
	TokenExists(service, account string) (bool, error)
}

// GetKeychainAccount returns the keychain account name for a GitHub username.
// This namespaces tokens by username to support future multi-account profiles.
func GetKeychainAccount(username string) string {
	return "github_" + username
}
