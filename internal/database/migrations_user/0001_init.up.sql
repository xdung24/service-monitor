-- 0001_init.up.sql (per-user data DB)
-- Consolidated schema equivalent to running migrations 0001-0011 on a fresh DB,
-- but without the users table (which lives in the shared users.db).

CREATE TABLE IF NOT EXISTS monitors (
    id                    INTEGER  PRIMARY KEY AUTOINCREMENT,
    name                  TEXT     NOT NULL,
    type                  TEXT     NOT NULL DEFAULT 'http',
    url                   TEXT     NOT NULL DEFAULT '',
    interval_seconds      INTEGER  NOT NULL DEFAULT 60,
    timeout_seconds       INTEGER  NOT NULL DEFAULT 30,
    active                INTEGER  NOT NULL DEFAULT 1,
    retries               INTEGER  NOT NULL DEFAULT 1,
    dns_server            TEXT     NOT NULL DEFAULT '',
    dns_record_type       TEXT     NOT NULL DEFAULT 'A',
    dns_expected          TEXT     NOT NULL DEFAULT '',
    http_accepted_statuses TEXT    NOT NULL DEFAULT '',
    http_ignore_tls       INTEGER  NOT NULL DEFAULT 0,
    http_method           TEXT     NOT NULL DEFAULT 'GET',
    http_keyword          TEXT     NOT NULL DEFAULT '',
    http_keyword_invert   INTEGER  NOT NULL DEFAULT 0,
    http_username         TEXT     NOT NULL DEFAULT '',
    http_password         TEXT     NOT NULL DEFAULT '',
    http_bearer_token     TEXT     NOT NULL DEFAULT '',
    http_max_redirects    INTEGER  NOT NULL DEFAULT 10,
    push_token            TEXT     NOT NULL DEFAULT '',
    http_header_name      TEXT     NOT NULL DEFAULT '',
    http_header_value     TEXT     NOT NULL DEFAULT '',
    http_body_type        TEXT     NOT NULL DEFAULT '',
    http_json_path        TEXT     NOT NULL DEFAULT '',
    http_json_expected    TEXT     NOT NULL DEFAULT '',
    http_xpath            TEXT     NOT NULL DEFAULT '',
    http_xpath_expected   TEXT     NOT NULL DEFAULT '',
    smtp_use_tls          INTEGER  NOT NULL DEFAULT 0,
    smtp_ignore_tls       INTEGER  NOT NULL DEFAULT 0,
    smtp_username         TEXT     NOT NULL DEFAULT '',
    smtp_password         TEXT     NOT NULL DEFAULT '',
    notify_on_failure     INTEGER  NOT NULL DEFAULT 1,
    notify_on_success     INTEGER  NOT NULL DEFAULT 1,
    notify_body_chars     INTEGER  NOT NULL DEFAULT 0,
    http_request_headers  TEXT     NOT NULL DEFAULT '',
    http_request_body     TEXT     NOT NULL DEFAULT '',
    last_status           INTEGER,
    last_notified_status  INTEGER,
    created_at            DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monitors_push_token ON monitors(push_token) WHERE push_token != '';

CREATE TABLE IF NOT EXISTS heartbeats (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER  NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status     INTEGER  NOT NULL DEFAULT 0,
    latency_ms INTEGER  NOT NULL DEFAULT 0,
    message    TEXT     NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_heartbeats_monitor_id ON heartbeats(monitor_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_created_at ON heartbeats(created_at);

CREATE TABLE IF NOT EXISTS notifications (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    type       TEXT     NOT NULL,
    config     TEXT     NOT NULL DEFAULT '{}',
    active     INTEGER  NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS monitor_notifications (
    monitor_id      INTEGER NOT NULL REFERENCES monitors(id)      ON DELETE CASCADE,
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    PRIMARY KEY (monitor_id, notification_id)
);

CREATE TABLE IF NOT EXISTS notification_logs (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    monitor_id        INTEGER  REFERENCES monitors(id)      ON DELETE SET NULL,
    notification_id   INTEGER  REFERENCES notifications(id) ON DELETE SET NULL,
    monitor_name      TEXT     NOT NULL DEFAULT '',
    notification_name TEXT     NOT NULL DEFAULT '',
    event_status      INTEGER  NOT NULL DEFAULT 0,
    success           INTEGER  NOT NULL DEFAULT 1,
    error             TEXT     NOT NULL DEFAULT '',
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_created_at     ON notification_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_notification_logs_monitor_id     ON notification_logs(monitor_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_notification_id ON notification_logs(notification_id);
