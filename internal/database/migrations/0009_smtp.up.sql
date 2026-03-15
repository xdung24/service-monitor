-- 0009_smtp.up.sql
-- Adds SMTP monitor type fields.
ALTER TABLE monitors ADD COLUMN smtp_use_tls    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN smtp_ignore_tls INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN smtp_username   TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN smtp_password   TEXT NOT NULL DEFAULT '';
