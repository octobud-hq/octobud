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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "test_token"

// newTestClient creates a clientImpl configured to use a test server.
func newTestClient(serverURL string) *clientImpl {
	return &clientImpl{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    serverURL,
		perPage:    defaultPerPage,
	}
}

func TestSetToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		serverStatus   int
		serverResponse string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "valid token",
			token:          "ghp_validtoken123",
			serverStatus:   http.StatusOK,
			serverResponse: `{"login": "testuser"}`,
			wantErr:        false,
		},
		{
			name:        "empty token",
			token:       "",
			wantErr:     true,
			errContains: "token cannot be empty",
		},
		{
			name:        "whitespace only token",
			token:       "   ",
			wantErr:     true,
			errContains: "token cannot be empty",
		},
		{
			name:           "invalid token (401)",
			token:          "invalid_token",
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"message": "Bad credentials"}`,
			wantErr:        true,
			errContains:    "token validation failed",
		},
		{
			name:           "server error (500)",
			token:          "some_token",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"message": "Internal Server Error"}`,
			wantErr:        true,
			errContains:    "API returned status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "/user", r.URL.Path)
					require.Equal(t, "Bearer "+tt.token, r.Header.Get("Authorization"))
					require.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
					require.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))

					w.WriteHeader(tt.serverStatus)
					_, err := w.Write([]byte(tt.serverResponse))
					assert.NoError(t, err, "failed to write response in test server")
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			err := client.SetToken(context.Background(), tt.token)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.token, client.token)
			}
		})
	}
}

