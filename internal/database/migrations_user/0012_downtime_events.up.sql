CREATE TABLE IF NOT EXISTS downtime_events (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    monitor_id  INTEGER  NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at  DATETIME NOT NULL,
    ended_at    DATETIME,        -- NULL = incident still open
    duration_s  INTEGER,         -- seconds, computed when closed
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_downtime_monitor_started
    ON downtime_events(monitor_id, started_at);

-- Composite index that makes LatestSince an efficient index-range scan.
CREATE INDEX IF NOT EXISTS idx_heartbeats_monitor_created
    ON heartbeats(monitor_id, created_at DESC);
