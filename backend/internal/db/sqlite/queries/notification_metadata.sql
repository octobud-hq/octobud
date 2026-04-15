-- name: DeleteNotificationLabels :exec
DELETE FROM notification_labels WHERE notification_id = ?;

-- name: InsertNotificationLabel :exec
INSERT INTO notification_labels (notification_id, name, color) VALUES (?, ?, ?);

-- name: DeleteNotificationAssignees :exec
DELETE FROM notification_assignees WHERE notification_id = ?;

-- name: InsertNotificationAssignee :exec
INSERT INTO notification_assignees (notification_id, login, github_id, avatar_url) VALUES (?, ?, ?, ?);

-- name: DeleteNotificationReviewers :exec
DELETE FROM notification_reviewers WHERE notification_id = ?;

-- name: InsertNotificationReviewer :exec
INSERT INTO notification_reviewers (notification_id, login, github_id, status, avatar_url) VALUES (?, ?, ?, ?, ?);

-- name: DeleteNotificationTeamReviewers :exec
DELETE FROM notification_team_reviewers WHERE notification_id = ?;

-- name: InsertNotificationTeamReviewer :exec
INSERT INTO notification_team_reviewers (notification_id, slug, github_id) VALUES (?, ?, ?);

-- name: UpdateNotificationDraft :exec
UPDATE notifications SET subject_draft = ? WHERE id = ?;

-- name: GetNotificationLabels :many
SELECT name, color FROM notification_labels WHERE notification_id = ?;

-- name: GetNotificationAssignees :many
SELECT login, github_id, avatar_url FROM notification_assignees WHERE notification_id = ?;

-- name: GetNotificationReviewers :many
SELECT login, github_id, status, avatar_url FROM notification_reviewers WHERE notification_id = ?;

-- name: GetNotificationTeamReviewers :many
SELECT slug, github_id FROM notification_team_reviewers WHERE notification_id = ?;
