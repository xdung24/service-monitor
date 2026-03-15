-- Add db_query column for database monitor types (mysql, postgres, redis, mongodb).
-- The connection string / DSN is stored in the existing `url` column.
ALTER TABLE monitors ADD COLUMN db_query TEXT NOT NULL DEFAULT '';