func TestGetAuthHeaders(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected map[string]string
	}{
		{
			name:  "with valid token",
			token: testToken,
			expected: map[string]string{
				"Authorization":        "Bearer test_token",
				"Accept":               "application/vnd.github+json",
				"X-GitHub-Api-Version": "2022-11-28",
			},
		},
		{
			name:     "empty token returns empty map",
			token:    "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &clientImpl{token: tt.token}
			result := client.GetAuthHeaders()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestWithPerPage(t *testing.T) {
	tests := []struct {
		name     string
		perPage  int
		expected int
	}{
		{
			name:     "positive value",
			perPage:  100,
			expected: 100,
		},
		{
			name:     "zero keeps default",
			perPage:  0,
			expected: defaultPerPage,
		},
		{
			name:     "negative keeps default",
			perPage:  -10,
			expected: defaultPerPage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &clientImpl{perPage: defaultPerPage}
			result := client.WithPerPage(tt.perPage)
			resultImpl, ok := result.(*clientImpl)
			require.True(t, ok, "result should be *clientImpl")
			require.Equal(t, tt.expected, resultImpl.perPage)
		})
	}
}

func TestFetchNotifications(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		since          *time.Time
		serverStatus   int
		serverResponse string
		wantCount      int
		wantErr        bool
		errContains    string
		validateURL    func(*testing.T, *http.Request)
	}{
		{
			name:         "successful fetch without since",
			since:        nil,
			serverStatus: http.StatusOK,
			serverResponse: `[
				{"id": "1", "reason": "mention", "updated_at": "2024-01-15T10:00:00Z"},
				{"id": "2", "reason": "review_requested", "updated_at": "2024-01-15T11:00:00Z"}
			]`,
			wantCount: 2,
			wantErr:   false,
			validateURL: func(t *testing.T, r *http.Request) {
				require.NotContains(t, r.URL.RawQuery, "since=")
			},
		},
		{
			name:         "successful fetch with since parameter",
			since:        &now,
			serverStatus: http.StatusOK,
			serverResponse: `[
				{"id": "3", "reason": "assign", "updated_at": "2024-01-15T12:30:00Z"}
			]`,
			wantCount: 1,
			wantErr:   false,
			validateURL: func(t *testing.T, r *http.Request) {
				require.Contains(t, r.URL.RawQuery, "since=2024-01-15T12:00:00Z")
			},
		},
		{
			name:           "empty response",
			since:          nil,
			serverStatus:   http.StatusOK,
			serverResponse: `[]`,
			wantCount:      0,
			wantErr:        false,
		},
		{
			name:           "whitespace response",
			since:          nil,
			serverStatus:   http.StatusOK,
			serverResponse: `   `,
			wantCount:      0,
			wantErr:        false,
		},
		{
			name:           "server error",
			since:          nil,
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"message": "Server Error"}`,
			wantErr:        true,
			errContains:    "API returned status 500",
		},
		{
			name:           "unauthorized",
			since:          nil,
			serverStatus:   http.StatusUnauthorized,
			serverResponse: `{"message": "Bad credentials"}`,
			wantErr:        true,
			errContains:    "API returned status 401",
		},
		{
			name:           "invalid JSON response",
			since:          nil,
			serverStatus:   http.StatusOK,
			serverResponse: `{invalid json}`,
			wantErr:        true,
			errContains:    "decode notifications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestCount++
					require.Equal(t, "/notifications", r.URL.Path)
					require.Contains(t, r.URL.RawQuery, "per_page=")
					require.Contains(t, r.URL.RawQuery, "page=")

					if tt.validateURL != nil {
						tt.validateURL(t, r)
					}

					w.WriteHeader(tt.serverStatus)
					_, err := w.Write([]byte(tt.serverResponse))
					assert.NoError(t, err, "failed to write response in test server")
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			notifications, err := client.FetchNotifications(
				context.Background(),
				tt.since,
				nil,
				false,
			)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, notifications, tt.wantCount)
			}
		})
	}
}

func TestFetchNotifications_Pagination(t *testing.T) {
	pageResponses := []string{
		// Page 1 - full page (2 items matching perPage)
		`[{"id": "1", "reason": "mention"}, {"id": "2", "reason": "assign"}]`,
		// Page 2 - partial page (signals end)
		`[{"id": "3", "reason": "review_requested"}]`,
	}

	pageNum := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if pageNum < len(pageResponses) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(pageResponses[pageNum]))
			assert.NoError(t, err, "failed to write response in test server")
			pageNum++
		} else {
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.token = testToken
	client.perPage = 2 // Set perPage to match our test data

	notifications, err := client.FetchNotifications(context.Background(), nil, nil, false)

	require.NoError(t, err)
	require.Len(t, notifications, 3)
	require.Equal(t, "1", notifications[0].ID)
	require.Equal(t, "2", notifications[1].ID)
	require.Equal(t, "3", notifications[2].ID)
	require.Equal(t, 2, pageNum) // Should have made 2 requests
}

func TestFetchNotifications_RawFieldPopulated(t *testing.T) {
	serverResponse := `[{"id": "123", "reason": "mention", "updated_at": "2024-01-15T10:00:00Z"}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(serverResponse))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.token = testToken

	notifications, err := client.FetchNotifications(context.Background(), nil, nil, false)

	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.NotNil(t, notifications[0].Raw)

	// Verify Raw can be unmarshaled back
	var rawData map[string]interface{}
	err = json.Unmarshal(notifications[0].Raw, &rawData)
	require.NoError(t, err)
	require.Equal(t, "123", rawData["id"])
}

func TestFetchNotifications_UnreadOnlyParameter(t *testing.T) {
	tests := []struct {
		name        string
		unreadOnly  bool
		expectedAll string // Expected value of "all" query param
	}{
		{
			name:        "unreadOnly=false sends all=true",
			unreadOnly:  false,
			expectedAll: "all=true",
		},
		{
			name:        "unreadOnly=true sends all=false",
			unreadOnly:  true,
			expectedAll: "all=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.Contains(t, r.URL.RawQuery, tt.expectedAll,
						"expected %s in query string, got: %s", tt.expectedAll, r.URL.RawQuery)
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			_, err := client.FetchNotifications(context.Background(), nil, nil, tt.unreadOnly)
			require.NoError(t, err)
		})
	}
}

