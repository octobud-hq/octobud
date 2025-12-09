//go:build !darwin

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

package crypto

import (
	"fmt"
)

// keychainFallback implements Keychain using SQLite encryption for non-macOS platforms.
// This is a no-op implementation that returns errors indicating keychain is not available.
// The actual token storage will be handled by the Encryptor + SQLite on these platforms.
type keychainFallback struct{}

// NewKeychain creates a new Keychain implementation for non-macOS platforms.
// This returns a fallback that indicates keychain is not available.
func NewKeychain() Keychain {
	return &keychainFallback{}
}

// StoreToken is not supported on non-macOS platforms.
// Tokens should be stored using Encryptor + SQLite instead.
func (k *keychainFallback) StoreToken(service, account, token string) error {
	return fmt.Errorf("keychain is not available on this platform, use SQLite encryption instead")
}

// GetToken is not supported on non-macOS platforms.
// Tokens should be retrieved using Encryptor + SQLite instead.
func (k *keychainFallback) GetToken(service, account string) (string, error) {
	return "", fmt.Errorf("keychain is not available on this platform, use SQLite encryption instead")
}

// DeleteToken is not supported on non-macOS platforms.
// Tokens should be deleted from SQLite instead.
func (k *keychainFallback) DeleteToken(service, account string) error {
	return fmt.Errorf("keychain is not available on this platform, use SQLite encryption instead")
}

// TokenExists is not supported on non-macOS platforms.
func (k *keychainFallback) TokenExists(service, account string) (bool, error) {
	return false, fmt.Errorf("keychain is not available on this platform, use SQLite encryption instead")
}

