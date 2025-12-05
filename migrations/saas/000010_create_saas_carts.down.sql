-- Drop carts table
DROP TRIGGER IF EXISTS update_carts_updated_at ON saas_carts;
DROP INDEX IF EXISTS idx_carts_expires_at;
DROP INDEX IF EXISTS idx_carts_status;
DROP INDEX IF EXISTS idx_carts_client_id;
DROP INDEX IF EXISTS idx_carts_customer_phone;
DROP TABLE IF EXISTS saas_carts CASCADE;
