-- Drop trigger
DROP TRIGGER IF EXISTS update_orders_updated_at ON saas_orders;

-- Drop indexes
DROP INDEX IF EXISTS idx_orders_client;
DROP INDEX IF EXISTS idx_orders_customer_phone;
DROP INDEX IF EXISTS idx_orders_order_number;
DROP INDEX IF EXISTS idx_orders_payment_status;
DROP INDEX IF EXISTS idx_orders_fulfillment_status;
DROP INDEX IF EXISTS idx_orders_created_at;

-- Drop table
DROP TABLE IF EXISTS saas_orders CASCADE;
