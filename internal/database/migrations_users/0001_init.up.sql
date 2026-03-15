-- 0001_init.up.sql (users DB)
-- Shared database: stores user accounts and a global push-token → username index.

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT    NOT NULL UNIQUE,
    password   TEXT    NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Maps every push-monitor token to its owning user so the unauthenticated
-- /push/:token endpoint can locate the correct per-user database.
CREATE TABLE IF NOT EXISTS push_tokens (
    token    TEXT NOT NULL PRIMARY KEY,
    username TEXT NOT NULL
);
