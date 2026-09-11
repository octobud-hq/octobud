-- +goose Up
-- Serves the repository selector's grouped counts and the repository filter
-- (WHERE user_id = ? AND repository_id IN (...)) from one index.
CREATE INDEX IF NOT EXISTS idx_notifications_user_repository
    ON notifications(user_id, repository_id);

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_user_repository;
