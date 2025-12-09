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

package notifications

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/octobud-hq/octobud/backend/internal/core/timeline"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

func TestFetchAndFilterTimelineEvents(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := timeline.NewService()

	tests := []struct {
		name          string
		timelinePages [][]types.TimelineEvent
		perPage       int
		page          int
		expectedCount int
		description   string
	}{
		{
			name: "filters out subscribed events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out 2 subscribed events and return 3",
		},
		{
			name: "filters out labeled events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out 2 labeled events and return 3",
		},
		{
			name: "filters out both subscribed and labeled events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out 2 subscribed and 2 labeled events and return 3",
		},
		{
			name: "filters out project events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "added_to_project_v2", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "project_v2_item_status_changed", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out project-related events",
		},
		{
			name: "filters out copilot events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "copilot_work_started", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "copilot_work_finished", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out copilot-related events",
		},
		{
			name: "filters out connected events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "connected", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "connected", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out connected events",
		},
		{
			name: "filters out review_requested events",
			timelinePages: [][]types.TimelineEvent{
				{
					{Event: "commented", CreatedAt: &now},
					{Event: "review_requested", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "review_requested", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
				},
			},
			perPage:       10,
			page:          1,
			expectedCount: 3, // Should only get commented, reviewed, committed
			description:   "should filter out review_requested events",
		},
		{
			name: "fetches multiple pages when needed",
			timelinePages: func() [][]types.TimelineEvent {
				// Page 1: mostly filtered events - only 1 valid event
				// Need 100 events so service continues to next page
				page1 := []types.TimelineEvent{
					{Event: "subscribed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
					{Event: "commented", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
				}
				// Pad with filtered events to reach 100
				for len(page1) < 100 {
					page1 = append(page1, types.TimelineEvent{Event: "subscribed", CreatedAt: &now})
				}

				// Page 2: more valid events - 3 more valid events
				page2 := []types.TimelineEvent{
					{Event: "reviewed", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
					{Event: "merged", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
				}
				// Pad with filtered events to reach 100
				for len(page2) < 100 {
					page2 = append(page2, types.TimelineEvent{Event: "subscribed", CreatedAt: &now})
				}

				return [][]types.TimelineEvent{page1, page2}
			}(),
			perPage:       3,
			page:          1,
			expectedCount: 3, // Should return first 3 valid events (perPage limit)
			description:   "should fetch multiple GitHub pages to satisfy the filtered result count",
		},
		{
			name: "handles pagination correctly",
			timelinePages: func() [][]types.TimelineEvent {
				// Need 100 events so service doesn't stop early
				page1 := []types.TimelineEvent{
					{Event: "commented", CreatedAt: &now},
					{Event: "reviewed", CreatedAt: &now},
					{Event: "subscribed", CreatedAt: &now},
					{Event: "committed", CreatedAt: &now},
					{Event: "labeled", CreatedAt: &now},
					{Event: "merged", CreatedAt: &now},
					{Event: "closed", CreatedAt: &now},
					{Event: "reopened", CreatedAt: &now},
				}
				// Pad with filtered events to reach 100
				for len(page1) < 100 {
					page1 = append(page1, types.TimelineEvent{Event: "subscribed", CreatedAt: &now})
				}
				return [][]types.TimelineEvent{page1}
			}(),
			perPage:       3,
			page:          2,
			expectedCount: 3, // Should return items 4-6 (page 2, perPage 3)
			description:   "should fetch enough events for the requested page and apply pagination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			client := githubmocks.NewMockClient(ctrl)
			subjectInfo := &types.SubjectInfo{
				Owner:  "test-owner",
				Repo:   "test-repo",
				Number: 123,
			}

			// Set up mock expectations for all pages we have data for
			for page := 1; page <= len(tt.timelinePages); page++ {
				events := tt.timelinePages[page-1]
				client.EXPECT().
					FetchTimeline(ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number, 100, page).
					Return(events, nil).
					AnyTimes()
			}

			// Set up expectations for pages beyond what we have data for
			for page := len(tt.timelinePages) + 1; page <= 10; page++ {
				client.EXPECT().
					FetchTimeline(ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number, 100, page).
					Return([]types.TimelineEvent{}, nil).
					AnyTimes()
			}

			// If no pages provided, service will try page 1 and get empty result
			if len(tt.timelinePages) == 0 {
				client.EXPECT().
					FetchTimeline(ctx, subjectInfo.Owner, subjectInfo.Repo, subjectInfo.Number, 100, 1).
					Return([]types.TimelineEvent{}, nil).
					Times(1)
			}

			result, err := service.FetchFilteredTimeline(
				ctx,
				client,
				subjectInfo,
				tt.perPage,
				tt.page,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Items) != tt.expectedCount {
				t.Errorf(
					"%s: expected %d events, got %d",
					tt.description,
					tt.expectedCount,
					len(result.Items),
				)
			}

			// Verify no filtered events are present
			for _, item := range result.Items {
				if timeline.FilteredEventTypes[item.Event] {
					t.Errorf("%s: found filtered event type '%s'", tt.description, item.Event)
				}
			}

			// Verify pagination metadata
			if result.Page != tt.page {
				t.Errorf("expected page %d, got %d", tt.page, result.Page)
			}
			if result.PerPage != tt.perPage {
				t.Errorf("expected perPage %d, got %d", tt.perPage, result.PerPage)
			}
		})
	}
}
