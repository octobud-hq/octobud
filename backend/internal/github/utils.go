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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

// ExtractAuthorFromSubject extracts author login and ID from subject JSON.
// Works for PRs, Issues, and other GitHub entities that have a "user" or "sender" field.
func ExtractAuthorFromSubject(subjectJSON json.RawMessage) (sql.NullString, sql.NullInt64) {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullString{}, sql.NullInt64{}
	}

	// Try "user" field first (PRs, Issues, etc.)
	if user, ok := data["user"].(map[string]interface{}); ok {
		return extractUserFields(user)
	}

	// Try "sender" field (some notification types)
	if sender, ok := data["sender"].(map[string]interface{}); ok {
		return extractUserFields(sender)
	}

	return sql.NullString{}, sql.NullInt64{}
}

// extractUserFields extracts login and ID from a user/sender object
func extractUserFields(user map[string]interface{}) (sql.NullString, sql.NullInt64) {
	var login sql.NullString
	var id sql.NullInt64

	if loginVal, ok := user["login"].(string); ok && loginVal != "" {
		login = sql.NullString{String: loginVal, Valid: true}
	}

	if idVal, ok := user["id"].(float64); ok {
		id = sql.NullInt64{Int64: int64(idVal), Valid: true}
	}

	return login, id
}

// ExtractSubjectNumber extracts the subject number from subject JSON.
// Works for PRs and Issues which have a "number" field.
func ExtractSubjectNumber(subjectJSON json.RawMessage) sql.NullInt32 {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullInt32{}
	}

	// Try "number" field (PRs, Issues, etc.)
	if numberVal, ok := data["number"].(float64); ok {
		// Convert to int32 (SQLite INTEGER maps to int32)
		// GitHub PR/issue numbers are well within int32 range
		return sql.NullInt32{Int32: int32(numberVal), Valid: true}
	}

	return sql.NullInt32{}
}

// ExtractSubjectState extracts the subject state from subject JSON.
// Works for PRs and Issues which have a "state" field (e.g., "open", "closed").
func ExtractSubjectState(subjectJSON json.RawMessage) sql.NullString {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullString{}
	}

	// Try "state" field (PRs, Issues, etc.)
	if stateVal, ok := data["state"].(string); ok && stateVal != "" {
		return sql.NullString{String: stateVal, Valid: true}
	}

	return sql.NullString{}
}

// ExtractSubjectMerged extracts the merged status from subject JSON.
// Works for Pull Requests which have a "merged" field (boolean).
func ExtractSubjectMerged(subjectJSON json.RawMessage) sql.NullBool {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullBool{}
	}

	// Try "merged" field (PRs only)
	if mergedVal, ok := data["merged"].(bool); ok {
		return sql.NullBool{Bool: mergedVal, Valid: true}
	}

	return sql.NullBool{}
}

// ExtractSubjectTitle extracts the title from subject JSON.
// Works for PRs, Issues, and Discussions which all have a "title" field.
func ExtractSubjectTitle(subjectJSON json.RawMessage) sql.NullString {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullString{}
	}

	if titleVal, ok := data["title"].(string); ok && titleVal != "" {
		return sql.NullString{String: titleVal, Valid: true}
	}

	return sql.NullString{}
}

// ExtractSubjectStateReason extracts the state_reason from subject JSON.
// Works for Issues which have a "state_reason" field (e.g., "completed", "not_planned").
func ExtractSubjectStateReason(subjectJSON json.RawMessage) sql.NullString {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullString{}
	}

	// Try "state_reason" field (Issues only)
	if reasonVal, ok := data["state_reason"].(string); ok && reasonVal != "" {
		return sql.NullString{String: reasonVal, Valid: true}
	}

	return sql.NullString{}
}

