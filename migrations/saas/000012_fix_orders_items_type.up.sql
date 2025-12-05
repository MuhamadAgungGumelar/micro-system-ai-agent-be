-- Fix items column type from JSON to JSONB
ALTER TABLE saas_orders ALTER COLUMN items TYPE JSONB USING items::jsonb;
