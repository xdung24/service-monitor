-- 0008_http_response_checks.down.sql
-- Rebuild monitors table without the response-assertion columns added in 0008.
CREATE TABLE monitors_new AS SELECT
    id, name, type, url, interval_seconds, timeout_seconds, active, retries,
    dns_server, dns_record_type, dns_expected,
    http_accepted_statuses, http_ignore_tls, http_method, http_keyword, http_keyword_invert,
    http_username, http_password, http_bearer_token, http_max_redirects,
    push_token,
    last_status, last_notified_status,
    created_at, updated_at
FROM monitors;
DROP TABLE monitors;
ALTER TABLE monitors_new RENAME TO monitors;
