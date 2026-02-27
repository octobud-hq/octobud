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
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/core/timeline"
	githubmocks "github.com/octobud-hq/octobud/backend/internal/github/mocks"
	"github.com/octobud-hq/octobud/backend/internal/github/types"
)

func TestFetchAndFilterTimelineEvents(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	service := timeline.NewService(zap.NewNop())

	subjectType := "issue"

	t.Run("returns all events from single page", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		client := githubmocks.NewMockClient(ctrl)
		subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

		events := []types.TimelineEvent{
			{Event: "commented", CreatedAt: &now},
			{Event: "reviewed", CreatedAt: &now},
			{Event: "committed", CreatedAt: &now},
		}
		// Single page: page 1 discovers lastPage=1 and returns data in one call
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
	})

	t.Run("fetches pages backwards when multi-page", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		client := githubmocks.NewMockClient(ctrl)
		subjectInfo := &types.SubjectInfo{Owner: "o", Repo: "r", Number: 1}

		page1Events := []types.TimelineEvent{
			{Event: "commented", CreatedAt: &now},
		}
		page2Events := []types.TimelineEvent{
			{Event: "committed", CreatedAt: &now},
			{Event: "merged", CreatedAt: &now},
		}

		// Page 1 probe returns lastPage=2
		call1 := client.EXPECT().
			FetchTimeline(ctx, "o", "r", 1, 30, 1).
			Return(nil, 2, nil).
			Times(1)
		// Fetch page 2 (newest)
		call2 := client.EXPECT().
			FetchTimeline(ctx, "o", "r", 1, 30, 2).
			Return(page2Events, 2, nil).
			Times(1).
			After(call1)
		// Fetch page 1 (oldest)
		client.EXPECT().
			FetchTimeline(ctx, "o", "r", 1, 30, 1).
			Return(page1Events, 2, nil).
			Times(1).
			After(call2)

		result, err := service.FetchFilteredTimeline(ctx, client, subjectInfo, subjectType, 10, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 3 {
			t.Errorf("expected 3 items, got %d", len(result.Items))
		}
	})

	t.Run("skips events without timestamps", func(t *testing.T) {
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
	})
}
