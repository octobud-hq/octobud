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

package github

import (
	"context"
	"fmt"
	"runtime"

	"github.com/octobud-hq/octobud/backend/internal/crypto"
	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/github"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
)

const darwinOS = "darwin"

// ErrNoTokenConfigured is returned when a user has no GitHub token configured
var ErrNoTokenConfigured = fmt.Errorf("no GitHub token configured for user")

// TokenProvider provides GitHub clients configured with per-user tokens
type TokenProvider interface {
	// GetClientForUser returns a configured GitHub client for the user.
	// Returns ErrNoTokenConfigured if user has no token configured.
	GetClientForUser(ctx context.Context, userID int64) (githubinterfaces.Client, error)
}

// tokenProvider implements TokenProvider
type tokenProvider struct {
	store     db.Store
	encryptor *crypto.Encryptor
	keychain  crypto.Keychain
}

// NewTokenProvider creates a new TokenProvider
func NewTokenProvider(
	store db.Store,
	encryptor *crypto.Encryptor,
	keychain crypto.Keychain,
) TokenProvider {
	return &tokenProvider{
		store:     store,
		encryptor: encryptor,
		keychain:  keychain,
	}
}

// GetClientForUser returns a configured GitHub client for the user
func (p *tokenProvider) GetClientForUser(
	ctx context.Context,
	userID int64,
) (githubinterfaces.Client, error) {
	// Get user from database
	// Note: Currently we only have GetUser() which gets the first user
	// For multi-tenant, we'd need GetUserByID(userID)
	// For now, we verify the userID matches
	user, err := p.store.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.ID != userID {
		return nil, fmt.Errorf("user ID mismatch: expected %d, got %d", userID, user.ID)
	}

	// Try to get token from keychain on macOS first
	var token string
	if p.keychain != nil && runtime.GOOS == darwinOS {
		if user.GithubUsername.Valid && user.GithubUsername.String != "" {
			account := crypto.GetKeychainAccount(user.GithubUsername.String)
			token, err = p.keychain.GetToken("io.octobud", account)
			if err != nil {
				// Keychain doesn't have token, fall back to SQLite
				token = ""
			}
		}
	}

	// Fall back to SQLite encryption if keychain doesn't have token
	if token == "" {
		if !user.GithubTokenEncrypted.Valid || user.GithubTokenEncrypted.String == "" {
			return nil, ErrNoTokenConfigured
		}

		// Decrypt token
		token, err = p.encryptor.Decrypt(user.GithubTokenEncrypted.String)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %w", err)
		}

		if token == "" {
			return nil, ErrNoTokenConfigured
		}
	}

	// Create and configure GitHub client
	client := github.NewClient()
	if err := client.SetToken(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to configure GitHub client: %w", err)
	}

	return client, nil
}
