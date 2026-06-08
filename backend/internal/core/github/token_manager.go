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

	"github.com/octobud-hq/octobud/backend/internal/db"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	"github.com/octobud-hq/octobud/backend/internal/xcrypto"
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

// TokenHealth classifies the current token's usability for the frontend banner.
type TokenHealth string

const (
	// TokenHealthOK indicates the token is working and not near expiration.
	TokenHealthOK TokenHealth = "ok"
	// TokenHealthExpiring indicates the token is valid but expires within the warning window.
	TokenHealthExpiring TokenHealth = "expiring"
	// TokenHealthInvalid indicates the token has expired or was rejected by GitHub (401).
	TokenHealthInvalid TokenHealth = "invalid"
)

// tokenExpirationWarningWindow is how far in advance we flag a token as "expiring".
const tokenExpirationWarningWindow = 7 * 24 * time.Hour

// AuthService defines the interface for updating GitHub identity
type AuthService interface {
	UpdateGitHubIdentity(ctx context.Context, githubUserID, githubUsername string) error
}

// TokenManager manages GitHub PAT storage and configuration
type TokenManager struct {
	mu           sync.RWMutex
	store        db.Store
	encryptor    *xcrypto.Encryptor
	keychain     xcrypto.Keychain
	githubClient githubinterfaces.Client
	authService  AuthService
	httpClient   *http.Client
	logger       *zap.Logger

	// Current state
	currentToken   string
	currentSource  TokenSource
	githubUsername string
	githubUserID   string

	// Observed token health — populated by the GitHub client's RoundTripper.
	// Guarded by healthMu (separate from mu) so the observer callback can fire
	// during operations that hold mu (e.g. Initialize → githubClient.SetToken).
	healthMu       sync.RWMutex
	tokenExpiresAt *time.Time
	tokenInvalid   bool

	// Recovery loop state — used when Initialize fails to load a previously
	// configured token (typically a transient keychain access failure right
	// after macOS login). Guarded by recoveryMu.
	recoveryMu     sync.Mutex
	recoveryCancel context.CancelFunc
	// recoveryGen increments each time a recovery goroutine is started or
	// canceled. The goroutine reads its generation at launch and only clears
	// recoveryCancel on exit if the current generation still matches —
	// preventing a stale defer from clobbering a newer goroutine's state.
	recoveryGen int
	// recoveryBackoff is the schedule of delays between retry attempts.
	// Overridable in tests. The final entry repeats indefinitely.
	recoveryBackoff []time.Duration
}

// defaultRecoveryBackoff is the schedule used in production: retry quickly at
// first (keychain often becomes accessible within seconds of login) and then
// back off so we don't spam the keychain or logs if the failure is persistent.
var defaultRecoveryBackoff = []time.Duration{
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
}

// NewTokenManager creates a new TokenManager
func NewTokenManager(
	store db.Store,
	encryptor *xcrypto.Encryptor,
	keychain xcrypto.Keychain,
	githubClient githubinterfaces.Client,
	authService AuthService,
	logger *zap.Logger,
) *TokenManager {
	m := &TokenManager{
		store:        store,
		encryptor:    encryptor,
		keychain:     keychain,
		githubClient: githubClient,
		authService:  authService,
		logger:       logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		currentSource:   TokenSourceNone,
		recoveryBackoff: defaultRecoveryBackoff,
	}
	githubClient.SetTokenObserver(m.observeResponse)
	return m
}

// observeResponse is the callback registered on the GitHub HTTP client's
// RoundTripper. It runs on every authenticated response so we can update
// in-memory token health without plumbing through each call site.
//
// Uses healthMu rather than mu because this fires from inside githubClient
// HTTP calls that are sometimes made while mu is already held by the caller
// (e.g. Initialize → githubClient.SetToken).
func (m *TokenManager) observeResponse(statusCode int, expiresAt *time.Time) {
	m.healthMu.Lock()
	if expiresAt != nil {
		if m.tokenExpiresAt == nil || !m.tokenExpiresAt.Equal(*expiresAt) {
			t := *expiresAt
			m.tokenExpiresAt = &t
		}
	}

	wasInvalid := m.tokenInvalid
	switch {
	case statusCode == http.StatusUnauthorized:
		m.tokenInvalid = true
	case statusCode >= 200 && statusCode < 300:
		m.tokenInvalid = false
	}
	transitioned := !wasInvalid && m.tokenInvalid
	m.healthMu.Unlock()

	if transitioned && m.logger != nil {
		m.logger.Warn(
			"github token observed as invalid (401); banner will prompt reconnect",
			zap.Int("statusCode", statusCode),
		)
	}
}

