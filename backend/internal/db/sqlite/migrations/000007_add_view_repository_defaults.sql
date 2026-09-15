-- +goose Up
-- Per-view default repository selection for the repository selector.
-- view_key is the custom view's id, a system view slug (inbox, archive, ...),
-- or "tag-<tag id>" for tag views. repository_ids is a JSON array of repository ids.
CREATE TABLE view_repository_defaults (
    user_id TEXT NOT NULL,
    view_key TEXT NOT NULL,
    repository_ids TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, view_key)
);

-- +goose Down
DROP TABLE IF EXISTS view_repository_defaults;
