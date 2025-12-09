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
	defaultPerPage = 50
	githubAPIBase  = "https://api.github.com"
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
		_ = resp.Body.Close() // Error can be ignored in defer
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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
		_ = resp.Body.Close() // Body already read, safe to ignore error

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
		_ = resp.Body.Close() // Error can be ignored in defer
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
		_ = resp.Body.Close() // Error can be ignored in defer
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
		_ = resp.Body.Close() // Error can be ignored in defer
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
		_ = resp.Body.Close() // Error can be ignored in defer
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
