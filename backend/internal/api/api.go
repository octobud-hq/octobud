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

// Package api provides the API handling logic for the service.
package api //nolint:revive // This is a package.

import (
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/octobud-hq/octobud/backend/internal/api/navigation"
	"github.com/octobud-hq/octobud/backend/internal/api/notifications"
	"github.com/octobud-hq/octobud/backend/internal/api/oauth"
	"github.com/octobud-hq/octobud/backend/internal/api/repositories"
	"github.com/octobud-hq/octobud/backend/internal/api/rules"
	"github.com/octobud-hq/octobud/backend/internal/api/tags"
	apiuser "github.com/octobud-hq/octobud/backend/internal/api/user"
	"github.com/octobud-hq/octobud/backend/internal/api/views"
	config "github.com/octobud-hq/octobud/backend/internal/config"
	authsvc "github.com/octobud-hq/octobud/backend/internal/core/auth"
	"github.com/octobud-hq/octobud/backend/internal/core/notification"
	"github.com/octobud-hq/octobud/backend/internal/core/pullrequest"
	"github.com/octobud-hq/octobud/backend/internal/core/repository"
	rulescore "github.com/octobud-hq/octobud/backend/internal/core/rules"
	"github.com/octobud-hq/octobud/backend/internal/core/syncstate"
	"github.com/octobud-hq/octobud/backend/internal/core/tag"
	timelinesvc "github.com/octobud-hq/octobud/backend/internal/core/timeline"
	"github.com/octobud-hq/octobud/backend/internal/core/update"
	"github.com/octobud-hq/octobud/backend/internal/core/view"
	"github.com/octobud-hq/octobud/backend/internal/db"
	githubinterfaces "github.com/octobud-hq/octobud/backend/internal/github/interfaces"
	"github.com/octobud-hq/octobud/backend/internal/jobs"
	"github.com/octobud-hq/octobud/backend/internal/osactions"
	"github.com/octobud-hq/octobud/backend/internal/sync"
)

// Handler wires HTTP routes to database-backed operations.
type Handler struct {
	logger         *zap.Logger
	store          db.Store
	notifications  *notification.Service
	syncService    sync.SyncOperations
	githubClient   githubinterfaces.Client
	timelineSvc    *timelinesvc.Service
	scheduler      jobs.Scheduler
	notificationsH *notifications.Handler
	tagsH          *tags.Handler
	viewsH         *views.Handler
	rulesH         *rules.Handler
	repositoriesH  *repositories.Handler
	userH          *apiuser.Handler
	oauthH         *oauth.Handler

	tokenManager          apiuser.TokenManagerInterface
	navigationBroadcaster *navigation.Broadcaster
}

// HandlerOption configures a Handler
type HandlerOption func(*Handler)

// WithSyncService configures the handler with a sync service for refreshing subject data.
// This enables the refresh-subject endpoint for notifications.
func WithSyncService(
	store db.Store,
	githubClient githubinterfaces.Client,
	logger *zap.Logger,
) HandlerOption {
	return func(h *Handler) {
		syncStateSvc := syncstate.NewSyncStateService(store)
		repositorySvc := repository.NewService(store)
		pullRequestSvc := pullrequest.NewService(store)
		notificationSvc := notification.NewService(store)
		h.syncService = sync.NewService(
			logger,
			time.Now,
			githubClient,
			syncStateSvc,
			repositorySvc,
			pullRequestSvc,
			notificationSvc,
			store, // userStore for sync settings
		)
		h.githubClient = githubClient
		h.timelineSvc = timelinesvc.NewService(logger)
	}
}

// WithScheduler configures the handler with a scheduler for queuing jobs.
func WithScheduler(scheduler jobs.Scheduler) HandlerOption {
	return func(h *Handler) {
		h.scheduler = scheduler
	}
}

