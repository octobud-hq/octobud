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

//go:build integration

// Package client provides a typed HTTP client for integration tests.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// Client is a typed HTTP client for the Octobud API.
// No authentication needed - trusts localhost.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New creates a new test client for the given base URL.
func New(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Notification represents a notification in API responses.
type Notification struct {
	ID                int64      `json:"id"`
	GithubID          string     `json:"githubId"`
	RepositoryID      int64      `json:"repositoryId"`
	SubjectType       string     `json:"subjectType"`
	SubjectTitle      string     `json:"subjectTitle"`
	Reason            *string    `json:"reason,omitempty"`
	Archived          bool       `json:"archived"`
	IsRead            bool       `json:"isRead"`
	Muted             bool       `json:"muted"`
	Starred           bool       `json:"starred"`
	Filtered          bool       `json:"filtered"`
	SnoozedUntil      *time.Time `json:"snoozedUntil,omitempty"`
	SnoozedAt         *time.Time `json:"snoozedAt,omitempty"`
	EffectiveSortDate time.Time  `json:"effectiveSortDate"`
	GithubUpdatedAt   *time.Time `json:"githubUpdatedAt,omitempty"`
	ImportedAt        time.Time  `json:"importedAt"`
}

// ListNotificationsResponse represents the response from listing notifications.
type ListNotificationsResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int64          `json:"total"`
	Page          int            `json:"page"`
	PageSize      int            `json:"pageSize"`
}

// NotificationResponse represents a single notification response.
type NotificationResponse struct {
	Notification Notification `json:"notification"`
}

// BulkResponse represents the response from bulk operations.
type BulkResponse struct {
	Count int `json:"count"`
}

// ListNotifications retrieves a list of notifications.
func (c *Client) ListNotifications(t *testing.T, query string, page, pageSize int) *ListNotificationsResponse {
	t.Helper()

	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(pageSize))
	}

	path := "/api/notifications"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(t, "GET", path, nil)
	if err != nil {
		t.Fatalf("ListNotifications request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("ListNotifications failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result ListNotificationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode ListNotifications response: %v", err)
	}

	return &result
}

// GetNotification retrieves a single notification by GitHub ID.
func (c *Client) GetNotification(t *testing.T, githubID string) *NotificationResponse {
	t.Helper()

	resp, err := c.doRequest(t, "GET", "/api/notifications/"+url.PathEscape(githubID), nil)
	if err != nil {
		t.Fatalf("GetNotification request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GetNotification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode GetNotification response: %v", err)
	}

	return &result
}

// SnoozeNotification snoozes a notification until the specified time.
func (c *Client) SnoozeNotification(t *testing.T, githubID string, until time.Time) *NotificationResponse {
	t.Helper()

	body := map[string]interface{}{
		"snoozedUntil": until.Format(time.RFC3339),
	}

	resp, err := c.doRequest(t, "POST", "/api/notifications/"+url.PathEscape(githubID)+"/snooze", body)
	if err != nil {
		t.Fatalf("SnoozeNotification request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("SnoozeNotification failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode SnoozeNotification response: %v", err)
	}

	return &result
}

// UnsnoozeNotification unsnoozes a notification.
func (c *Client) UnsnoozeNotification(t *testing.T, githubID string) *NotificationResponse {
	t.Helper()

	resp, err := c.doRequest(t, "POST", "/api/notifications/"+url.PathEscape(githubID)+"/unsnooze", nil)
	if err != nil {
		t.Fatalf("UnsnoozeNotification request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("UnsnoozeNotification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode UnsnoozeNotification response: %v", err)
	}

	return &result
}

// ArchiveNotification archives a notification.
func (c *Client) ArchiveNotification(t *testing.T, githubID string) *NotificationResponse {
	t.Helper()

	resp, err := c.doRequest(t, "POST", "/api/notifications/"+url.PathEscape(githubID)+"/archive", nil)
	if err != nil {
		t.Fatalf("ArchiveNotification request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("ArchiveNotification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode ArchiveNotification response: %v", err)
	}

	return &result
}

// UnarchiveNotification unarchives a notification.
func (c *Client) UnarchiveNotification(t *testing.T, githubID string) *NotificationResponse {
	t.Helper()

	resp, err := c.doRequest(t, "POST", "/api/notifications/"+url.PathEscape(githubID)+"/unarchive", nil)
	if err != nil {
		t.Fatalf("UnarchiveNotification request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("UnarchiveNotification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode UnarchiveNotification response: %v", err)
	}

	return &result
}

// MuteNotification mutes a notification.
func (c *Client) MuteNotification(t *testing.T, githubID string) *NotificationResponse {
	t.Helper()

	resp, err := c.doRequest(t, "POST", "/api/notifications/"+url.PathEscape(githubID)+"/mute", nil)
	if err != nil {
		t.Fatalf("MuteNotification request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("MuteNotification failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result NotificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode MuteNotification response: %v", err)
	}

	return &result
}

// BulkSnoozeRequest represents a bulk snooze request.
type BulkSnoozeRequest struct {
	GithubIDs    []string  `json:"github_ids,omitempty"`
	Query        string    `json:"query,omitempty"`
	SnoozedUntil time.Time `json:"snoozed_until"`
}

// BulkSnooze snoozes multiple notifications.
func (c *Client) BulkSnooze(t *testing.T, req BulkSnoozeRequest) *BulkResponse {
	t.Helper()

	body := map[string]interface{}{
		"snoozedUntil": req.SnoozedUntil.Format(time.RFC3339),
	}
	if len(req.GithubIDs) > 0 {
		body["githubIDs"] = req.GithubIDs
	}
	if req.Query != "" {
		body["query"] = req.Query
	}

	resp, err := c.doRequest(t, "POST", "/api/notifications/bulk/snooze", body)
	if err != nil {
		t.Fatalf("BulkSnooze request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("BulkSnooze failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result BulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode BulkSnooze response: %v", err)
	}

	return &result
}

// BulkArchive archives multiple notifications.
func (c *Client) BulkArchive(t *testing.T, githubIDs []string, query string) *BulkResponse {
	t.Helper()

	body := make(map[string]interface{})
	if len(githubIDs) > 0 {
		body["githubIDs"] = githubIDs
	}
	if query != "" {
		body["query"] = query
	}

	resp, err := c.doRequest(t, "POST", "/api/notifications/bulk/archive", body)
	if err != nil {
		t.Fatalf("BulkArchive request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("BulkArchive failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result BulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode BulkArchive response: %v", err)
	}

	return &result
}

// BulkUnsnooze unsnoozes multiple notifications.
func (c *Client) BulkUnsnooze(t *testing.T, githubIDs []string, query string) *BulkResponse {
	t.Helper()

	body := make(map[string]interface{})
	if len(githubIDs) > 0 {
		body["githubIDs"] = githubIDs
	}
	if query != "" {
		body["query"] = query
	}

	resp, err := c.doRequest(t, "POST", "/api/notifications/bulk/unsnooze", body)
	if err != nil {
		t.Fatalf("BulkUnsnooze request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("BulkUnsnooze failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result BulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode BulkUnsnooze response: %v", err)
	}

	return &result
}

// doRequest performs an HTTP request.
// No authentication needed - trusts localhost.
func (c *Client) doRequest(t *testing.T, method, path string, body interface{}) (*http.Response, error) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	fullURL := c.BaseURL + path
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)
}
