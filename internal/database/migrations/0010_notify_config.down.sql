-- 0010_notify_config.down.sql
-- Rebuild monitors table without the notify_* columns added in 0010.
CREATE TABLE monitors_new AS SELECT
    id, name, type, url, interval_seconds, timeout_seconds, active, retries,
    dns_server, dns_record_type, dns_expected,
    http_accepted_statuses, http_ignore_tls, http_method, http_keyword, http_keyword_invert,
    http_username, http_password, http_bearer_token, http_max_redirects,
    push_token,
    http_header_name, http_header_value, http_body_type,
    http_json_path, http_json_expected, http_xpath, http_xpath_expected,
    smtp_use_tls, smtp_ignore_tls, smtp_username, smtp_password,
    last_status, last_notified_status,
    created_at, updated_at
FROM monitors;
DROP TABLE monitors;
ALTER TABLE monitors_new RENAME TO monitors;