func TestFetchNotifications_BeforeParameter(t *testing.T) {
	beforeTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		before         *time.Time
		expectBefore   bool
		expectedBefore string
	}{
		{
			name:         "before=nil does not include before param",
			before:       nil,
			expectBefore: false,
		},
		{
			name:           "before time is included in query",
			before:         &beforeTime,
			expectBefore:   true,
			expectedBefore: "before=2024-01-15T12:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tt.expectBefore {
						require.Contains(
							t,
							r.URL.RawQuery,
							tt.expectedBefore,
							"expected %s in query string, got: %s",
							tt.expectedBefore,
							r.URL.RawQuery,
						)
					} else {
						require.NotContains(t, r.URL.RawQuery, "before=",
							"did not expect before param, got: %s", r.URL.RawQuery)
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			_, err := client.FetchNotifications(context.Background(), nil, tt.before, false)
			require.NoError(t, err)
		})
	}
}

func TestFetchSubjectRaw(t *testing.T) {
	tests := []struct {
		name           string
		subjectURL     string
		serverStatus   int
		serverResponse string
		wantNil        bool
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful fetch",
			subjectURL:     "/repos/owner/repo/issues/1",
			serverStatus:   http.StatusOK,
			serverResponse: `{"id": 1, "title": "Test Issue", "state": "open"}`,
			wantNil:        false,
			wantErr:        false,
		},
		{
			name:       "empty subject URL",
			subjectURL: "",
			wantNil:    true,
			wantErr:    false,
		},
		{
			name:       "whitespace subject URL",
			subjectURL: "   ",
			wantNil:    true,
			wantErr:    false,
		},
		{
			name:           "not found (404)",
			subjectURL:     "/repos/owner/repo/issues/999",
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"message": "Not Found"}`,
			wantErr:        true,
			errContains:    "subject status 404",
		},
		{
			name:           "server error (500)",
			subjectURL:     "/repos/owner/repo/issues/1",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `{"message": "Server Error"}`,
			wantErr:        true,
			errContains:    "subject status 500",
		},
		{
			name:           "invalid JSON response",
			subjectURL:     "/repos/owner/repo/issues/1",
			serverStatus:   http.StatusOK,
			serverResponse: `{invalid json`,
			wantErr:        true,
			errContains:    "unmarshal subject body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.serverStatus)
					_, err := w.Write([]byte(tt.serverResponse))
					assert.NoError(t, err, "failed to write response in test server")
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			var url string
			if tt.subjectURL != "" && tt.subjectURL != "   " {
				url = server.URL + tt.subjectURL
			} else {
				url = tt.subjectURL
			}

			raw, err := client.FetchSubjectRaw(context.Background(), url)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				if tt.wantNil {
					require.Nil(t, raw)
				} else {
					require.NotNil(t, raw)
				}
			}
		})
	}
}

func TestFetchIssueComments(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		number         int
		perPage        int
		page           int
		serverStatus   int
		serverResponse string
		wantCount      int
		wantErr        bool
		errContains    string
	}{
		{
			name:         "successful fetch",
			owner:        "testowner",
			repo:         "testrepo",
			number:       42,
			perPage:      10,
			page:         1,
			serverStatus: http.StatusOK,
			serverResponse: `[
				{"id": 1, "body": "Comment 1", "user": {"login": "user1"}},
				{"id": 2, "body": "Comment 2", "user": {"login": "user2"}}
			]`,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:           "empty comments",
			owner:          "testowner",
			repo:           "testrepo",
			number:         42,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusOK,
			serverResponse: `[]`,
			wantCount:      0,
			wantErr:        false,
		},
		{
			name:           "not found",
			owner:          "testowner",
			repo:           "testrepo",
			number:         999,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"message": "Not Found"}`,
			wantErr:        true,
			errContains:    "comments status 404",
		},
		{
			name:           "invalid JSON",
			owner:          "testowner",
			repo:           "testrepo",
			number:         42,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusOK,
			serverResponse: `{invalid}`,
			wantErr:        true,
			errContains:    "unmarshal comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := "/repos/" + tt.owner + "/" + tt.repo + "/issues/" + itoa(
						tt.number,
					) + "/comments"
					require.Equal(t, expectedPath, r.URL.Path)

					w.WriteHeader(tt.serverStatus)
					_, err := w.Write([]byte(tt.serverResponse))
					assert.NoError(t, err, "failed to write response in test server")
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			comments, err := client.FetchIssueComments(
				context.Background(),
				tt.owner,
				tt.repo,
				tt.number,
				tt.perPage,
				tt.page,
			)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, comments, tt.wantCount)
			}
		})
	}
}

