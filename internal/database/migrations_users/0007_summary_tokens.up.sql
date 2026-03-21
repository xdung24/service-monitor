-- Maps every status-page summary UUID to its owning user so the unauthenticated
-- /summary/:uuid endpoint can locate the correct per-user database.
CREATE TABLE IF NOT EXISTS summary_tokens (
    uuid     TEXT NOT NULL PRIMARY KEY,
    username TEXT NOT NULL
);
