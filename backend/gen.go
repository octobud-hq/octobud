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

// Package backend contains the code for the backend.
package backend

//go:generate mockgen -destination=internal/github/mocks/mock_client.go -package=mocks github.com/octobud-hq/octobud/backend/internal/github/interfaces Client
//go:generate mockgen -source=internal/db/store.go -destination=internal/db/mocks/mock_store.go -package=mocks
//go:generate mockgen -source=internal/core/notification/service.go -destination=internal/core/notification/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/rules/service.go -destination=internal/core/rules/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/view/service.go -destination=internal/core/view/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/tag/service.go -destination=internal/core/tag/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/timeline/timeline.go -destination=internal/core/timeline/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/syncstate/service.go -destination=internal/core/syncstate/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/auth/service.go -destination=internal/core/auth/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/repository/service.go -destination=internal/core/repository/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/core/pullrequest/service.go -destination=internal/core/pullrequest/mocks/mock_service.go -package=mocks
//go:generate mockgen -source=internal/jobs/scheduler.go -destination=internal/jobs/mocks/mock_scheduler.go -package=mocks
//go:generate mockgen -source=internal/jobs/handlers/rule_matcher.go -destination=internal/jobs/mocks/mock_rule_matcher.go -package=mocks
//go:generate mockgen -destination=internal/sync/mocks/mock_sync.go -package=syncmocks github.com/octobud-hq/octobud/backend/internal/sync SyncOperations
