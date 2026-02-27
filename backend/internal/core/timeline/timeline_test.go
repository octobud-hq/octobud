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

package timeline

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

func TestFetchFilteredTimeline_SinglePage(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	events := []types.TimelineEvent{
		{Event: "commented", CreatedAt: &now},
		{Event: "reviewed", CreatedAt: &now},
		{Event: "committed", CreatedAt: &now},
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
	if result.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestFetchFilteredTimeline_NewestFirst(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	t1 := now.Add(-2 * time.Hour)
	t2 := now.Add(-1 * time.Hour)
	t3 := now

	// REST returns oldest first: t1, t2, t3
	events := []types.TimelineEvent{
		{Event: "commented", CreatedAt: &t1, ID: json.RawMessage(`"oldest"`)},
		{Event: "reviewed", CreatedAt: &t2, ID: json.RawMessage(`"middle"`)},
		{Event: "committed", CreatedAt: &t3, ID: json.RawMessage(`"newest"`)},
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	// Should be reversed: newest first
	if result.Items[0].Event != "committed" {
		t.Errorf(
			"expected first item to be 'committed' (newest), got %q",
			result.Items[0].Event,
		)
	}
	if result.Items[2].Event != "commented" {
		t.Errorf("expected last item to be 'commented' (oldest), got %q", result.Items[2].Event)
	}
}

func TestFetchFilteredTimeline_StopsEarly(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	page3Events := make([]types.TimelineEvent, 5)
	for i := range page3Events {
		ts := now.Add(-time.Duration(i) * time.Minute)
		page3Events[i] = types.TimelineEvent{Event: "commented", CreatedAt: &ts}
	}

	// Page 1 probe returns lastPage=3
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(nil, 3, nil).
		Times(1)
	// Fetch page 3 (newest) — returns 5 items, enough for needed=2
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 3).
		Return(page3Events, 3, nil).
		Times(1)
	// Pages 2 and 1 should NOT be fetched

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if !result.HasMore {
		t.Error("expected hasMore=true (unfetched pages remain)")
	}
}

func TestFetchFilteredTimeline_FetchesMorePages(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	// Page 2 (newest page): all filtered events
	page2Events := []types.TimelineEvent{
		{Event: "subscribed", CreatedAt: &now},
		{Event: "referenced", CreatedAt: &now},
	}
	// Page 1 (oldest page): real events
	t1 := now.Add(-1 * time.Hour)
	page1Events := []types.TimelineEvent{
		{Event: "commented", CreatedAt: &t1},
		{Event: "reviewed", CreatedAt: &t1},
	}

	// Page 1 probe returns lastPage=2
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(nil, 2, nil).
		Times(1)
	// Fetch page 2 (newest) — all filtered, 0 valid items
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 2).
		Return(page2Events, 2, nil).
		Times(1)
	// Fetch page 1 — 2 valid items
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(page1Events, 2, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.HasMore {
		t.Error("expected hasMore=false (all pages fetched)")
	}
}

func TestFetchFilteredTimeline_EmptyTimeline(t *testing.T) {
	ctx := context.Background()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return([]types.TimelineEvent{}, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestFetchFilteredTimeline_SkipsNoTimestamp(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	events := []types.TimelineEvent{
		{Event: "commented", CreatedAt: &now},
		{Event: "labeled"}, // No timestamp
		{Event: "committed", CreatedAt: &now},
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestFetchFilteredTimeline_Pagination(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	events := make([]types.TimelineEvent, 6)
	for i := range events {
		ts := now.Add(time.Duration(i) * time.Minute)
		events[i] = types.TimelineEvent{
			Event:     "commented",
			CreatedAt: &ts,
			ID:        json.RawMessage(`"id-` + string(rune('a'+i)) + `"`),
		}
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 3, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items on page 2, got %d", len(result.Items))
	}
	if result.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Page)
	}
}

func TestFetchFilteredTimeline_FiltersNoisyEvents(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	subjectType := "issue"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	events := []types.TimelineEvent{
		{Event: "commented", CreatedAt: &now},
		{Event: "subscribed", CreatedAt: &now},
		{Event: "referenced", CreatedAt: &now},
		{Event: "committed", CreatedAt: &now},
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items (subscribed/referenced filtered), got %d", len(result.Items))
	}
}

func TestFetchFilteredTimeline_CrossReferenceSource(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	sourceJSON := json.RawMessage(`{
		"type": "issue",
		"issue": {
			"number": 456,
			"title": "Fix the bug",
			"html_url": "https://github.com/org/repo/pull/456",
			"state": "open"
		}
	}`)

	events := []types.TimelineEvent{
		{
			Event:     "cross-referenced",
			CreatedAt: &now,
			Actor:     &types.SimpleUser{Login: "user1"},
			SourceRaw: sourceJSON,
		},
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, "issue", 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.SourceType != "PullRequest" {
		t.Errorf("expected SourceType 'PullRequest', got %q", item.SourceType)
	}
	if item.SourceNumber != 456 {
		t.Errorf("expected SourceNumber 456, got %d", item.SourceNumber)
	}
	if item.SourceTitle != "Fix the bug" {
		t.Errorf("expected SourceTitle 'Fix the bug', got %q", item.SourceTitle)
	}
}

func TestFetchFilteredTimeline_ReviewEvent(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := NewService(zap.NewNop())
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := githubmocks.NewMockClient(ctrl)
	subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

	events := []types.TimelineEvent{
		{
			Event:       "reviewed",
			ID:          json.RawMessage(`12345`),
			SubmittedAt: &now,
			State:       "approved",
			User:        &types.SimpleUser{Login: "reviewer1"},
			Body:        "Looks good!",
		},
	}
	client.EXPECT().
		FetchTimeline(ctx, "o", "r", 1, 30, 1).
		Return(events, 1, nil).
		Times(1)

	result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, "pull_request", 10, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.State != "APPROVED" {
		t.Errorf("expected state 'APPROVED', got %q", item.State)
	}
	if item.AuthorLogin != "reviewer1" {
		t.Errorf("expected author 'reviewer1', got %q", item.AuthorLogin)
	}
}

func TestSupportsTimeline(t *testing.T) {
	tests := []struct {
		name        string
		subjectType string
		expected    bool
	}{
		{"pull request", "PullRequest", true},
		{"pull request lowercase", "pullrequest", true},
		{"pull request no underscore", "pull_request", true},
		{"issue", "Issue", true},
		{"issue lowercase", "issue", true},
		{"discussion", "Discussion", true},
		{"discussion lowercase", "discussion", true},
		{"unsupported type", "Commit", false},
		{"unsupported type", "Release", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SupportsTimeline(tt.subjectType)
			if result != tt.expected {
				t.Errorf(
					"SupportsTimeline(%q) = %v, expected %v",
					tt.subjectType,
					result,
					tt.expected,
				)
			}
		})
	}
}