// PullRequestData represents extracted pull request data from GitHub API responses.
// This is a pure data structure with no database dependencies.
type PullRequestData struct {
	GithubID    *int64
	NodeID      *string
	Number      int
	Title       *string
	State       *string
	Draft       *bool
	Merged      *bool
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	ClosedAt    *time.Time
	MergedAt    *time.Time
	AuthorLogin *string
	AuthorID    *int64
}

// ExtractPullRequestData parses PR data from subject JSON.
// This is a pure extraction function that returns structured data with no database dependencies.
func ExtractPullRequestData(subjectJSON json.RawMessage) (*PullRequestData, error) {
	var prData struct {
		ID        *int64  `json:"id"`
		NodeID    *string `json:"node_id"`
		Number    *int    `json:"number"`
		Title     *string `json:"title"`
		State     *string `json:"state"`
		Draft     *bool   `json:"draft"`
		Merged    *bool   `json:"merged"`
		CreatedAt *string `json:"created_at"`
		UpdatedAt *string `json:"updated_at"`
		ClosedAt  *string `json:"closed_at"`
		MergedAt  *string `json:"merged_at"`
		User      *struct {
			Login *string `json:"login"`
			ID    *int64  `json:"id"`
		} `json:"user"`
	}

	if err := json.Unmarshal(subjectJSON, &prData); err != nil {
		return nil, fmt.Errorf("unmarshal PR subject: %w", err)
	}

	// Require at least number field
	if prData.Number == nil {
		return nil, errors.New("PR subject missing number field")
	}

	result := &PullRequestData{
		GithubID: prData.ID,
		NodeID:   prData.NodeID,
		Number:   *prData.Number,
		Title:    prData.Title,
		State:    prData.State,
		Draft:    prData.Draft,
		Merged:   prData.Merged,
	}

	// Parse timestamps
	if prData.CreatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *prData.CreatedAt); err == nil {
			result.CreatedAt = &t
		}
	}
	if prData.UpdatedAt != nil {
		if t, err := time.Parse(time.RFC3339, *prData.UpdatedAt); err == nil {
			result.UpdatedAt = &t
		}
	}
	if prData.ClosedAt != nil {
		if t, err := time.Parse(time.RFC3339, *prData.ClosedAt); err == nil {
			result.ClosedAt = &t
		}
	}
	if prData.MergedAt != nil {
		if t, err := time.Parse(time.RFC3339, *prData.MergedAt); err == nil {
			result.MergedAt = &t
		}
	}

	// Extract author info
	if prData.User != nil {
		result.AuthorLogin = prData.User.Login
		result.AuthorID = prData.User.ID
	}

	return result, nil
}

// LabelData represents a GitHub label extracted from subject JSON.
type LabelData struct {
	Name  string
	Color string // hex color without '#' prefix
}

// UserData represents a GitHub user extracted from subject JSON.
type UserData struct {
	Login     string
	GithubID  int64
	AvatarURL string
}

// TeamData represents a GitHub team extracted from subject JSON.
type TeamData struct {
	Slug     string
	GithubID int64
}

// ExtractSubjectDraft extracts the draft status from subject JSON.
// Works for Pull Requests which have a "draft" field (boolean).
func ExtractSubjectDraft(subjectJSON json.RawMessage) sql.NullBool {
	var data map[string]interface{}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return sql.NullBool{}
	}

	if draftVal, ok := data["draft"].(bool); ok {
		return sql.NullBool{Bool: draftVal, Valid: true}
	}

	return sql.NullBool{}
}

// ExtractSubjectLabels extracts the labels array from subject JSON.
// Works for PRs and Issues which have a "labels" field.
func ExtractSubjectLabels(subjectJSON json.RawMessage) []LabelData {
	var data struct {
		Labels []struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return nil
	}

	labels := make([]LabelData, 0, len(data.Labels))
	for _, l := range data.Labels {
		if l.Name != "" {
			labels = append(labels, LabelData{Name: l.Name, Color: l.Color})
		}
	}
	return labels
}

