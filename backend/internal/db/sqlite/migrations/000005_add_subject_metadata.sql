-- +goose Up
-- Scalar: draft status on notifications table
ALTER TABLE notifications ADD COLUMN subject_draft INTEGER;

-- Enrichment version for background re-enrichment pipeline
ALTER TABLE notifications ADD COLUMN enrichment_version INTEGER NOT NULL DEFAULT 0;

-- Junction table: labels from GitHub subject
CREATE TABLE notification_labels (
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT
);
CREATE INDEX idx_notification_labels_notif ON notification_labels(notification_id);
CREATE INDEX idx_notification_labels_name ON notification_labels(name);

-- Junction table: assignees from GitHub subject
CREATE TABLE notification_assignees (
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    login TEXT NOT NULL,
    github_id INTEGER,
    avatar_url TEXT
);
CREATE INDEX idx_notification_assignees_notif ON notification_assignees(notification_id);
CREATE INDEX idx_notification_assignees_login ON notification_assignees(login);

-- Junction table: reviewers (individual users) from GitHub PR subject
CREATE TABLE notification_reviewers (
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    login TEXT NOT NULL,
    github_id INTEGER,
    status TEXT NOT NULL DEFAULT 'pending',
    avatar_url TEXT
);
CREATE INDEX idx_notification_reviewers_notif ON notification_reviewers(notification_id);
CREATE INDEX idx_notification_reviewers_login ON notification_reviewers(login);

-- Junction table: team reviewers from GitHub PR subject
CREATE TABLE notification_team_reviewers (
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    github_id INTEGER
);
CREATE INDEX idx_notification_team_reviewers_notif ON notification_team_reviewers(notification_id);
CREATE INDEX idx_notification_team_reviewers_slug ON notification_team_reviewers(slug);

-- +goose Down
DROP INDEX IF EXISTS idx_notification_team_reviewers_slug;
DROP INDEX IF EXISTS idx_notification_team_reviewers_notif;
DROP TABLE IF EXISTS notification_team_reviewers;

DROP INDEX IF EXISTS idx_notification_reviewers_login;
DROP INDEX IF EXISTS idx_notification_reviewers_notif;
DROP TABLE IF EXISTS notification_reviewers;

DROP INDEX IF EXISTS idx_notification_assignees_login;
DROP INDEX IF EXISTS idx_notification_assignees_notif;
DROP TABLE IF EXISTS notification_assignees;

DROP INDEX IF EXISTS idx_notification_labels_name;
DROP INDEX IF EXISTS idx_notification_labels_notif;
DROP TABLE IF EXISTS notification_labels;
