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

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/db/mocks"
	"github.com/octobud-hq/octobud/backend/internal/models"
)

func TestService_ListRepositories(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, []models.Repository)
	}{
		{
			name: "success returns repositories with data transformation",
			setupMock: func(m *mocks.MockStore) {
				now := time.Now().UTC()
				dbRepos := []db.Repository{
					{
						ID:        1,
						Name:      "repo1",
						FullName:  "owner/repo1",
						Private:   sql.NullBool{Bool: true, Valid: true},
						CreatedAt: sql.NullTime{Time: now, Valid: true},
					},
					{
						ID:        2,
						Name:      "repo2",
						FullName:  "owner/repo2",
						Private:   sql.NullBool{Bool: false, Valid: true},
						CreatedAt: sql.NullTime{Time: now, Valid: true},
					},
				}
				m.EXPECT().
					ListRepositories(gomock.Any(), "test-user-id").
					Return(dbRepos, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, repos []models.Repository) {
				require.Len(t, repos, 2)
				require.Equal(t, "repo1", repos[0].Name)
				require.Equal(t, "owner/repo1", repos[0].FullName)
				require.NotNil(t, repos[0].Private)
				require.True(t, *repos[0].Private)
				require.NotNil(t, repos[1].Private)
				require.False(t, *repos[1].Private)
			},
		},
		{
			name: "error wrapping database failure",
			setupMock: func(m *mocks.MockStore) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					ListRepositories(gomock.Any(), "test-user-id").
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToLoadRepositories))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ListRepositories(ctx, "test-user-id")

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_ListRepositoriesAsMap(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockStore)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, map[int64]db.Repository)
	}{
		{
			name: "success returns repositories as map",
			setupMock: func(m *mocks.MockStore) {
				dbRepos := []db.Repository{
					{ID: 1, Name: "repo1", FullName: "owner/repo1"},
					{ID: 2, Name: "repo2", FullName: "owner/repo2"},
				}
				m.EXPECT().
					ListRepositories(gomock.Any(), "test-user-id").
					Return(dbRepos, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, repoMap map[int64]db.Repository) {
				require.Len(t, repoMap, 2)
				require.Contains(t, repoMap, int64(1))
				require.Contains(t, repoMap, int64(2))
				require.Equal(t, "repo1", repoMap[1].Name)
				require.Equal(t, "repo2", repoMap[2].Name)
			},
		},
		{
			name: "error wrapping database failure",
			setupMock: func(m *mocks.MockStore) {
				dbError := errors.New("database connection failed")
				m.EXPECT().
					ListRepositories(gomock.Any(), "test-user-id").
					Return(nil, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.True(t, errors.Is(err, ErrFailedToListRepositoriesAsMap))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.ListRepositoriesAsMap(ctx, "test-user-id")

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestService_GetRepositoryByID(t *testing.T) {
	tests := []struct {
		name        string
		repoID      int64
		setupMock   func(*mocks.MockStore, int64)
		expectErr   bool
		checkErr    func(*testing.T, error)
		checkResult func(*testing.T, db.Repository)
	}{
		{
			name:   "success returns repository",
			repoID: 1,
			setupMock: func(m *mocks.MockStore, id int64) {
				expectedRepo := db.Repository{
					ID:       id,
					Name:     "repo1",
					FullName: "owner/repo1",
				}
				m.EXPECT().
					GetRepositoryByID(gomock.Any(), "test-user-id", id).
					Return(expectedRepo, nil)
			},
			expectErr: false,
			checkResult: func(t *testing.T, repo db.Repository) {
				require.Equal(t, int64(1), repo.ID)
				require.Equal(t, "repo1", repo.Name)
			},
		},
		{
			name:   "error passes through",
			repoID: 999,
			setupMock: func(m *mocks.MockStore, id int64) {
				dbError := sql.ErrNoRows
				m.EXPECT().
					GetRepositoryByID(gomock.Any(), "test-user-id", id).
					Return(db.Repository{}, dbError)
			},
			expectErr: true,
			checkErr: func(t *testing.T, err error) {
				require.ErrorIs(t, err, sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockStore(ctrl)
			tt.setupMock(mockQuerier, tt.repoID)
			service := NewService(mockQuerier)

			ctx := context.Background()
			result, err := service.GetRepositoryByID(ctx, "test-user-id", tt.repoID)

			if tt.expectErr {
				require.Error(t, err)
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestRepositoryFromModel_DataTransformation(t *testing.T) {
	// Test that RepositoryFromDB correctly transforms db.Repository to Repository
	now := time.Now().UTC()
	dbRepo := db.Repository{
		ID:            1,
		GithubID:      sql.NullInt64{Int64: 12345, Valid: true},
		NodeID:        sql.NullString{String: "node123", Valid: true},
		Name:          "repo1",
		FullName:      "owner/repo1",
		OwnerLogin:    sql.NullString{String: "owner", Valid: true},
		OwnerID:       sql.NullInt64{Int64: 67890, Valid: true},
		Private:       sql.NullBool{Bool: true, Valid: true},
		Description:   sql.NullString{String: "A test repo", Valid: true},
		HTMLURL:       sql.NullString{String: "https://github.com/owner/repo1", Valid: true},
		Fork:          sql.NullBool{Bool: false, Valid: true},
		Visibility:    sql.NullString{String: "private", Valid: true},
		DefaultBranch: sql.NullString{String: "main", Valid: true},
		Archived:      sql.NullBool{Bool: false, Valid: true},
		Disabled:      sql.NullBool{Bool: false, Valid: true},
		PushedAt:      sql.NullTime{Time: now, Valid: true},
		CreatedAt:     sql.NullTime{Time: now, Valid: true},
		UpdatedAt:     sql.NullTime{Time: now, Valid: true},
		Raw: db.NullRawMessage{
			RawMessage: json.RawMessage(`{"id":12345}`),
			Valid:      true,
		},
		OwnerAvatarURL: sql.NullString{
			String: "https://avatars.githubusercontent.com/u/67890",
			Valid:  true,
		},
		OwnerHTMLURL: sql.NullString{String: "https://github.com/owner", Valid: true},
	}

	result := models.RepositoryFromDB(dbRepo)

	require.Equal(t, int64(1), result.ID)
	require.NotNil(t, result.GithubID)
	require.Equal(t, int64(12345), *result.GithubID)
	require.NotNil(t, result.NodeID)
	require.Equal(t, "node123", *result.NodeID)
	require.Equal(t, "repo1", result.Name)
	require.Equal(t, "owner/repo1", result.FullName)
	require.NotNil(t, result.Private)
	require.True(t, *result.Private)
	require.NotNil(t, result.Description)
	require.Equal(t, "A test repo", *result.Description)
	require.NotNil(t, result.PushedAt)
	require.Equal(t, now, *result.PushedAt)
	require.NotEmpty(t, result.Raw)
}
