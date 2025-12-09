-- +goose Up
-- Add update_settings field to users table for auto-update configuration
ALTER TABLE users ADD COLUMN update_settings TEXT;

-- +goose Down
-- Remove update_settings field
ALTER TABLE users DROP COLUMN update_settings;

