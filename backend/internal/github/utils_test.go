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
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAuthorFromSubject(t *testing.T) {
	tests := []struct {
		name          string
		subjectJSON   json.RawMessage
		expectLogin   bool
		expectID      bool
		expectedLogin string
		expectedID    int64
	}{
		{
			name: "extract from user field",
			subjectJSON: json.RawMessage(`{
				"user": {
					"login": "octocat",
					"id": 12345
				}
			}`),
			expectLogin:   true,
			expectID:      true,
			expectedLogin: "octocat",
			expectedID:    12345,
		},
		{
			name: "extract from sender field",
			subjectJSON: json.RawMessage(`{
				"sender": {
					"login": "testuser",
					"id": 67890
				}
			}`),
			expectLogin:   true,
			expectID:      true,
			expectedLogin: "testuser",
			expectedID:    67890,
		},
		{
			name: "user field takes precedence over sender",
			subjectJSON: json.RawMessage(`{
				"user": {
					"login": "user1",
					"id": 111
				},
				"sender": {
					"login": "sender1",
					"id": 222
				}
			}`),
			expectLogin:   true,
			expectID:      true,
			expectedLogin: "user1",
			expectedID:    111,
		},
		{
			name: "missing login field",
			subjectJSON: json.RawMessage(`{
				"user": {
					"id": 12345
				}
			}`),
			expectLogin: false,
			expectID:    true,
			expectedID:  12345,
		},
		{
			name: "missing id field",
			subjectJSON: json.RawMessage(`{
				"user": {
					"login": "octocat"
				}
			}`),
			expectLogin:   true,
			expectID:      false,
			expectedLogin: "octocat",
		},
		{
			name: "empty login string",
			subjectJSON: json.RawMessage(`{
				"user": {
					"login": "",
					"id": 12345
				}
			}`),
			expectLogin: false,
			expectID:    true,
			expectedID:  12345,
		},
		{
			name:        "no user or sender field",
			subjectJSON: json.RawMessage(`{"title": "test"}`),
			expectLogin: false,
			expectID:    false,
		},
		{
			name:        "invalid JSON",
			subjectJSON: json.RawMessage(`{invalid json}`),
			expectLogin: false,
			expectID:    false,
		},
		{
			name: "user field is not an object",
			subjectJSON: json.RawMessage(`{
				"user": "not an object"
			}`),
			expectLogin: false,
			expectID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login, id := ExtractAuthorFromSubject(tt.subjectJSON)

			if tt.expectLogin {
				if !login.Valid {
					t.Errorf("expected login to be valid, got invalid")
				} else if login.String != tt.expectedLogin {
					t.Errorf("expected login %q, got %q", tt.expectedLogin, login.String)
				}
			} else {
				if login.Valid {
					t.Errorf("expected login to be invalid, got valid: %q", login.String)
				}
			}

			if tt.expectID {
				if !id.Valid {
					t.Errorf("expected id to be valid, got invalid")
				} else if id.Int64 != tt.expectedID {
					t.Errorf("expected id %d, got %d", tt.expectedID, id.Int64)
				}
			} else {
				if id.Valid {
					t.Errorf("expected id to be invalid, got valid: %d", id.Int64)
				}
			}
		})
	}
}

func TestExtractAuthorFromSubject_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name        string
		subjectJSON json.RawMessage
		expectValid bool
	}{
		{
			name: "PR subject with user",
			subjectJSON: json.RawMessage(`{
				"id": 123,
				"number": 42,
				"title": "Test PR",
				"user": {
					"login": "pr-author",
					"id": 999
				}
			}`),
			expectValid: true,
		},
		{
			name: "Issue subject with user",
			subjectJSON: json.RawMessage(`{
				"id": 456,
				"number": 10,
				"title": "Test Issue",
				"user": {
					"login": "issue-author",
					"id": 888
				}
			}`),
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login, id := ExtractAuthorFromSubject(tt.subjectJSON)

			if tt.expectValid {
				if !login.Valid || !id.Valid {
					t.Errorf(
						"expected both login and id to be valid, got login=%v, id=%v",
						login.Valid,
						id.Valid,
					)
				}
			}
		})
	}
}

