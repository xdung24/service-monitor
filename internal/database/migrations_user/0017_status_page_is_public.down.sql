-- SQLite does not support DROP COLUMN on older versions; recreate the table without is_public.
CREATE TABLE status_pages_bak AS SELECT id, name, slug, description, summary_uuid, created_at, updated_at FROM status_pages;
DROP TABLE status_pages;
ALTER TABLE status_pages_bak RENAME TO status_pages;