// ExtractSubjectAssignees extracts the assignees array from subject JSON.
// Works for PRs and Issues which have an "assignees" field.
func ExtractSubjectAssignees(subjectJSON json.RawMessage) []UserData {
	var data struct {
		Assignees []struct {
			Login     string `json:"login"`
			ID        int64  `json:"id"`
			AvatarURL string `json:"avatar_url"`
		} `json:"assignees"`
	}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return nil
	}

	users := make([]UserData, 0, len(data.Assignees))
	for _, a := range data.Assignees {
		if a.Login != "" {
			users = append(users, UserData{Login: a.Login, GithubID: a.ID, AvatarURL: a.AvatarURL})
		}
	}
	return users
}

// ExtractSubjectReviewers extracts the requested_reviewers array from subject JSON.
// Works for Pull Requests which have a "requested_reviewers" field.
// Note: GitHub clears this array after a review is submitted, so this reflects
// currently pending review requests only.
func ExtractSubjectReviewers(subjectJSON json.RawMessage) []UserData {
	var data struct {
		RequestedReviewers []struct {
			Login     string `json:"login"`
			ID        int64  `json:"id"`
			AvatarURL string `json:"avatar_url"`
		} `json:"requested_reviewers"`
	}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return nil
	}

	users := make([]UserData, 0, len(data.RequestedReviewers))
	for _, r := range data.RequestedReviewers {
		if r.Login != "" {
			users = append(users, UserData{Login: r.Login, GithubID: r.ID, AvatarURL: r.AvatarURL})
		}
	}
	return users
}

// ExtractSubjectTeamReviewers extracts the requested_teams array from subject JSON.
// Works for Pull Requests which have a "requested_teams" field.
func ExtractSubjectTeamReviewers(subjectJSON json.RawMessage) []TeamData {
	var data struct {
		RequestedTeams []struct {
			Slug string `json:"slug"`
			ID   int64  `json:"id"`
		} `json:"requested_teams"`
	}
	if err := json.Unmarshal(subjectJSON, &data); err != nil {
		return nil
	}

	teams := make([]TeamData, 0, len(data.RequestedTeams))
	for _, t := range data.RequestedTeams {
		if t.Slug != "" {
			teams = append(teams, TeamData{Slug: t.Slug, GithubID: t.ID})
		}
	}
	return teams
}

// ExtractSubjectInfo parses a subject URL or raw JSON to extract repository owner, repo, and number.
// This is used to make GitHub API calls for the subject (e.g., fetching timeline).
// Note: Subject type should come from the notification's SubjectType field, not from URL parsing.
func ExtractSubjectInfo(subjectURL string, subjectRaw json.RawMessage) (*types.SubjectInfo, error) {
	// Try to extract from subject URL first
	// Format: https://api.github.com/repos/{owner}/{repo}/{type}/{number}
	if subjectURL != "" {
		parts := strings.Split(strings.TrimPrefix(subjectURL, "https://api.github.com/repos/"), "/")
		if len(parts) >= 4 {
			var number int
			_, err := fmt.Sscanf(parts[3], "%d", &number)
			if err == nil {
				return &types.SubjectInfo{
					Owner:  parts[0],
					Repo:   parts[1],
					Number: number,
				}, nil
			}
		}
	}

	// Try to extract from subject_raw JSON
	if len(subjectRaw) > 0 {
		var raw map[string]interface{}
		if err := json.Unmarshal(subjectRaw, &raw); err == nil {
			if num, ok := raw["number"].(float64); ok {
				if htmlURL, ok := raw["html_url"].(string); ok {
					// Parse HTML URL: https://github.com/{owner}/{repo}/{type}/{number}
					parts := strings.Split(strings.TrimPrefix(htmlURL, "https://github.com/"), "/")
					if len(parts) >= 4 {
						return &types.SubjectInfo{
							Owner:  parts[0],
							Repo:   parts[1],
							Number: int(num),
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("could not extract subject info from URL or raw JSON")
}
