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

package sync

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/github"
)

// CurrentEnrichmentVersion is the current version of the enrichment pipeline.
// Bump this when adding new enrichments that require reprocessing existing notifications.
//
// Version history:
//
//	1 - Initial: extract metadata (labels, assignees, draft, team reviewers) from subject_raw,
//	    fetch PR reviews from API and merge with requested_reviewers.
const CurrentEnrichmentVersion int32 = 1

// EnrichNotificationBatch processes a batch of notifications that need enrichment.
// Returns the number of notifications successfully enriched.
func (s *Service) EnrichNotificationBatch(
	ctx context.Context,
	userID string,
	batchSize int32,
) (int64, error) {
	rows, err := s.userStore.ListNotificationsForEnrichment(
		ctx, userID, CurrentEnrichmentVersion, batchSize,
	)
	if err != nil {
		return 0, err
	}

	if len(rows) == 0 {
		return 0, nil
	}

	s.logger.Info("enrichment batch starting",
		zap.Int("count", len(rows)),
		zap.Int32("targetVersion", CurrentEnrichmentVersion),
	)

	var processed int64
	for _, row := range rows {
		if err := s.enrichNotification(ctx, row); err != nil {
			s.logger.Warn(
				"enrichment failed for notification, skipping",
				zap.String("githubID", row.GithubID),
				zap.Error(err),
			)
			// Stamp the version anyway to avoid retrying forever on permanently broken rows.
			// The next full sync will re-enrich with fresh data.
			if stampErr := s.userStore.UpdateNotificationEnrichmentVersion(
				ctx, row.ID, CurrentEnrichmentVersion,
			); stampErr != nil {
				s.logger.Warn("failed to stamp enrichment version after error",
					zap.String("githubID", row.GithubID),
					zap.Error(stampErr),
				)
			}
		} else {
			processed++
		}
	}

	return processed, nil
}

// enrichNotification runs all enrichment steps for a single notification
// and stamps it with the current enrichment version.
func (s *Service) enrichNotification(ctx context.Context, row db.EnrichmentRow) error {
	if !row.SubjectRaw.Valid || len(row.SubjectRaw.RawMessage) == 0 {
		// No subject data to enrich from - just stamp and move on
		return s.userStore.UpdateNotificationEnrichmentVersion(
			ctx, row.ID, CurrentEnrichmentVersion,
		)
	}

	subjectJSON := row.SubjectRaw.RawMessage

	// --- Version 1 enrichments ---

	// Extract and store metadata (labels, assignees, team reviewers)
	metadata := extractNotificationMetadata(subjectJSON)

	// For PRs: fetch reviews and merge with requested reviewers
	if strings.EqualFold(row.SubjectType, "PullRequest") {
		metadata.Reviewers = s.fetchAndMergeReviewers(ctx, row.SubjectURL, subjectJSON)
		s.logger.Debug("enriched PR reviewers",
			zap.String("githubID", row.GithubID),
			zap.Int("reviewerCount", len(metadata.Reviewers)),
		)
	}

	if err := s.userStore.ReplaceNotificationMetadata(ctx, row.ID, metadata); err != nil {
		return err
	}

	// Extract and store draft status
	draftVal := github.ExtractSubjectDraft(subjectJSON)
	if err := s.userStore.UpdateNotificationDraft(ctx, row.ID, draftVal.Bool); err != nil {
		return err
	}

	// Stamp with current version
	return s.userStore.UpdateNotificationEnrichmentVersion(
		ctx, row.ID, CurrentEnrichmentVersion,
	)
}
