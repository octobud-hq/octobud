-- +goose Up
-- Add muted_until field to users table for temporary global mute functionality
ALTER TABLE users ADD COLUMN muted_until TEXT;

-- +goose Down
-- Remove muted_until field
ALTER TABLE users DROP COLUMN muted_until;

