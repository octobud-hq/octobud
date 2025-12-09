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

package view

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/octobud-hq/octobud/backend/internal/db"
	"github.com/octobud-hq/octobud/backend/internal/models"
	"github.com/octobud-hq/octobud/backend/internal/query"
)

// Error definitions
var (
	ErrFailedToGetView                  = errors.New("failed to get view")
	ErrFailedToLoadViews                = errors.New("failed to load views")
	ErrFailedToCalculateViewCounts      = errors.New("failed to calculate view counts")
	ErrFailedToCalculateInboxCount      = errors.New("failed to calculate inbox count")
	ErrFailedToCalculateEverythingCount = errors.New("failed to calculate everything count")
	ErrFailedToCalculateArchiveCount    = errors.New("failed to calculate archive count")
	ErrFailedToCalculateSnoozedCount    = errors.New("failed to calculate snoozed count")
	ErrFailedToCalculateStarredCount    = errors.New("failed to calculate starred count")
	ErrFailedToCreateView               = errors.New("failed to create view")
	ErrFailedToUpdateView               = errors.New("failed to update view")
	ErrFailedToDeleteView               = errors.New("failed to delete view")
	ErrFailedToReorderViews             = errors.New("failed to reorder views")
	ErrViewNotFound                     = errors.New("view not found")
	ErrInvalidQuery                     = errors.New("invalid query")
	ErrViewNameAlreadyExists            = errors.New("a view with that name already exists")
	ErrFailedToCheckLinkedRules         = errors.New("failed to check for linked rules")
	ErrFailedToValidateViews            = errors.New("failed to validate views")
	ErrFailedToLoadUpdatedViews         = errors.New("failed to load updated views")
	// Validation errors
	ErrNameRequired                = errors.New("name is required")
	ErrNameCannotBeEmpty           = errors.New("name cannot be empty")
	ErrNameMustContainAlphanumeric = errors.New(
		"name must contain at least one alphanumeric character",
	)
	ErrSlugReserved            = errors.New("slug is reserved and cannot be used")
	ErrQueryRequired           = errors.New("query is required")
	ErrQueryCannotBeEmpty      = errors.New("query cannot be empty")
	ErrCannotReorderSystemView = errors.New("cannot reorder system view")
)

// GetView returns a view by ID
func (s *Service) GetView(ctx context.Context, userID string, viewID string) (db.View, error) {
	view, err := s.queries.GetView(ctx, userID, viewID)
	if err != nil {
		return db.View{}, errors.Join(ErrFailedToGetView, err)
	}
	return view, nil
}

// ListViewsWithCounts returns all views (custom + system) with their unread counts
func (s *Service) ListViewsWithCounts(ctx context.Context, userID string) ([]models.View, error) {
	views, err := s.queries.ListViews(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadViews, err)
	}

	// Build response for custom views with counts
	response := make([]models.View, 0, len(views)+6)
	for _, view := range views {
		// Calculate "new" count for this view
		unreadCount, countErr := s.calculateViewUnreadCount(ctx, userID, view.Query)
		if countErr != nil {
			return nil, errors.Join(ErrFailedToCalculateViewCounts, countErr)
		}

		viewResp := models.ViewFromDB(view)
		viewResp.UnreadCount = unreadCount
		response = append(response, viewResp)
	}

	// Inject system views: inbox, everything, snoozed, archive, starred
	inboxCount, err := s.calculateInboxUnreadCount(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToCalculateInboxCount, err)
	}

	response = append(response, models.View{
		ID:          "inbox",
		Name:        "Inbox",
		Slug:        "inbox",
		Icon:        models.StrPtr("inbox"),
		SystemView:  true,
		Query:       "in:inbox",
		UnreadCount: inboxCount,
	})

	everythingCount, err := s.calculateEverythingUnreadCount(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToCalculateEverythingCount, err)
	}

	response = append(response, models.View{
		ID:          "everything",
		Name:        "Everything",
		Slug:        "everything",
		Icon:        models.StrPtr("infinity"),
		SystemView:  true,
		Query:       "in:anywhere",
		UnreadCount: everythingCount,
	})

	archiveCount, err := s.calculateArchiveUnreadCount(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToCalculateArchiveCount, err)
	}

	response = append(response, models.View{
		ID:          "archive",
		Name:        "Archive",
		Slug:        "archive",
		Icon:        models.StrPtr("archive"),
		SystemView:  true,
		Query:       "in:archive",
		UnreadCount: archiveCount,
	})

	snoozedCount, err := s.calculateSnoozedUnreadCount(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToCalculateSnoozedCount, err)
	}

	response = append(response, models.View{
		ID:          "snoozed",
		Name:        "Snoozed",
		Slug:        "snoozed",
		Icon:        models.StrPtr("clock"),
		SystemView:  true,
		Query:       "in:snoozed",
		UnreadCount: snoozedCount,
	})

	starredCount, err := s.calculateStarredUnreadCount(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToCalculateStarredCount, err)
	}

	response = append(response, models.View{
		ID:          "starred",
		Name:        "Starred",
		Slug:        "starred",
		Icon:        models.StrPtr("star"),
		SystemView:  true,
		Query:       "is:starred",
		UnreadCount: starredCount,
	})

	return response, nil
}

