-- Remove deleted_at column
DROP INDEX IF EXISTS idx_carts_deleted_at;
ALTER TABLE saas_carts DROP COLUMN IF EXISTS deleted_at;
