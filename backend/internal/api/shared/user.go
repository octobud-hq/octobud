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

package shared

import (
	"context"
	"errors"
	"net/http"

	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
)

// ErrNoGitHubIdentity is returned when the user has not connected their GitHub account
var ErrNoGitHubIdentity = errors.New("user has not connected their GitHub account")

// contextKey is a type for context keys to avoid collisions
type contextKey string

// userIDContextKey is the key used to store the user ID in context
const userIDContextKey contextKey = "userID"

// ContextWithUserID returns a new context with the user ID stored
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// GetUserIDFromContext retrieves the user ID from context.
// Returns empty string if not found.
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDContextKey).(string); ok {
		return userID
	}
	return ""
}

// GetUserID retrieves the current user's ID (GitHub user ID) from the auth service.
// This is used to scope all data operations to the current user.
// Returns ErrNoGitHubIdentity if the user has not connected their GitHub account.
func GetUserID(ctx context.Context, authSvc authsvc.AuthService) (string, error) {
	user, err := authSvc.GetUser(ctx)
	if err != nil {
		return "", err
	}

	if user.GithubUserID == "" {
		return "", ErrNoGitHubIdentity
	}

	return user.GithubUserID, nil
}

// RequireUserID is a helper that gets the user ID and writes an HTTP error response if not available.
// Returns the userID and true if successful, or empty string and false if an error was written.
func RequireUserID(
	ctx context.Context,
	w http.ResponseWriter,
	authSvc authsvc.AuthService,
) (string, bool) {
	userID, err := GetUserID(ctx, authSvc)
	if err != nil {
		if errors.Is(err, ErrNoGitHubIdentity) {
			WriteError(w, http.StatusUnauthorized, "GitHub account not connected")
		} else {
			WriteError(w, http.StatusInternalServerError, "Failed to get user")
		}
		return "", false
	}
	return userID, true
}