func TestExtractSubjectNumber(t *testing.T) {
	tests := []struct {
		name        string
		subjectJSON json.RawMessage
		expectValid bool
		expectedNum int32
	}{
		{
			name: "PR with number field",
			subjectJSON: json.RawMessage(`{
				"number": 42
			}`),
			expectValid: true,
			expectedNum: 42,
		},
		{
			name: "Issue with number field",
			subjectJSON: json.RawMessage(`{
				"number": 123
			}`),
			expectValid: true,
			expectedNum: 123,
		},
		{
			name:        "missing number field",
			subjectJSON: json.RawMessage(`{"title": "test"}`),
			expectValid: false,
		},
		{
			name:        "invalid JSON",
			subjectJSON: json.RawMessage(`{invalid json}`),
			expectValid: false,
		},
		{
			name: "number is not a number",
			subjectJSON: json.RawMessage(`{
				"number": "not a number"
			}`),
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSubjectNumber(tt.subjectJSON)

			if tt.expectValid {
				if !result.Valid {
					t.Errorf("expected number to be valid, got invalid")
				} else if result.Int32 != tt.expectedNum {
					t.Errorf("expected number %d, got %d", tt.expectedNum, result.Int32)
				}
			} else {
				if result.Valid {
					t.Errorf("expected number to be invalid, got valid: %d", result.Int32)
				}
			}
		})
	}
}

func TestExtractSubjectMerged(t *testing.T) {
	tests := []struct {
		name        string
		subjectJSON json.RawMessage
		want        sql.NullBool
	}{
		{
			name:        "PR with merged true",
			subjectJSON: json.RawMessage(`{"merged": true, "state": "closed"}`),
			want:        sql.NullBool{Bool: true, Valid: true},
		},
		{
			name:        "PR with merged false",
			subjectJSON: json.RawMessage(`{"merged": false, "state": "open"}`),
			want:        sql.NullBool{Bool: false, Valid: true},
		},
		{
			name:        "Issue without merged field",
			subjectJSON: json.RawMessage(`{"state": "open"}`),
			want:        sql.NullBool{Valid: false},
		},
		{
			name:        "Invalid JSON",
			subjectJSON: json.RawMessage(`{invalid json}`),
			want:        sql.NullBool{Valid: false},
		},
		{
			name:        "Empty JSON",
			subjectJSON: json.RawMessage(`{}`),
			want:        sql.NullBool{Valid: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSubjectMerged(tt.subjectJSON)
			if result.Valid != tt.want.Valid {
				t.Errorf("ExtractSubjectMerged() Valid = %v, want %v", result.Valid, tt.want.Valid)
			}
			if result.Valid && result.Bool != tt.want.Bool {
				t.Errorf("ExtractSubjectMerged() Bool = %v, want %v", result.Bool, tt.want.Bool)
			}
		})
	}
}

func TestExtractSubjectStateReason(t *testing.T) {
	tests := []struct {
		name        string
		subjectJSON json.RawMessage
		want        sql.NullString
	}{
		{
			name:        "Issue with state_reason completed",
			subjectJSON: json.RawMessage(`{"state": "closed", "state_reason": "completed"}`),
			want:        sql.NullString{String: "completed", Valid: true},
		},
		{
			name:        "Issue with state_reason not_planned",
			subjectJSON: json.RawMessage(`{"state": "closed", "state_reason": "not_planned"}`),
			want:        sql.NullString{String: "not_planned", Valid: true},
		},
		{
			name:        "PR without state_reason field",
			subjectJSON: json.RawMessage(`{"state": "open", "merged": false}`),
			want:        sql.NullString{Valid: false},
		},
		{
			name:        "Empty state_reason",
			subjectJSON: json.RawMessage(`{"state": "closed", "state_reason": ""}`),
			want:        sql.NullString{Valid: false},
		},
		{
			name:        "Invalid JSON",
			subjectJSON: json.RawMessage(`{invalid json}`),
			want:        sql.NullString{Valid: false},
		},
		{
			name:        "Empty JSON",
			subjectJSON: json.RawMessage(`{}`),
			want:        sql.NullString{Valid: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSubjectStateReason(tt.subjectJSON)
			if result.Valid != tt.want.Valid {
				t.Errorf(
					"ExtractSubjectStateReason() Valid = %v, want %v",
					result.Valid,
					tt.want.Valid,
				)
			}
			if result.Valid && result.String != tt.want.String {
				t.Errorf(
					"ExtractSubjectStateReason() String = %v, want %v",
					result.String,
					tt.want.String,
				)
			}
		})
	}
}