func TestFetchPullRequestReviews(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		number         int
		perPage        int
		page           int
		serverStatus   int
		serverResponse string
		wantCount      int
		wantErr        bool
		errContains    string
	}{
		{
			name:         "successful fetch",
			owner:        "testowner",
			repo:         "testrepo",
			number:       42,
			perPage:      10,
			page:         1,
			serverStatus: http.StatusOK,
			serverResponse: `[
				{"id": 1, "state": "APPROVED", "user": {"login": "reviewer1"}},
				{"id": 2, "state": "CHANGES_REQUESTED", "user": {"login": "reviewer2"}}
			]`,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:           "empty reviews",
			owner:          "testowner",
			repo:           "testrepo",
			number:         42,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusOK,
			serverResponse: `[]`,
			wantCount:      0,
			wantErr:        false,
		},
		{
			name:           "not found",
			owner:          "testowner",
			repo:           "testrepo",
			number:         999,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"message": "Not Found"}`,
			wantErr:        true,
			errContains:    "reviews status 404",
		},
		{
			name:           "invalid JSON",
			owner:          "testowner",
			repo:           "testrepo",
			number:         42,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusOK,
			serverResponse: `{invalid}`,
			wantErr:        true,
			errContains:    "unmarshal reviews",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := "/repos/" + tt.owner + "/" + tt.repo + "/pulls/" + itoa(
						tt.number,
					) + "/reviews"
					require.Equal(t, expectedPath, r.URL.Path)

					w.WriteHeader(tt.serverStatus)
					_, err := w.Write([]byte(tt.serverResponse))
					assert.NoError(t, err, "failed to write response in test server")
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			reviews, err := client.FetchPullRequestReviews(
				context.Background(),
				tt.owner,
				tt.repo,
				tt.number,
				tt.perPage,
				tt.page,
			)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, reviews, tt.wantCount)
			}
		})
	}
}

func TestFetchTimeline(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		number         int
		perPage        int
		page           int
		serverStatus   int
		serverResponse string
		wantCount      int
		wantErr        bool
		errContains    string
	}{
		{
			name:         "successful fetch",
			owner:        "testowner",
			repo:         "testrepo",
			number:       42,
			perPage:      10,
			page:         1,
			serverStatus: http.StatusOK,
			serverResponse: `[
				{"event": "commented", "actor": {"login": "user1"}},
				{"event": "reviewed", "state": "approved", "user": {"login": "user2"}},
				{"event": "committed", "sha": "abc123"}
			]`,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:           "empty timeline",
			owner:          "testowner",
			repo:           "testrepo",
			number:         42,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusOK,
			serverResponse: `[]`,
			wantCount:      0,
			wantErr:        false,
		},
		{
			name:           "not found",
			owner:          "testowner",
			repo:           "testrepo",
			number:         999,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusNotFound,
			serverResponse: `{"message": "Not Found"}`,
			wantErr:        true,
			errContains:    "timeline status 404",
		},
		{
			name:           "invalid JSON",
			owner:          "testowner",
			repo:           "testrepo",
			number:         42,
			perPage:        10,
			page:           1,
			serverStatus:   http.StatusOK,
			serverResponse: `{invalid}`,
			wantErr:        true,
			errContains:    "unmarshal timeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := "/repos/" + tt.owner + "/" + tt.repo + "/issues/" + itoa(
						tt.number,
					) + "/timeline"
					require.Equal(t, expectedPath, r.URL.Path)

					w.WriteHeader(tt.serverStatus)
					_, err := w.Write([]byte(tt.serverResponse))
					assert.NoError(t, err, "failed to write response in test server")
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			timeline, err := client.FetchTimeline(
				context.Background(),
				tt.owner,
				tt.repo,
				tt.number,
				tt.perPage,
				tt.page,
			)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Len(t, timeline, tt.wantCount)
			}
		})
	}
}

func TestFetchTimeline_EventTypes(t *testing.T) {
	serverResponse := `[
		{"event": "commented", "body": "Test comment", "actor": {"login": "commenter"}},
		{"event": "reviewed", "state": "approved", "body": "LGTM", "user": {"login": "reviewer"}},
		{"event": "committed", "sha": "abc123def", "message": "Fix bug"},
		{"event": "merged", "commit_id": "xyz789", "actor": {"login": "merger"}}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(serverResponse))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.token = testToken

	timeline, err := client.FetchTimeline(context.Background(), "owner", "repo", 1, 100, 1)

	require.NoError(t, err)
	require.Len(t, timeline, 4)

	// Verify different event types are parsed correctly
	require.Equal(t, "commented", timeline[0].Event)
	require.Equal(t, "Test comment", timeline[0].Body)
	require.NotNil(t, timeline[0].Actor)
	require.Equal(t, "commenter", timeline[0].Actor.Login)

	require.Equal(t, "reviewed", timeline[1].Event)
	require.Equal(t, "approved", timeline[1].State)
	require.NotNil(t, timeline[1].User)
	require.Equal(t, "reviewer", timeline[1].User.Login)

	require.Equal(t, "committed", timeline[2].Event)
	require.Equal(t, "abc123def", timeline[2].SHA)
	require.Equal(t, "Fix bug", timeline[2].Message)

	require.Equal(t, "merged", timeline[3].Event)
	require.Equal(t, "xyz789", timeline[3].CommitID)
}

