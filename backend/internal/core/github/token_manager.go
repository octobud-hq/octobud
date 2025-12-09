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

// Package github provides services for managing GitHub integration.
package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/crypto"
	"github.com/octobud-hq/octobud/backend/internal/db"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
)

// TokenSource indicates where the active token came from
type TokenSource string

const (
	// TokenSourceNone indicates no token is configured.
	TokenSourceNone TokenSource = "none"
	// TokenSourceStored indicates the token was loaded from storage.
	TokenSourceStored TokenSource = "stored"
	// TokenSourceOAuth indicates the token was obtained via OAuth flow.
	TokenSourceOAuth TokenSource = "oauth"
)

// Status represents the current GitHub connection status
type Status struct {
	Connected      bool        `json:"connected"`
	Source         TokenSource `json:"source"`
	MaskedToken    string      `json:"maskedToken,omitempty"`
	GitHubUsername string      `json:"githubUsername,omitempty"`
	GitHubUserID   string      `json:"githubUserId,omitempty"`
}

// AuthService defines the interface for updating GitHub identity
type AuthService interface {
	UpdateGitHubIdentity(ctx context.Context, githubUserID, githubUsername string) error
}

// TokenManager manages GitHub PAT storage and configuration
type TokenManager struct {
	mu           sync.RWMutex
	store        db.Store
	encryptor    *crypto.Encryptor
	keychain     crypto.Keychain
	githubClient githubinterfaces.Client
	authService  AuthService
	httpClient   *http.Client
	logger       *zap.Logger

	// Current state
	currentToken   string
	currentSource  TokenSource
	githubUsername string
	githubUserID   string
}

// NewTokenManager creates a new TokenManager
func NewTokenManager(
	store db.Store,
	encryptor *crypto.Encryptor,
	keychain crypto.Keychain,
	githubClient githubinterfaces.Client,
	authService AuthService,
	logger *zap.Logger,
) *TokenManager {
	return &TokenManager{
		store:        store,
		encryptor:    encryptor,
		keychain:     keychain,
		githubClient: githubClient,
		authService:  authService,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		currentSource: TokenSourceNone,
	}
}

// Initialize loads the token from storage and configures the GitHub client.
// Should be called at startup.
func (m *TokenManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to load stored token first (takes priority)
	user, err := m.store.GetUser(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// On macOS, try to load from keychain first
	if m.keychain != nil && runtime.GOOS == darwinOS {
		// If we have a username, try to load from keychain
		if user.GithubUsername.Valid && user.GithubUsername.String != "" {
			account := crypto.GetKeychainAccount(user.GithubUsername.String)
			token, err := m.keychain.GetToken("io.octobud", account)
			if err == nil && token != "" {
				// Validate the token
				userID, username, err := m.validateAndGetUser(ctx, token)
				if err != nil {
					// Token is invalid - log warning but don't delete immediately to avoid second keychain prompt.
					// The invalid token will be automatically overwritten on next SetToken() call.
					m.logger.Warn("stored token in keychain is invalid", zap.Error(err))
				} else {
					// Set the token on the GitHub client
					if err := m.githubClient.SetToken(ctx, token); err != nil {
						return fmt.Errorf("failed to set token on GitHub client: %w", err)
					}
					m.currentToken = token
					m.currentSource = TokenSourceStored
					m.githubUsername = username
					m.githubUserID = userID
					return nil
				}
			}
		}
	}

	// Fall back to SQLite encryption for non-macOS or if keychain doesn't have token
	if user.GithubTokenEncrypted.Valid && user.GithubTokenEncrypted.String != "" {
		token, err := m.encryptor.Decrypt(user.GithubTokenEncrypted.String)
		if err != nil {
			// Log warning but don't fail
			m.logger.Warn("failed to decrypt stored token", zap.Error(err))
		} else if token != "" {
			// Validate the stored token
			userID, username, err := m.validateAndGetUser(ctx, token)
			if err != nil {
				// Token is invalid, clear it
				m.logger.Warn("stored token is invalid", zap.Error(err))
			} else {
				// Set the token on the GitHub client
				if err := m.githubClient.SetToken(ctx, token); err != nil {
					return fmt.Errorf("failed to set token on GitHub client: %w", err)
				}
				m.currentToken = token
				m.currentSource = TokenSourceStored
				m.githubUsername = username
				m.githubUserID = userID
				return nil
			}
		}
	}

	// No token configured - this is OK, user can set one via UI
	m.currentSource = TokenSourceNone
	return nil
}

// GetStatus returns the current GitHub connection status
func (m *TokenManager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Status{
		Connected:      m.currentSource != TokenSourceNone,
		Source:         m.currentSource,
		MaskedToken:    crypto.MaskToken(m.currentToken),
		GitHubUsername: m.githubUsername,
		GitHubUserID:   m.githubUserID,
	}
}

// GetStatusConnected returns whether a token is configured
func (m *TokenManager) GetStatusConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSource != TokenSourceNone
}

// GetStatusSource returns the source of the current token ("none", "stored", or "oauth")
func (m *TokenManager) GetStatusSource() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return string(m.currentSource)
}

// GetStatusMaskedToken returns a masked version of the current token
func (m *TokenManager) GetStatusMaskedToken() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return crypto.MaskToken(m.currentToken)
}

// GetStatusGitHubUsername returns the GitHub username associated with the token
func (m *TokenManager) GetStatusGitHubUsername() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.githubUsername
}

// GetStatusGitHubUserID returns the GitHub user ID associated with the token
func (m *TokenManager) GetStatusGitHubUserID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.githubUserID
}

