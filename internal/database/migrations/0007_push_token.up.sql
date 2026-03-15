-- 0007_push_token.up.sql
-- Adds a unique push token for push/heartbeat monitors.
ALTER TABLE monitors ADD COLUMN push_token TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_monitors_push_token ON monitors(push_token) WHERE push_token != '';
