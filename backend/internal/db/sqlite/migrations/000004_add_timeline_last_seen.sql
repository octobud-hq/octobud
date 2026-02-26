-- +goose Up
ALTER TABLE notifications ADD COLUMN timeline_last_seen_at TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly, so this is a no-op for down migration
-- In practice, the column will remain but be unused