// IsConnected returns true if a valid GitHub token is configured
func (m *TokenManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSource != TokenSourceNone
}

// SetToken validates, encrypts, and stores a new GitHub token (PAT).
// Returns the GitHub username associated with the token.
func (m *TokenManager) SetToken(ctx context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Validate token and get user info
	userID, username, err := m.validateAndGetUser(ctx, token)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	// Store token based on platform
	if m.keychain != nil && runtime.GOOS == darwinOS {
		// Store in keychain on macOS
		account := crypto.GetKeychainAccount(username)
		if err := m.keychain.StoreToken("io.octobud", account, token); err != nil {
			return "", fmt.Errorf("failed to store token in keychain: %w", err)
		}
		// Clear SQLite encrypted token on macOS (set to NULL)
		_, err = m.store.UpdateUserGitHubToken(ctx, sql.NullString{
			String: "",
			Valid:  false,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clear SQLite token: %w", err)
		}
	} else {
		// Encrypt and store in SQLite for non-macOS
		encrypted, err := m.encryptor.Encrypt(token)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt token: %w", err)
		}
		_, err = m.store.UpdateUserGitHubToken(ctx, sql.NullString{
			String: encrypted,
			Valid:  true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to store token: %w", err)
		}
	}

	// Update GitHub identity in database
	if m.authService != nil {
		if err := m.authService.UpdateGitHubIdentity(ctx, userID, username); err != nil {
			return "", fmt.Errorf("failed to update GitHub identity: %w", err)
		}
	}

	// Update GitHub client
	if err := m.githubClient.SetToken(ctx, token); err != nil {
		return "", fmt.Errorf("failed to set token on GitHub client: %w", err)
	}

	// Update internal state
	m.mu.Lock()
	m.currentToken = token
	m.currentSource = TokenSourceStored
	m.githubUsername = username
	m.githubUserID = userID
	m.mu.Unlock()

	return username, nil
}

// SetTokenFromOAuth validates, encrypts, and stores a token from OAuth flow.
// Returns the GitHub username associated with the token.
func (m *TokenManager) SetTokenFromOAuth(ctx context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Validate token and get user info
	userID, username, err := m.validateAndGetUser(ctx, token)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	// Store token based on platform
	if m.keychain != nil && runtime.GOOS == darwinOS {
		// Store in keychain on macOS
		account := crypto.GetKeychainAccount(username)
		if err := m.keychain.StoreToken("io.octobud", account, token); err != nil {
			return "", fmt.Errorf("failed to store token in keychain: %w", err)
		}
		// Clear SQLite encrypted token on macOS (set to NULL)
		_, err = m.store.UpdateUserGitHubToken(ctx, sql.NullString{
			String: "",
			Valid:  false,
		})
		if err != nil {
			return "", fmt.Errorf("failed to clear SQLite token: %w", err)
		}
	} else {
		// Encrypt and store in SQLite for non-macOS
		encrypted, err := m.encryptor.Encrypt(token)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt token: %w", err)
		}
		_, err = m.store.UpdateUserGitHubToken(ctx, sql.NullString{
			String: encrypted,
			Valid:  true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to store token: %w", err)
		}
	}

	// Update GitHub identity in database
	if m.authService != nil {
		if err := m.authService.UpdateGitHubIdentity(ctx, userID, username); err != nil {
			return "", fmt.Errorf("failed to update GitHub identity: %w", err)
		}
	}

	// Update GitHub client
	if err := m.githubClient.SetToken(ctx, token); err != nil {
		return "", fmt.Errorf("failed to set token on GitHub client: %w", err)
	}

	// Update internal state - mark as OAuth source
	m.mu.Lock()
	m.currentToken = token
	m.currentSource = TokenSourceOAuth
	m.githubUsername = username
	m.githubUserID = userID
	m.mu.Unlock()

	return username, nil
}

// ClearToken removes the stored token.
func (m *TokenManager) ClearToken(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get username before clearing
	user, err := m.store.GetUser(ctx)
	username := ""
	if err == nil && user.GithubUsername.Valid {
		username = user.GithubUsername.String
	}

	// Clear from keychain on macOS
	if m.keychain != nil && runtime.GOOS == darwinOS && username != "" {
		account := crypto.GetKeychainAccount(username)
		_ = m.keychain.DeleteToken("io.octobud", account)
	}

	// Clear from database (this also clears GitHub identity)
	_, err = m.store.ClearUserGitHubToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear token: %w", err)
	}

	// Clear internal state
	m.currentToken = ""
	m.currentSource = TokenSourceNone
	m.githubUsername = ""
	m.githubUserID = ""

	return nil
}

// validateAndGetUser validates a GitHub token and returns the user ID and username
func (m *TokenManager) validateAndGetUser(
	ctx context.Context,
	token string,
) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", http.NoBody)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to GitHub: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", fmt.Errorf("invalid token: unauthorized")
	}

	if resp.StatusCode == http.StatusForbidden {
		// Check for rate limiting
		if strings.Contains(string(body), "rate limit") {
			return "", "", fmt.Errorf("GitHub API rate limit exceeded")
		}
		return "", "", fmt.Errorf("token does not have required permissions")
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf(
			"GitHub API returned status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	// Parse response to get user info
	var userResponse struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &userResponse); err != nil {
		return "", "", fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	if userResponse.Login == "" {
		return "", "", fmt.Errorf("GitHub response missing username")
	}

	if userResponse.ID == 0 {
		return "", "", fmt.Errorf("GitHub response missing user ID")
	}

	return strconv.FormatInt(userResponse.ID, 10), userResponse.Login, nil
}