func TestNewClient(t *testing.T) {
	client := NewClient()

	require.NotNil(t, client)

	// Type assert to access internal fields
	impl, ok := client.(*clientImpl)
	require.True(t, ok)
	require.NotNil(t, impl.httpClient)
	require.Equal(t, githubAPIBase, impl.baseURL)
	require.Equal(t, defaultPerPage, impl.perPage)
}

func TestFetchNotifications_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	client.token = testToken

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.FetchNotifications(ctx, nil, nil, false)
	require.Error(t, err)
}

func TestRequestHeaders(t *testing.T) {
	tests := []struct {
		name        string
		fetchMethod func(client *clientImpl, server *httptest.Server) error
		expectedURL string
	}{
		{
			name: "FetchNotifications sets correct headers",
			fetchMethod: func(client *clientImpl, _ *httptest.Server) error {
				_, err := client.FetchNotifications(context.Background(), nil, nil, false)
				return err
			},
			expectedURL: "/notifications",
		},
		{
			name: "FetchSubjectRaw sets correct headers",
			fetchMethod: func(client *clientImpl, server *httptest.Server) error {
				_, err := client.FetchSubjectRaw(
					context.Background(),
					server.URL+"/repos/owner/repo/issues/1",
				)
				return err
			},
			expectedURL: "/repos/owner/repo/issues/1",
		},
		{
			name: "FetchIssueComments sets correct headers",
			fetchMethod: func(client *clientImpl, _ *httptest.Server) error {
				_, err := client.FetchIssueComments(context.Background(), "owner", "repo", 1, 10, 1)
				return err
			},
			expectedURL: "/repos/owner/repo/issues/1/comments",
		},
		{
			name: "FetchPullRequestReviews sets correct headers",
			fetchMethod: func(client *clientImpl, _ *httptest.Server) error {
				_, err := client.FetchPullRequestReviews(
					context.Background(),
					"owner",
					"repo",
					1,
					10,
					1,
				)
				return err
			},
			expectedURL: "/repos/owner/repo/pulls/1/reviews",
		},
		{
			name: "FetchTimeline sets correct headers",
			fetchMethod: func(client *clientImpl, _ *httptest.Server) error {
				_, err := client.FetchTimeline(context.Background(), "owner", "repo", 1, 10, 1)
				return err
			},
			expectedURL: "/repos/owner/repo/issues/1/timeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify common headers are set correctly
					require.Equal(t, "Bearer test_token", r.Header.Get("Authorization"))
					require.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
					require.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))

					w.WriteHeader(http.StatusOK)

					_, _ = w.Write([]byte(`[]`))
				}),
			)
			defer server.Close()

			client := newTestClient(server.URL)
			client.token = testToken

			err := tt.fetchMethod(client, server)
			require.NoError(t, err)
		})
	}
}

// itoa converts an int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}