// WithTokenManager configures the handler with a GitHub token manager.
// This enables GitHub token management endpoints.
func WithTokenManager(tokenManager apiuser.TokenManagerInterface) HandlerOption {
	return func(h *Handler) {
		h.tokenManager = tokenManager
	}
}

// WithNavigationBroadcaster configures the handler with a navigation event broadcaster.
// This enables navigation event broadcasting for tray menu actions.
func WithNavigationBroadcaster(broadcaster *navigation.Broadcaster) HandlerOption {
	return func(h *Handler) {
		h.navigationBroadcaster = broadcaster
	}
}

// NewHandler returns an API handler backed by the provided db store.
func NewHandler(store db.Store, opts ...HandlerOption) *Handler {
	// Initialize zap logger with human-readable console format
	logger := config.NewConsoleLogger()

	// Create all business logic services
	notificationsSvc := notification.NewService(store)
	repositorySvc := repository.NewService(store)
	tagSvc := tag.NewService(store)
	viewSvc := view.NewService(store)
	ruleSvc := rulescore.NewService(store)
	syncStateSvc := syncstate.NewSyncStateService(store)
	authService := authsvc.NewService(store)

	h := &Handler{
		logger:        logger,
		store:         store,
		notifications: notificationsSvc,
	}

	// Apply options
	for _, opt := range opts {
		opt(h)
	}

	// Create all resource handlers
	h.notificationsH = notifications.New(
		logger, store, notificationsSvc, repositorySvc, tagSvc,
		h.timelineSvc, h.githubClient, h.syncService, h.scheduler, authService,
	)
	h.tagsH = tags.New(logger, tagSvc, authService)
	h.viewsH = views.New(logger, viewSvc, authService)
	h.rulesH = rules.NewWithScheduler(logger, ruleSvc, viewSvc, h.scheduler, authService)
	h.repositoriesH = repositories.New(logger, repositorySvc, authService)

	// Create user handler
	h.userH = apiuser.New(logger, authService)

	// Wire up scheduler and sync state
	if h.scheduler != nil {
		h.userH = h.userH.WithScheduler(h.scheduler)
	}
	h.userH = h.userH.WithSyncStateService(syncStateSvc)
	h.userH = h.userH.WithStore(store)

	// Wire up token manager
	if h.tokenManager != nil {
		h.userH = h.userH.WithTokenManager(h.tokenManager)
		// Create OAuth handler with token manager
		h.oauthH = oauth.New(logger, h.tokenManager)
	}

	// Wire up update service
	updateService := update.NewService(logger)
	h.userH = h.userH.WithUpdateService(updateService)

	// Wire up OS actions service (for restart and browser tab activation)
	osActionsSvc := osactions.NewService()
	h.userH = h.userH.WithOSActionsService(osActionsSvc)

	return h
}

// Register attaches all API routes to the provided router.
// Note: The router passed in should already be scoped to /api (e.g., via router.Route("/api", ...))
func (h *Handler) Register(r chi.Router) {
	h.notificationsH.Register(r)
	h.tagsH.Register(r)
	h.viewsH.Register(r)
	h.rulesH.Register(r)
	h.repositoriesH.Register(r)
}

// RegisterAllRoutes registers all API routes.
func (h *Handler) RegisterAllRoutes(r chi.Router) {
	// User routes
	if h.userH != nil {
		h.userH.Register(r)
	}

	// OAuth routes
	if h.oauthH != nil {
		h.oauthH.Register(r)
	}

	// Navigation routes (SSE for tray menu navigation)
	if h.navigationBroadcaster != nil {
		navHandler := navigation.NewHandler(h.logger, h.navigationBroadcaster)
		navHandler.Register(r)
	}

	// All other API routes
	h.Register(r)
}

// GetNavigationBroadcaster returns the navigation broadcaster if configured.
func (h *Handler) GetNavigationBroadcaster() *navigation.Broadcaster {
	return h.navigationBroadcaster
}