func TestExtractSubjectState(t *testing.T) {
	tests := []struct {
		name          string
		subjectJSON   json.RawMessage
		expectValid   bool
		expectedState string
	}{
		{
			name: "PR with open state",
			subjectJSON: json.RawMessage(`{
				"state": "open"
			}`),
			expectValid:   true,
			expectedState: "open",
		},
		{
			name: "Issue with closed state",
			subjectJSON: json.RawMessage(`{
				"state": "closed"
			}`),
			expectValid:   true,
			expectedState: "closed",
		},
		{
			name:        "missing state field",
			subjectJSON: json.RawMessage(`{"title": "test"}`),
			expectValid: false,
		},
		{
			name:        "invalid JSON",
			subjectJSON: json.RawMessage(`{invalid json}`),
			expectValid: false,
		},
		{
			name: "state is not a string",
			subjectJSON: json.RawMessage(`{
				"state": 123
			}`),
			expectValid: false,
		},
		{
			name: "empty state string",
			subjectJSON: json.RawMessage(`{
				"state": ""
			}`),
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSubjectState(tt.subjectJSON)

			if tt.expectValid {
				if !result.Valid {
					t.Errorf("expected state to be valid, got invalid")
				} else if result.String != tt.expectedState {
					t.Errorf("expected state %q, got %q", tt.expectedState, result.String)
				}
			} else {
				if result.Valid {
					t.Errorf("expected state to be invalid, got valid: %q", result.String)
				}
			}
		})
	}
}