// resetHealth clears observed health state — called on token changes.
func (m *TokenManager) resetHealth() {
	m.healthMu.Lock()
	defer m.healthMu.Unlock()
	m.tokenExpiresAt = nil
	m.tokenInvalid = false
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
			account := xcrypto.GetKeychainAccount(user.GithubUsername.String)
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

// StartKeychainRecovery launches a background goroutine that retries
// Initialize() on a backoff schedule when the app started with no loaded
// token but the user was previously connected. This handles the case where
// the macOS keychain is briefly inaccessible right after a system restart:
// without this, sync runs with an empty token, GitHub returns 401, and the
// user has to quit and relaunch the app to recover.
//
// No-op when the token loaded successfully, when there's no record of a
// prior connection (fresh install), or when running on a platform without
// keychain support (only macOS has the brief-inaccessibility issue this
// addresses). Safe to call multiple times — concurrent calls beyond the
// first are ignored.
func (m *TokenManager) StartKeychainRecovery(ctx context.Context) {
	if runtime.GOOS != darwinOS {
		return
	}

	m.mu.RLock()
	source := m.currentSource
	m.mu.RUnlock()
	if source != TokenSourceNone {
		return
	}

	user, err := m.store.GetUser(ctx)
	if err != nil || !user.GithubUsername.Valid || user.GithubUsername.String == "" {
		return
	}

	m.recoveryMu.Lock()
	if m.recoveryCancel != nil {
		m.recoveryMu.Unlock()
		return
	}
	recoveryCtx, cancel := context.WithCancel(ctx)
	m.recoveryCancel = cancel
	m.recoveryGen++
	gen := m.recoveryGen
	m.recoveryMu.Unlock()

	m.logger.Info(
		"token did not load on startup; starting keychain recovery loop",
		zap.String("github_username", user.GithubUsername.String),
	)
	go m.runRecoveryLoop(recoveryCtx, gen)
}

// runRecoveryLoop retries Initialize on the configured backoff until the
// token loads or the context is canceled. The final backoff entry repeats
// indefinitely — a keychain that's failing now may become accessible later
// (e.g. user finally taps through a Touch ID prompt).
func (m *TokenManager) runRecoveryLoop(ctx context.Context, gen int) {
	defer func() {
		m.recoveryMu.Lock()
		if m.recoveryGen == gen {
			m.recoveryCancel = nil
		}
		m.recoveryMu.Unlock()
	}()

	if len(m.recoveryBackoff) == 0 {
		return
	}

	for attempt := 0; ; attempt++ {
		delay := m.recoveryBackoff[len(m.recoveryBackoff)-1]
		if attempt < len(m.recoveryBackoff) {
			delay = m.recoveryBackoff[attempt]
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if err := m.Initialize(ctx); err != nil {
			m.logger.Warn(
				"keychain recovery attempt failed",
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			continue
		}

		m.mu.RLock()
		source := m.currentSource
		username := m.githubUsername
		m.mu.RUnlock()
		if source != TokenSourceNone {
			m.logger.Info(
				"GitHub token recovered from keychain",
				zap.Int("attempt", attempt+1),
				zap.String("source", string(source)),
				zap.String("github_username", username),
			)
			return
		}
	}
}

// stopRecovery cancels any in-flight recovery loop. Called when the user
// takes explicit action (set/clear token) — at that point the loop is
// either redundant (we just got a working token) or moot (user cleared).
func (m *TokenManager) stopRecovery() {
	m.recoveryMu.Lock()
	if m.recoveryCancel != nil {
		m.recoveryCancel()
		m.recoveryCancel = nil
		// Bump the generation so the canceled goroutine's defer becomes a
		// no-op — protects against it clobbering a future loop's state.
		m.recoveryGen++
	}
	m.recoveryMu.Unlock()
}

// GetStatus returns the current GitHub connection status
func (m *TokenManager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Status{
		Connected:      m.currentSource != TokenSourceNone,
		Source:         m.currentSource,
		MaskedToken:    xcrypto.MaskToken(m.currentToken),
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
	return xcrypto.MaskToken(m.currentToken)
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

// GetTokenExpiresAt returns the last observed token expiration, or nil if the
// token has no expiration (classic PAT) or we have not yet made an authenticated
// GitHub call this session.
func (m *TokenManager) GetTokenExpiresAt() *time.Time {
	m.healthMu.RLock()
	defer m.healthMu.RUnlock()
	if m.tokenExpiresAt == nil {
		return nil
	}
	t := *m.tokenExpiresAt
	return &t
}

// GetTokenHealth classifies the current token's usability. Only evaluated for
// stored PATs — OAuth tokens expire on a short cycle and are refreshed
// automatically, so surfacing expiration/invalidation to the user isn't
// actionable. Returns TokenHealthOK for any non-PAT source (disconnected state
// is surfaced separately via Connected=false).
func (m *TokenManager) GetTokenHealth() TokenHealth {
	m.mu.RLock()
	source := m.currentSource
	m.mu.RUnlock()
	if source != TokenSourceStored {
		return TokenHealthOK
	}

	m.healthMu.RLock()
	invalid := m.tokenInvalid
	var expiresAt *time.Time
	if m.tokenExpiresAt != nil {
		t := *m.tokenExpiresAt
		expiresAt = &t
	}
	m.healthMu.RUnlock()

	if invalid {
		return TokenHealthInvalid
	}
	if expiresAt != nil {
		remaining := time.Until(*expiresAt)
		if remaining <= 0 {
			return TokenHealthInvalid
		}
		if remaining <= tokenExpirationWarningWindow {
			return TokenHealthExpiring
		}
	}
	return TokenHealthOK
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
	m.stopRecovery()

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Validate token and get user info
	userID, username, err := m.validateAndGetUser(ctx, token)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	// Get old username before updating, so we can delete old keychain item if username changed
	var oldUsername string
	user, err := m.store.GetUser(ctx)
	if err == nil && user.GithubUsername.Valid {
		oldUsername = user.GithubUsername.String
	}

	// Store token based on platform
	if m.keychain != nil && runtime.GOOS == darwinOS {
		// Delete old keychain item if username changed (graceful failure - log but don't fail)
		if oldUsername != "" && oldUsername != username {
			oldAccount := xcrypto.GetKeychainAccount(oldUsername)
			if deleteErr := m.keychain.DeleteToken("io.octobud", oldAccount); deleteErr != nil {
				m.logger.Warn("failed to delete old keychain item when changing username",
					zap.String("old_username", oldUsername),
					zap.Error(deleteErr),
				)
				// Continue - don't fail the operation
			}
		}

		// Store in keychain on macOS
		account := xcrypto.GetKeychainAccount(username)
		if storeErr := m.keychain.StoreToken("io.octobud", account, token); storeErr != nil {
			return "", fmt.Errorf("failed to store token in keychain: %w", storeErr)
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

	// Update internal state (resetHealth uses healthMu — independent of mu).
	m.mu.Lock()
	m.currentToken = token
	m.currentSource = TokenSourceStored
	m.githubUsername = username
	m.githubUserID = userID
	m.resetHealth()
	m.mu.Unlock()
	return username, nil
}

// SetTokenFromOAuth validates, encrypts, and stores a token from OAuth flow.
// Returns the GitHub username associated with the token.
func (m *TokenManager) SetTokenFromOAuth(ctx context.Context, token string) (string, error) {
	m.stopRecovery()

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Validate token and get user info
	userID, username, err := m.validateAndGetUser(ctx, token)
	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	// Get old username before updating, so we can delete old keychain item if username changed
	var oldUsername string
	user, err := m.store.GetUser(ctx)
	if err == nil && user.GithubUsername.Valid {
		oldUsername = user.GithubUsername.String
	}

	// Store token based on platform
	if m.keychain != nil && runtime.GOOS == darwinOS {
		// Delete old keychain item if username changed (graceful failure - log but don't fail)
		if oldUsername != "" && oldUsername != username {
			oldAccount := xcrypto.GetKeychainAccount(oldUsername)
			if deleteErr := m.keychain.DeleteToken("io.octobud", oldAccount); deleteErr != nil {
				m.logger.Warn("failed to delete old keychain item when changing username via OAuth",
					zap.String("old_username", oldUsername),
					zap.Error(deleteErr),
				)
				// Continue - don't fail the operation
			}
		}

		// Store in keychain on macOS
		account := xcrypto.GetKeychainAccount(username)
		if storeErr := m.keychain.StoreToken("io.octobud", account, token); storeErr != nil {
			return "", fmt.Errorf("failed to store token in keychain: %w", storeErr)
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

	// Update internal state - mark as OAuth source.
	// (resetHealth uses healthMu — independent of mu.)
	m.mu.Lock()
	m.currentToken = token
	m.currentSource = TokenSourceOAuth
	m.githubUsername = username
	m.githubUserID = userID
	m.resetHealth()
	m.mu.Unlock()

	return username, nil
}

// ClearToken removes the stored token.
func (m *TokenManager) ClearToken(ctx context.Context) error {
	m.stopRecovery()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get username before clearing (for keychain deletion)
	user, err := m.store.GetUser(ctx)
	username := ""
	if err == nil && user.GithubUsername.Valid {
		username = user.GithubUsername.String
	}

	// Also check current state for username (in case database is out of sync)
	if username == "" && m.githubUsername != "" {
		username = m.githubUsername
	}

	// Clear from keychain on macOS (with graceful failure handling)
	if m.keychain != nil && runtime.GOOS == darwinOS {
		if username != "" {
			// Delete keychain item for known username
			account := xcrypto.GetKeychainAccount(username)
			if deleteErr := m.keychain.DeleteToken("io.octobud", account); deleteErr != nil {
				// Log warning but don't fail - keychain deletion failures shouldn't block disconnection
				m.logger.Warn("failed to delete token from keychain during disconnect",
					zap.String("username", username),
					zap.Error(deleteErr),
				)
			}
		} else {
			// If we don't have a username, log a warning but continue
			// This could happen in edge cases where the database is in an inconsistent state
			m.logger.Debug("skipping keychain deletion: no username available",
				zap.String("current_username", m.githubUsername),
			)
		}
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
	m.resetHealth()

	return nil
}

// validateAndGetUser validates a GitHub token and returns the user ID and username
func (m *TokenManager) validateAndGetUser(
	ctx context.Context,
	token string,
) (userID, username string, err error) {
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
		if closeErr := resp.Body.Close(); closeErr != nil {
			m.logger.Warn("failed to close response body", zap.Error(closeErr))
		}
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
