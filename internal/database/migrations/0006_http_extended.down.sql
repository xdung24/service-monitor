-- 0006_http_extended.down.sql
-- Rebuild monitors table without the http_* columns added in 0006.
CREATE TABLE monitors_new AS SELECT
    id, name, type, url, interval_seconds, timeout_seconds, active, retries,
    dns_server, dns_record_type, dns_expected,
    last_status, last_notified_status,
    created_at, updated_at
FROM monitors;
DROP TABLE monitors;
ALTER TABLE monitors_new RENAME TO monitors;