func TestExtractPullRequestData(t *testing.T) {
	tests := []struct {
		name         string
		subjectJSON  json.RawMessage
		expectErr    bool
		expectErrMsg string
		validateData func(*testing.T, *PullRequestData)
	}{
		{
			name: "valid PR with all fields",
			subjectJSON: json.RawMessage(`{
				"id": 123456,
				"node_id": "PR_kwDOABC123",
				"number": 42,
				"title": "Test PR",
				"state": "open",
				"draft": false,
				"merged": false,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z",
				"closed_at": null,
				"merged_at": null,
				"user": {
					"login": "testuser",
					"id": 789
				}
			}`),
			expectErr: false,
			validateData: func(t *testing.T, data *PullRequestData) {
				require.NotNil(t, data)
				require.Equal(t, 42, data.Number)
				require.NotNil(t, data.GithubID)
				require.Equal(t, int64(123456), *data.GithubID)
				require.NotNil(t, data.NodeID)
				require.Equal(t, "PR_kwDOABC123", *data.NodeID)
				require.NotNil(t, data.Title)
				require.Equal(t, "Test PR", *data.Title)
				require.NotNil(t, data.State)
				require.Equal(t, "open", *data.State)
				require.NotNil(t, data.Draft)
				require.False(t, *data.Draft)
				require.NotNil(t, data.Merged)
				require.False(t, *data.Merged)
				require.NotNil(t, data.AuthorLogin)
				require.Equal(t, "testuser", *data.AuthorLogin)
				require.NotNil(t, data.AuthorID)
				require.Equal(t, int64(789), *data.AuthorID)
				require.NotNil(t, data.CreatedAt)
				require.NotNil(t, data.UpdatedAt)
				require.Nil(t, data.ClosedAt)
				require.Nil(t, data.MergedAt)
			},
		},
		{
			name: "PR missing number field",
			subjectJSON: json.RawMessage(`{
				"id": 123456,
				"title": "Test PR"
			}`),
			expectErr:    true,
			expectErrMsg: "PR subject missing number field",
		},
		{
			name: "PR with minimal fields",
			subjectJSON: json.RawMessage(`{
				"number": 10
			}`),
			expectErr: false,
			validateData: func(t *testing.T, data *PullRequestData) {
				require.NotNil(t, data)
				require.Equal(t, 10, data.Number)
				require.Nil(t, data.GithubID)
				require.Nil(t, data.NodeID)
				require.Nil(t, data.Title)
				require.Nil(t, data.State)
				require.Nil(t, data.Draft)
				require.Nil(t, data.Merged)
				require.Nil(t, data.AuthorLogin)
				require.Nil(t, data.AuthorID)
				require.Nil(t, data.CreatedAt)
				require.Nil(t, data.UpdatedAt)
				require.Nil(t, data.ClosedAt)
				require.Nil(t, data.MergedAt)
			},
		},
		{
			name: "PR with null timestamps",
			subjectJSON: json.RawMessage(`{
				"number": 5,
				"created_at": null,
				"updated_at": null,
				"closed_at": null,
				"merged_at": null
			}`),
			expectErr: false,
			validateData: func(t *testing.T, data *PullRequestData) {
				require.NotNil(t, data)
				require.Equal(t, 5, data.Number)
				require.Nil(t, data.CreatedAt)
				require.Nil(t, data.UpdatedAt)
				require.Nil(t, data.ClosedAt)
				require.Nil(t, data.MergedAt)
			},
		},
		{
			name:        "PR with invalid JSON",
			subjectJSON: json.RawMessage(`{invalid json}`),
			expectErr:   true,
		},
		{
			name: "PR with user but missing login",
			subjectJSON: json.RawMessage(`{
				"number": 7,
				"user": {
					"id": 555
				}
			}`),
			expectErr: false,
			validateData: func(t *testing.T, data *PullRequestData) {
				require.NotNil(t, data)
				require.Equal(t, 7, data.Number)
				require.Nil(t, data.AuthorLogin)
				require.NotNil(t, data.AuthorID)
				require.Equal(t, int64(555), *data.AuthorID)
			},
		},
		{
			name: "PR with merged state",
			subjectJSON: json.RawMessage(`{
				"number": 8,
				"state": "closed",
				"merged": true,
				"merged_at": "2024-01-15T12:00:00Z"
			}`),
			expectErr: false,
			validateData: func(t *testing.T, data *PullRequestData) {
				require.NotNil(t, data)
				require.Equal(t, 8, data.Number)
				require.NotNil(t, data.State)
				require.Equal(t, "closed", *data.State)
				require.NotNil(t, data.Merged)
				require.True(t, *data.Merged)
				require.NotNil(t, data.MergedAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ExtractPullRequestData(tt.subjectJSON)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectErrMsg != "" {
					require.Contains(t, err.Error(), tt.expectErrMsg)
				}
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.NotNil(t, data)
				if tt.validateData != nil {
					tt.validateData(t, data)
				}
			}
		})
	}
}

func TestExtractPullRequestData_TimeParsing(t *testing.T) {
	subjectJSON := json.RawMessage(`{
		"number": 1,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"closed_at": "2024-01-03T00:00:00Z",
		"merged_at": "2024-01-04T00:00:00Z"
	}`)

	data, err := ExtractPullRequestData(subjectJSON)
	require.NoError(t, err)
	require.NotNil(t, data)
	require.NotNil(t, data.CreatedAt)
	require.NotNil(t, data.UpdatedAt)
	require.NotNil(t, data.ClosedAt)
	require.NotNil(t, data.MergedAt)

	// Verify times are parsed correctly
	expectedCreated, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.NoError(t, err, "failed to parse expectedCreated time")
	require.WithinDuration(t, expectedCreated, *data.CreatedAt, time.Second)

	expectedUpdated, err := time.Parse(time.RFC3339, "2024-01-02T00:00:00Z")
	assert.NoError(t, err, "failed to parse expectedUpdated time")
	require.WithinDuration(t, expectedUpdated, *data.UpdatedAt, time.Second)

	expectedClosed, err := time.Parse(time.RFC3339, "2024-01-03T00:00:00Z")
	assert.NoError(t, err, "failed to parse expectedClosed time")
	require.WithinDuration(t, expectedClosed, *data.ClosedAt, time.Second)

	expectedMerged, err := time.Parse(time.RFC3339, "2024-01-04T00:00:00Z")
	assert.NoError(t, err, "failed to parse expectedMerged time")
	require.WithinDuration(t, expectedMerged, *data.MergedAt, time.Second)
}

