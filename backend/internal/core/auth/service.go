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

// Package auth provides the auth service.
// Authentication is based on GitHub identity - no local username/password.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

// Error definitions
var (
	ErrUserNotFound = errors.New("user not found")
)

//go:generate mockgen -source=service.go -destination=mocks/mock_service.go -package=mocks

// AuthService is the interface for the auth service.
//
//nolint:revive // exported type name stutters with package name
type AuthService interface {
	GetUser(ctx context.Context) (*models.User, error)
	EnsureUser(ctx context.Context) error
	UpdateGitHubIdentity(ctx context.Context, githubUserID, githubUsername string) error
	GetUserSyncSettings(ctx context.Context) (*models.SyncSettings, error)
	UpdateUserSyncSettings(ctx context.Context, settings *models.SyncSettings) error
	GetUserUpdateSettings(ctx context.Context) (*models.UpdateSettings, error)
	UpdateUserUpdateSettings(ctx context.Context, settings *models.UpdateSettings) error
	HasSyncSettings(ctx context.Context) (bool, error)
	HasGitHubIdentity(ctx context.Context) (bool, error)
	UpdateUserMutedUntil(ctx context.Context, mutedUntil sql.NullTime) (*models.User, error)
}

// Service provides business logic for authentication operations
type Service struct {
	queries db.Store
}

// NewService constructs a Service backed by the provided queries
func NewService(queries db.Store) *Service {
	return &Service{
		queries: queries,
	}
}

// GetUser retrieves the single user from the database
func (s *Service) GetUser(ctx context.Context) (*models.User, error) {
	user, err := s.queries.GetUser(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	syncSettings, err := models.SyncSettingsFromJSON(user.SyncSettings.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sync settings: %w", err)
	}

	retentionSettings, err := models.RetentionSettingsFromJSON(user.RetentionSettings.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse retention settings: %w", err)
	}

	updateSettings, err := models.UpdateSettingsFromJSON(user.UpdateSettings.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update settings: %w", err)
	}

	return &models.User{
		ID:                user.ID,
		GithubUserID:      user.GithubUserID.String,
		GithubUsername:    user.GithubUsername.String,
		SyncSettings:      syncSettings,
		RetentionSettings: retentionSettings,
		UpdateSettings:    updateSettings,
		MutedUntil:        user.MutedUntil,
	}, nil
}

// EnsureUser creates the user record if it doesn't exist
// This is called on startup to ensure the single user record exists
func (s *Service) EnsureUser(ctx context.Context) error {
	_, err := s.GetUser(ctx)
	if err == nil {
		// User already exists
		return nil
	}

	if !errors.Is(err, ErrUserNotFound) {
		// Some other error occurred
		return fmt.Errorf("failed to check for existing user: %w", err)
	}

	// User doesn't exist, create the record
	_, err = s.queries.CreateUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateGitHubIdentity updates the user's GitHub identity
// This is called when the user connects their GitHub account
func (s *Service) UpdateGitHubIdentity(
	ctx context.Context,
	githubUserID, githubUsername string,
) error {
	_, err := s.queries.UpdateUserGitHubIdentity(ctx, db.UpdateUserGitHubIdentityParams{
		GithubUserID:   sql.NullString{String: githubUserID, Valid: githubUserID != ""},
		GithubUsername: sql.NullString{String: githubUsername, Valid: githubUsername != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to update GitHub identity: %w", err)
	}
	return nil
}

// GetUserSyncSettings retrieves the user's sync settings
func (s *Service) GetUserSyncSettings(ctx context.Context) (*models.SyncSettings, error) {
	user, err := s.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	return user.SyncSettings, nil
}

// UpdateUserSyncSettings updates the user's sync settings
func (s *Service) UpdateUserSyncSettings(ctx context.Context, settings *models.SyncSettings) error {
	jsonData, err := settings.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal sync settings: %w", err)
	}

	var rawMessage db.NullRawMessage
	if len(jsonData) > 0 {
		rawMessage = db.NullRawMessage{
			RawMessage: jsonData,
			Valid:      true,
		}
	}

	_, err = s.queries.UpdateUserSyncSettings(ctx, rawMessage)
	if err != nil {
		return fmt.Errorf("failed to update sync settings: %w", err)
	}
	return nil
}

// HasSyncSettings checks if the user has sync settings configured
func (s *Service) HasSyncSettings(ctx context.Context) (bool, error) {
	settings, err := s.GetUserSyncSettings(ctx)
	if err != nil {
		return false, err
	}
	return settings != nil && settings.SetupCompleted, nil
}

// GetUserUpdateSettings retrieves the user's update settings
func (s *Service) GetUserUpdateSettings(ctx context.Context) (*models.UpdateSettings, error) {
	user, err := s.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if user.UpdateSettings == nil {
		return models.DefaultUpdateSettings(), nil
	}
	return user.UpdateSettings, nil
}

// UpdateUserUpdateSettings updates the user's update settings
func (s *Service) UpdateUserUpdateSettings(ctx context.Context, settings *models.UpdateSettings) error {
	jsonData, err := settings.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal update settings: %w", err)
	}

	var rawMessage db.NullRawMessage
	if len(jsonData) > 0 {
		rawMessage = db.NullRawMessage{
			RawMessage: jsonData,
			Valid:      true,
		}
	}

	_, err = s.queries.UpdateUserUpdateSettings(ctx, rawMessage)
	if err != nil {
		return fmt.Errorf("failed to update update settings: %w", err)
	}
	return nil
}

// HasGitHubIdentity checks if the user has connected their GitHub account
func (s *Service) HasGitHubIdentity(ctx context.Context) (bool, error) {
	user, err := s.GetUser(ctx)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}
	return user.GithubUserID != "", nil
}

// UpdateUserMutedUntil updates the user's mute status
func (s *Service) UpdateUserMutedUntil(
	ctx context.Context,
	mutedUntil sql.NullTime,
) (*models.User, error) {
	updatedUser, err := s.queries.UpdateUserMutedUntil(ctx, mutedUntil)
	if err != nil {
		return nil, fmt.Errorf("failed to update muted until: %w", err)
	}

	syncSettings, err := models.SyncSettingsFromJSON(updatedUser.SyncSettings.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sync settings: %w", err)
	}

	retentionSettings, err := models.RetentionSettingsFromJSON(
		updatedUser.RetentionSettings.RawMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse retention settings: %w", err)
	}

	updateSettings, err := models.UpdateSettingsFromJSON(updatedUser.UpdateSettings.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update settings: %w", err)
	}

	return &models.User{
		ID:                updatedUser.ID,
		GithubUserID:      updatedUser.GithubUserID.String,
		GithubUsername:    updatedUser.GithubUsername.String,
		SyncSettings:      syncSettings,
		RetentionSettings: retentionSettings,
		UpdateSettings:    updateSettings,
		MutedUntil:        updatedUser.MutedUntil,
	}, nil
}
