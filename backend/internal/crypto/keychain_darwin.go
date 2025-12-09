//go:build darwin

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
	"strings"

	"github.com/keybase/go-keychain"
)

// keychainImpl implements Keychain using macOS keychain services.
type keychainImpl struct{}

// NewKeychain creates a new Keychain implementation for macOS.
func NewKeychain() Keychain {
	return &keychainImpl{}
}

// StoreToken stores a token in the macOS keychain.
func (k *keychainImpl) StoreToken(service, account, token string) error {
	if service == "" {
		return fmt.Errorf("service cannot be empty")
	}
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Delete existing item if it exists (to update it)
	// Use DeleteToken which handles "not found" errors gracefully
	if err := k.DeleteToken(service, account); err != nil {
		// If deletion failed due to ownership, we can't proceed - the user needs to manually delete it
		// This should not happen in normal operation if the app bundle ID is consistent
		if strings.Contains(err.Error(), "-25244") {
			return fmt.Errorf("keychain item exists but cannot be updated due to ownership conflict. Please manually delete it: Open Keychain Access.app, search for '%s' with account '%s', and delete it. Then try again", service, account) //nolint:lll // Long error string.
		}
		// For other delete errors, fail immediately rather than trying to work around
		return fmt.Errorf("failed to delete existing keychain item: %w", err)
	}

	// Create a new generic password item
	item := keychain.NewGenericPassword(service, account, "", []byte(token), "")

	// Add the new token
	err := keychain.AddItem(item)
	if err != nil {
		// If we get a duplicate item error, this is unexpected since we just deleted
		// This might indicate a race condition or the delete didn't fully complete
		errStr := err.Error()
		if strings.Contains(errStr, "-25299") || errStr == "Item already exists" {
			return fmt.Errorf("failed to store token: item already exists (this should not happen). Please try again")
		}
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}

	return nil
}

// GetToken retrieves a token from the macOS keychain.
func (k *keychainImpl) GetToken(service, account string) (string, error) {
	if service == "" {
		return "", fmt.Errorf("service cannot be empty")
	}
	if account == "" {
		return "", fmt.Errorf("account cannot be empty")
	}

	// Get the token from keychain
	data, err := keychain.GetGenericPassword(service, account, "", "")
	if err != nil {
		return "", fmt.Errorf("failed to get token from keychain: %w", err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("token not found in keychain")
	}

	return string(data), nil
}

// DeleteToken removes a token from the macOS keychain.
func (k *keychainImpl) DeleteToken(service, account string) error {
	if service == "" {
		return fmt.Errorf("service cannot be empty")
	}
	if account == "" {
		return fmt.Errorf("account cannot be empty")
	}

	err := keychain.DeleteGenericPasswordItem(service, account)
	if err != nil {
		// It's OK if the item doesn't exist
		if err.Error() == "Item not found" {
			return nil
		}
		// Check for ownership error - this happens when the item was created by a different
		// bundle ID or unsigned binary (common during development)
		errStr := err.Error()
		if errStr == "An invalid attempt to change the owner of an item. (-25244)" ||
			strings.Contains(errStr, "-25244") {
			return fmt.Errorf("cannot delete keychain item due to ownership (created by different app version): %w. Please manually delete it from Keychain Access.app", err)
		}
		return fmt.Errorf("failed to delete token from keychain: %w", err)
	}

	return nil
}

// TokenExists checks if a token exists in the macOS keychain.
func (k *keychainImpl) TokenExists(service, account string) (bool, error) {
	if service == "" {
		return false, fmt.Errorf("service cannot be empty")
	}
	if account == "" {
		return false, fmt.Errorf("account cannot be empty")
	}

	// Try to get the token - if it exists, return true
	_, err := keychain.GetGenericPassword(service, account, "", "")
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "Item not found" || err.Error() == "Could not find the item" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check keychain: %w", err)
	}

	return true, nil
}
