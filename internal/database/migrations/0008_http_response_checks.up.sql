-- 0008_http_response_checks.up.sql
-- Adds response assertion options (header, body-type, JSON path, XPath) to monitors.
ALTER TABLE monitors ADD COLUMN http_header_name    TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_header_value   TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_body_type      TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_json_path      TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_json_expected  TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_xpath          TEXT NOT NULL DEFAULT '';
ALTER TABLE monitors ADD COLUMN http_xpath_expected TEXT NOT NULL DEFAULT '';
