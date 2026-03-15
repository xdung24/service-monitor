-- 0006_http_extended.up.sql
-- Adds HTTP-specific check options to the monitors table.
ALTER TABLE monitors ADD COLUMN http_accepted_statuses TEXT    NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_ignore_tls         INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN http_method             TEXT    NOT NULL DEFAULT 'GET';
ALTER TABLE monitors ADD COLUMN http_keyword            TEXT    NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_keyword_invert     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE monitors ADD COLUMN http_username           TEXT    NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_password           TEXT    NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_bearer_token       TEXT    NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_max_redirects      INTEGER NOT NULL DEFAULT 10;