func TestExtractPullRequestData_InvalidTimeFormat(t *testing.T) {
	subjectJSON := json.RawMessage(`{
		"number": 1,
		"created_at": "invalid-time-format"
	}`)

	data, err := ExtractPullRequestData(subjectJSON)
	require.NoError(t, err)
	require.NotNil(t, data)
	// Invalid time format should result in nil time pointer
	require.Nil(t, data.CreatedAt)
}

func TestExtractSubjectInfo(t *testing.T) {
	tests := []struct {
		name           string
		subjectURL     string
		subjectRaw     json.RawMessage
		expectedOwner  string
		expectedRepo   string
		expectedNumber int
		expectErr      bool
	}{
		{
			name:           "issue from API URL",
			subjectURL:     "https://api.github.com/repos/owner/repo/issues/123",
			subjectRaw:     nil,
			expectedOwner:  "owner",
			expectedRepo:   "repo",
			expectedNumber: 123,
			expectErr:      false,
		},
		{
			name:           "pull request from API URL",
			subjectURL:     "https://api.github.com/repos/owner/repo/pulls/456",
			subjectRaw:     nil,
			expectedOwner:  "owner",
			expectedRepo:   "repo",
			expectedNumber: 456,
			expectErr:      false,
		},
		{
			name:           "discussion from API URL",
			subjectURL:     "https://api.github.com/repos/owner/repo/discussions/789",
			subjectRaw:     nil,
			expectedOwner:  "owner",
			expectedRepo:   "repo",
			expectedNumber: 789,
			expectErr:      false,
		},
		{
			name:       "issue from HTML URL in raw JSON",
			subjectURL: "",
			subjectRaw: json.RawMessage(`{
				"number": 10,
				"html_url": "https://github.com/test-owner/test-repo/issues/10"
			}`),
			expectedOwner:  "test-owner",
			expectedRepo:   "test-repo",
			expectedNumber: 10,
			expectErr:      false,
		},
		{
			name:       "pull request from HTML URL in raw JSON",
			subjectURL: "",
			subjectRaw: json.RawMessage(`{
				"number": 20,
				"html_url": "https://github.com/test-owner/test-repo/pull/20"
			}`),
			expectedOwner:  "test-owner",
			expectedRepo:   "test-repo",
			expectedNumber: 20,
			expectErr:      false,
		},
		{
			name:       "discussion from HTML URL in raw JSON",
			subjectURL: "",
			subjectRaw: json.RawMessage(`{
				"number": 30,
				"html_url": "https://github.com/test-owner/test-repo/discussions/30"
			}`),
			expectedOwner:  "test-owner",
			expectedRepo:   "test-repo",
			expectedNumber: 30,
			expectErr:      false,
		},
		{
			name:       "no URL or raw JSON",
			subjectURL: "",
			subjectRaw: nil,
			expectErr:  true,
		},
		{
			name:       "raw JSON missing number",
			subjectURL: "",
			subjectRaw: json.RawMessage(`{
				"html_url": "https://github.com/owner/repo/issues/10"
			}`),
			expectErr: true,
		},
		{
			name:       "raw JSON missing html_url",
			subjectURL: "",
			subjectRaw: json.RawMessage(`{
				"number": 10
			}`),
			expectErr: true,
		},
		{
			name:       "API URL takes precedence over raw JSON",
			subjectURL: "https://api.github.com/repos/api-owner/api-repo/issues/100",
			subjectRaw: json.RawMessage(`{
				"number": 200,
				"html_url": "https://github.com/raw-owner/raw-repo/issues/200"
			}`),
			expectedOwner:  "api-owner",
			expectedRepo:   "api-repo",
			expectedNumber: 100,
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractSubjectInfo(tt.subjectURL, tt.subjectRaw)

			if tt.expectErr {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tt.expectedOwner, result.Owner)
				require.Equal(t, tt.expectedRepo, result.Repo)
				require.Equal(t, tt.expectedNumber, result.Number)
			}
		})
	}
}
