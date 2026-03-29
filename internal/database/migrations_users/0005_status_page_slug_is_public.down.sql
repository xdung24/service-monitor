-- SQLite <3.35 does not support DROP COLUMN; recreate without is_public.
CREATE TABLE status_page_slugs_bak AS SELECT slug, username, name FROM status_page_slugs;
DROP TABLE status_page_slugs;
ALTER TABLE status_page_slugs_bak RENAME TO status_page_slugs;
CREATE UNIQUE INDEX IF NOT EXISTS idx_status_page_slugs_slug ON status_page_slugs(slug);
