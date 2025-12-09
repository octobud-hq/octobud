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

// Package github provides the client for the GitHub API.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

const (
	defaultPerPage    = 50
	githubAPIBase     = "https://api.github.com"
	githubGraphQLBase = "https://api.github.com/graphql"
)

// clientImpl wraps calls to the GitHub API for interacting with the Notifications API.
type clientImpl struct {
	httpClient *http.Client
	baseURL    string
	perPage    int
	token      string
}

// NewClient constructs an HTTP-backed GitHub client.
func NewClient() githubinterfaces.Client {
	return &clientImpl{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: githubAPIBase,
		perPage: defaultPerPage,
	}
}

// SetToken sets the GitHub token directly and validates it.
func (c *clientImpl) SetToken(ctx context.Context, token string) error {
	c.token = strings.TrimSpace(token)
	if c.token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Validate the token by making a test API call
	if err := c.validateToken(ctx); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	return nil
}

// validateToken makes a simple API call to verify the token works
func (c *clientImpl) validateToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/user", http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - log if we had a logger, but can't return it
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			// If we can't read the body, just return the status code
			return fmt.Errorf(
				"API returned status %d (failed to read body: %w)",
				resp.StatusCode,
				err,
			)
		}
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// WithPerPage allows the page size to be adjusted for fetching notifications.
func (c *clientImpl) WithPerPage(perPage int) githubinterfaces.Client {
	if perPage > 0 {
		c.perPage = perPage
	}
	return c
}

// GetAuthHeaders returns the authentication headers for GitHub API requests.
// Returns an empty map if no token is configured.
func (c *clientImpl) GetAuthHeaders() map[string]string {
	if c.token == "" {
		return map[string]string{}
	}
	return map[string]string{
		"Authorization":        "Bearer " + c.token,
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	}
}

// FetchNotifications retrieves notification threads updated since the given instant.
// When since is nil, GitHub returns the default notification window (typically 90 days).
// When before is nil, no upper bound is applied.
// When unreadOnly is false (the safe default), all=true is sent to GitHub to fetch all notifications.
// When unreadOnly is true, all=false is sent to only fetch unread notifications.
func (c *clientImpl) FetchNotifications(
	ctx context.Context,
	since *time.Time,
	before *time.Time,
	unreadOnly bool,
) ([]types.NotificationThread, error) {
	perPage := c.perPage
	if perPage <= 0 {
		perPage = defaultPerPage
	}

	var (
		allNotifications []types.NotificationThread
		page             = 1
	)

	// all=true fetches all notifications (read and unread)
	// all=false fetches only unread notifications
	// We invert unreadOnly to get the 'all' parameter value
	fetchAll := !unreadOnly

	for {
		// Build URL with query parameters
		url := fmt.Sprintf(
			"%s/notifications?all=%t&per_page=%d&page=%d",
			c.baseURL,
			fetchAll,
			perPage,
			page,
		)
		if since != nil {
			url += "&since=" + since.UTC().Format(time.RFC3339)
		}
		if before != nil {
			url += "&before=" + before.UTC().Format(time.RFC3339)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("github: create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("github: fetch notifications page %d: %w", page, err)
		}

		payload, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - log if we had a logger, but can't return it
			_ = closeErr
		}

		if err != nil {
			return nil, fmt.Errorf("github: read response body page %d: %w", page, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(
				"github: API returned status %d: %s",
				resp.StatusCode,
				string(payload),
			)
		}

		if len(bytes.TrimSpace(payload)) == 0 {
			break
		}

		var pageItems []types.NotificationThread
		if err := json.Unmarshal(payload, &pageItems); err != nil {
			return nil, fmt.Errorf("github: decode notifications page %d: %w", page, err)
		}

		if len(pageItems) == 0 {
			break
		}

		for i := range pageItems {
			raw, err := json.Marshal(pageItems[i])
			if err != nil {
				return nil, fmt.Errorf("github: encode raw notification payload: %w", err)
			}
			pageItems[i].Raw = raw
		}

		allNotifications = append(allNotifications, pageItems...)

		if len(pageItems) < perPage {
			break
		}

		page++
	}

	return allNotifications, nil
}

// FetchSubjectRaw retrieves the raw JSON payload for a notification subject.
func (c *clientImpl) FetchSubjectRaw(
	ctx context.Context,
	subjectURL string,
) (json.RawMessage, error) {
	if strings.TrimSpace(subjectURL) == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", subjectURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("github: create subject request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: fetch subject: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - log if we had a logger, but can't return it
			_ = closeErr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: read subject body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: subject status %d: %s", resp.StatusCode, string(body))
	}

	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("github: unmarshal subject body: %w", err)
	}

	return raw, nil
}

// FetchIssueComments retrieves comments for an issue or pull request.
func (c *clientImpl) FetchIssueComments(
	ctx context.Context,
	owner, repo string,
	number, perPage, page int,
) ([]types.IssueComment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=%d&page=%d",
		c.baseURL, owner, repo, number, perPage, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("github: create comments request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: fetch comments: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - log if we had a logger, but can't return it
			_ = closeErr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: read comments body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: comments status %d: %s", resp.StatusCode, string(body))
	}

	var comments []types.IssueComment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("github: unmarshal comments: %w", err)
	}

	return comments, nil
}

