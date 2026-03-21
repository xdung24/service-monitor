-- Optional summary UUID for the public JSON API endpoint (/summary/:uuid).
-- Empty string means the feature is disabled for that status page.
ALTER TABLE status_pages ADD COLUMN summary_uuid TEXT NOT NULL DEFAULT '';