// CreateView creates a new view
func (s *Service) CreateView(
	ctx context.Context,
	userID, name string,
	description, icon *string,
	isDefault *bool,
	queryStr string,
) (models.View, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.View{}, ErrNameRequired
	}

	slug := models.Slugify(name)
	if slug == "" {
		return models.View{}, ErrNameMustContainAlphanumeric
	}
	if _, isReserved := reservedSlugs[slug]; isReserved {
		return models.View{}, fmt.Errorf(
			"slug '%s' is reserved and cannot be used: %w",
			slug,
			ErrSlugReserved,
		)
	}

	queryStr = strings.TrimSpace(queryStr)
	if queryStr == "" {
		return models.View{}, ErrQueryRequired
	}

	// Validate the query by attempting to parse and validate it
	if _, err := query.ParseAndValidate(queryStr); err != nil {
		return models.View{}, errors.Join(ErrInvalidQuery, err)
	}

	params := db.CreateViewParams{
		Name:        name,
		Slug:        slug,
		Description: models.StringPtrToNull(description),
		Icon:        models.StringPtrToNull(icon),
		Query:       models.StringPtrToNull(&queryStr),
	}
	if isDefault != nil {
		params.IsDefault = *isDefault
	}

	view, err := s.queries.CreateView(ctx, userID, params)
	if err != nil {
		if models.IsUniqueViolation(err) {
			return models.View{}, errors.Join(ErrViewNameAlreadyExists, err)
		}
		return models.View{}, errors.Join(ErrFailedToCreateView, err)
	}

	// Calculate initial count for the new view
	unreadCount, err := s.calculateViewUnreadCount(ctx, userID, view.Query)
	if err != nil {
		// Don't fail creation if count calculation fails
		unreadCount = 0
	}

	resp := models.ViewFromDB(view)
	resp.UnreadCount = unreadCount
	return resp, nil
}