// FetchPullRequestReviews retrieves reviews for a pull request.
func (c *clientImpl) FetchPullRequestReviews(
	ctx context.Context,
	owner, repo string,
	number, perPage, page int,
) ([]types.PullRequestReview, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews?per_page=%d&page=%d",
		c.baseURL, owner, repo, number, perPage, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("github: create reviews request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: fetch reviews: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - log if we had a logger, but can't return it
			_ = closeErr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: read reviews body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: reviews status %d: %s", resp.StatusCode, string(body))
	}

	var reviews []types.PullRequestReview
	if err := json.Unmarshal(body, &reviews); err != nil {
		return nil, fmt.Errorf("github: unmarshal reviews: %w", err)
	}

	return reviews, nil
}

// FetchTimeline retrieves timeline events for an issue or pull request.
func (c *clientImpl) FetchTimeline(
	ctx context.Context,
	owner, repo string,
	number, perPage, page int,
) ([]types.TimelineEvent, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/timeline?per_page=%d&page=%d",
		c.baseURL, owner, repo, number, perPage, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("github: create timeline request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: fetch timeline: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Error closing response body - log if we had a logger, but can't return it
			_ = closeErr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: read timeline body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: timeline status %d: %s", resp.StatusCode, string(body))
	}

	var timeline []types.TimelineEvent
	if err := json.Unmarshal(body, &timeline); err != nil {
		return nil, fmt.Errorf("github: unmarshal timeline: %w", err)
	}

	return timeline, nil
}

// GraphQLRequest represents a GraphQL request payload.
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

// DiscussionComment represents a comment from a discussion via GraphQL.
type DiscussionComment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	URL string `json:"url"`
}

// DiscussionCommentsResponse represents the GraphQL response for discussion comments.
type DiscussionCommentsResponse struct {
	Repository struct {
		Discussion struct {
			Comments struct {
				Nodes    []DiscussionComment `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"comments"`
		} `json:"discussion"`
	} `json:"repository"`
}

// FetchDiscussionComments retrieves comments for a discussion using GraphQL API.
// Returns timeline events, hasNextPage, endCursor, and error.
func (c *clientImpl) FetchDiscussionComments( //nolint:gocritic // Prefer in-line naming of results.
	ctx context.Context,
	owner, repo string,
	number, first int,
	after string, // cursor for pagination
) ([]types.TimelineEvent, bool, string, error) {
	// GraphQL query to fetch discussion comments
	query := `
		query GetDiscussionComments($owner: String!, $repo: String!, $number: Int!, $first: Int!, $after: String) {
			repository(owner: $owner, name: $repo) {
				discussion(number: $number) {
					comments(first: $first, after: $after, orderBy: {field: CREATED_AT, direction: DESC}) {
						nodes {
							id
							body
							createdAt
							updatedAt
							author {
								login
								avatarUrl
							}
							url
						}
						pageInfo {
							hasNextPage
							endCursor
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"number": number,
		"first":  first,
	}
	if after != "" {
		variables["after"] = after
	}

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, false, "", fmt.Errorf("github: marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		githubGraphQLBase,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, false, "", fmt.Errorf("github: create graphql request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, "", fmt.Errorf("github: execute graphql request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, "", fmt.Errorf("github: read graphql response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, "", fmt.Errorf(
			"github: graphql status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return nil, false, "", fmt.Errorf("github: unmarshal graphql response: %w", err)
	}

	// Check for GraphQL errors
	if len(graphqlResp.Errors) > 0 {
		var errorMsgs []string
		for _, err := range graphqlResp.Errors {
			errorMsgs = append(errorMsgs, err.Message)
		}
		return nil, false, "", fmt.Errorf("github: graphql errors: %v", errorMsgs)
	}

	// Parse the data
	var data DiscussionCommentsResponse
	if err := json.Unmarshal(graphqlResp.Data, &data); err != nil {
		return nil, false, "", fmt.Errorf("github: unmarshal graphql data: %w", err)
	}

	comments := data.Repository.Discussion.Comments.Nodes
	hasNextPage := data.Repository.Discussion.Comments.PageInfo.HasNextPage
	endCursor := data.Repository.Discussion.Comments.PageInfo.EndCursor

	// Convert DiscussionComment to TimelineEvent
	timelineEvents := make([]types.TimelineEvent, len(comments))
	for i, comment := range comments {
		createdAt := comment.CreatedAt
		updatedAt := comment.UpdatedAt
		timelineEvents[i] = types.TimelineEvent{
			Event:     "commented",
			ID:        json.RawMessage(fmt.Sprintf(`%q`, comment.ID)),
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
			Body:      comment.Body,
			HTMLURL:   comment.URL,
			User: &types.SimpleUser{
				Login:     comment.Author.Login,
				AvatarURL: comment.Author.AvatarURL,
			},
		}
	}

	return timelineEvents, hasNextPage, endCursor, nil
}
