-- +goose Up
-- SQLite Schema for Octobud
-- This schema uses:
-- - INTEGER for booleans (0/1)
-- - TEXT for timestamps (ISO 8601 format)
-- - TEXT for JSON
-- - TEXT UUIDs for user-created objects (views, tags, rules)
-- - user_id (GitHub user ID) for multi-user/sync support

-- Users table: single user with GitHub identity
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    github_user_id TEXT,
    github_username TEXT,
    github_token_encrypted TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    sync_settings TEXT,
    retention_settings TEXT
);

-- Repositories: GitHub repository data, scoped by user
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    github_id INTEGER,
    node_id TEXT,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    owner_login TEXT,
    owner_id INTEGER,
    private INTEGER,
    description TEXT,
    html_url TEXT,
    fork INTEGER,
    visibility TEXT,
    default_branch TEXT,
    archived INTEGER,
    disabled INTEGER,
    pushed_at TEXT,
    created_at TEXT,
    updated_at TEXT,
    raw TEXT,
    owner_avatar_url TEXT,
    owner_html_url TEXT,
    UNIQUE(user_id, full_name)
);

-- Pull requests: GitHub PR data, scoped by user
CREATE TABLE IF NOT EXISTS pull_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    repository_id INTEGER NOT NULL REFERENCES repositories(id),
    github_id INTEGER,
    node_id TEXT,
    number INTEGER NOT NULL,
    title TEXT,
    state TEXT,
    draft INTEGER,
    merged INTEGER,
    author_login TEXT,
    author_id INTEGER,
    created_at TEXT,
    updated_at TEXT,
    closed_at TEXT,
    merged_at TEXT,
    raw TEXT,
    UNIQUE(user_id, repository_id, number)
);

-- Notifications: GitHub notification data with triage state, scoped by user
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    github_id TEXT NOT NULL,
    repository_id INTEGER NOT NULL REFERENCES repositories(id),
    pull_request_id INTEGER REFERENCES pull_requests(id),
    subject_type TEXT NOT NULL,
    subject_title TEXT NOT NULL,
    subject_url TEXT,
    subject_latest_comment_url TEXT,
    reason TEXT,
    archived INTEGER NOT NULL DEFAULT 0,
    github_unread INTEGER,
    github_updated_at TEXT,
    github_last_read_at TEXT,
    github_url TEXT,
    github_subscription_url TEXT,
    imported_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    payload TEXT,
    subject_raw TEXT,
    subject_fetched_at TEXT,
    author_login TEXT,
    author_id INTEGER,
    is_read INTEGER NOT NULL DEFAULT 0,
    muted INTEGER NOT NULL DEFAULT 0,
    snoozed_until TEXT,
    effective_sort_date TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    snoozed_at TEXT,
    starred INTEGER NOT NULL DEFAULT 0,
    filtered INTEGER NOT NULL DEFAULT 0,
    subject_number INTEGER,
    subject_state TEXT,
    subject_merged INTEGER,
    subject_state_reason TEXT,
    UNIQUE(user_id, github_id)
);

-- Tags: user-created labels with UUID primary key
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))),
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    display_order INTEGER NOT NULL DEFAULT 0,
    slug TEXT NOT NULL,
    UNIQUE(user_id, name),
    UNIQUE(user_id, slug)
);

-- Tag assignments: links tags to entities (notifications)
CREATE TABLE IF NOT EXISTS tag_assignments (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))),
    user_id TEXT NOT NULL,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, tag_id, entity_type, entity_id)
);

-- Views: user-created saved queries with UUID primary key
CREATE TABLE IF NOT EXISTS views (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))),
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    icon TEXT,
    slug TEXT NOT NULL,
    query TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE(user_id, name),
    UNIQUE(user_id, slug)
);

-- Sync state: per-user sync progress tracking
CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,
    last_successful_poll TEXT,
    last_notification_etag TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    latest_notification_at TEXT,
    initial_sync_completed_at TEXT,
    oldest_notification_synced_at TEXT
);

-- Rules: user-created automation rules with UUID primary key
CREATE TABLE IF NOT EXISTS rules (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6)))),
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    query TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    actions TEXT NOT NULL DEFAULT '{}',
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    view_id TEXT REFERENCES views(id) ON DELETE SET NULL,
    UNIQUE(user_id, name)
);

-- Jobs: internal background job queue (no user scoping)
CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    queue TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    scheduled_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    started_at TEXT,
    completed_at TEXT,
    last_error TEXT
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_repositories_user_id ON repositories(user_id);
CREATE INDEX IF NOT EXISTS idx_pull_requests_user_id ON pull_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_github_id ON notifications(user_id, github_id);
CREATE INDEX IF NOT EXISTS idx_notifications_repository_id ON notifications(repository_id);
CREATE INDEX IF NOT EXISTS idx_notifications_effective_sort_date ON notifications(user_id, effective_sort_date);
CREATE INDEX IF NOT EXISTS idx_tags_user_id ON tags(user_id);
CREATE INDEX IF NOT EXISTS idx_tag_assignments_user_id ON tag_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_tag_assignments_entity ON tag_assignments(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_views_user_id ON views(user_id);
CREATE INDEX IF NOT EXISTS idx_rules_user_id ON rules(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_poll ON jobs(queue, status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_jobs_stale ON jobs(status, started_at) WHERE status = 'processing';

-- +goose Down
DROP INDEX IF EXISTS idx_jobs_stale;
DROP INDEX IF EXISTS idx_jobs_poll;
DROP INDEX IF EXISTS idx_rules_user_id;
DROP INDEX IF EXISTS idx_views_user_id;
DROP INDEX IF EXISTS idx_tag_assignments_entity;
DROP INDEX IF EXISTS idx_tag_assignments_user_id;
DROP INDEX IF EXISTS idx_tags_user_id;
DROP INDEX IF EXISTS idx_notifications_effective_sort_date;
DROP INDEX IF EXISTS idx_notifications_repository_id;
DROP INDEX IF EXISTS idx_notifications_github_id;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_pull_requests_user_id;
DROP INDEX IF EXISTS idx_repositories_user_id;

DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS rules;
DROP TABLE IF EXISTS sync_state;
DROP TABLE IF EXISTS views;
DROP TABLE IF EXISTS tag_assignments;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS users;