// UpdateView updates an existing view
func (s *Service) UpdateView(
	ctx context.Context,
	userID string,
	viewID string,
	name, description, icon *string,
	isDefault *bool,
	queryStr *string,
) (models.View, error) {
	params := db.UpdateViewParams{
		ID: viewID,
	}

	if name != nil {
		nameTrimmed := strings.TrimSpace(*name)
		if nameTrimmed == "" {
			return models.View{}, ErrNameCannotBeEmpty
		}
		params.Name = sql.NullString{String: nameTrimmed, Valid: true}

		slug := models.Slugify(nameTrimmed)
		if slug == "" {
			return models.View{}, ErrNameMustContainAlphanumeric
		}
		if _, isReserved := reservedSlugs[slug]; isReserved {
			return models.View{}, fmt.Errorf(
				"slug '%s' is reserved and cannot be used: %w",
				slug,
				ErrSlugReserved,
			)
		}
		params.Slug = sql.NullString{String: slug, Valid: true}
	}
	if description != nil {
		params.Description = sql.NullString{String: strings.TrimSpace(*description), Valid: true}
	}
	if icon != nil {
		iconTrimmed := strings.TrimSpace(*icon)
		if iconTrimmed == "" {
			params.Icon = sql.NullString{Valid: false}
		} else {
			params.Icon = sql.NullString{String: iconTrimmed, Valid: true}
		}
	}
	if isDefault != nil {
		params.IsDefault = sql.NullBool{Bool: *isDefault, Valid: true}
	}
	if queryStr != nil {
		queryTrimmed := strings.TrimSpace(*queryStr)
		if queryTrimmed == "" {
			return models.View{}, ErrQueryCannotBeEmpty
		}
		// Validate the query by attempting to parse and validate it
		if _, err := query.ParseAndValidate(queryTrimmed); err != nil {
			return models.View{}, errors.Join(ErrInvalidQuery, err)
		}
		params.Query = sql.NullString{String: queryTrimmed, Valid: true}
	}

	view, err := s.queries.UpdateView(ctx, userID, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.View{}, errors.Join(ErrViewNotFound, err)
		}
		if models.IsUniqueViolation(err) {
			return models.View{}, errors.Join(ErrViewNameAlreadyExists, err)
		}
		return models.View{}, errors.Join(ErrFailedToUpdateView, err)
	}

	// Calculate count for the updated view
	unreadCount, err := s.calculateViewUnreadCount(ctx, userID, view.Query)
	if err != nil {
		// Don't fail update if count calculation fails
		unreadCount = 0
	}

	resp := models.ViewFromDB(view)
	resp.UnreadCount = unreadCount
	return resp, nil
}

// DeleteView deletes a view, returning the count of linked rules if any
// If force is true, the view will be deleted even if it has linked rules (which will be cascade deleted)
func (s *Service) DeleteView(
	ctx context.Context,
	userID string,
	viewID string,
	force bool,
) (linkedRuleCount int, err error) {
	// Check if there are any rules linked to this view
	linkedRules, err := s.queries.GetRulesByViewID(
		ctx,
		userID,
		sql.NullString{String: viewID, Valid: true},
	)
	if err != nil {
		return 0, errors.Join(ErrFailedToCheckLinkedRules, err)
	}

	linkedRuleCount = len(linkedRules)

	// If there are linked rules and force is not set, return the count (caller should handle the conflict)
	if linkedRuleCount > 0 && !force {
		return linkedRuleCount, nil
	}

	if _, err := s.queries.DeleteView(ctx, userID, viewID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.Join(ErrViewNotFound, err)
		}
		return 0, errors.Join(ErrFailedToDeleteView, err)
	}

	return linkedRuleCount, nil
}

// ReorderViews updates the display order of views
func (s *Service) ReorderViews(
	ctx context.Context,
	userID string,
	viewIDs []string,
) ([]models.View, error) {
	if len(viewIDs) == 0 {
		return nil, fmt.Errorf("viewIDs cannot be empty")
	}

	// Validate all view IDs exist and are not system views
	for _, id := range viewIDs {
		view, err := s.queries.GetView(ctx, userID, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.Join(ErrViewNotFound, err)
			}
			return nil, errors.Join(ErrFailedToValidateViews, err)
		}

		// Check if it's a system view (system views have reserved slugs)
		if _, isReserved := reservedSlugs[view.Slug]; isReserved {
			return nil, fmt.Errorf(
				"cannot reorder system view: %s: %w",
				view.Slug,
				ErrCannotReorderSystemView,
			)
		}
	}

	// Update display order for each view
	// Use increments of 100 to allow for future insertions
	for i, viewID := range viewIDs {
		displayOrder := int32((i + 1) * 100)
		err := s.queries.UpdateViewOrder(ctx, userID, db.UpdateViewOrderParams{
			ID:           viewID,
			DisplayOrder: displayOrder,
		})
		if err != nil {
			return nil, errors.Join(ErrFailedToReorderViews, err)
		}
	}

	// Return the updated views list
	views, err := s.queries.ListViews(ctx, userID)
	if err != nil {
		return nil, errors.Join(ErrFailedToLoadUpdatedViews, err)
	}

	viewResponses := make([]models.View, len(views))
	for i, view := range views {
		viewResponses[i] = models.ViewFromDB(view)
	}

	return viewResponses, nil
}
