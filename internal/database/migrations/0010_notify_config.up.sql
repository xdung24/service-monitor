-- 0010_notify_config.up.sql
-- Add per-monitor notification trigger settings and optional response-body excerpt.
ALTER TABLE monitors ADD COLUMN notify_on_failure INTEGER NOT NULL DEFAULT 1;
ALTER TABLE monitors ADD COLUMN notify_on_success INTEGER NOT NULL DEFAULT 1;
ALTER TABLE monitors ADD COLUMN notify_body_chars  INTEGER NOT NULL DEFAULT 0;
