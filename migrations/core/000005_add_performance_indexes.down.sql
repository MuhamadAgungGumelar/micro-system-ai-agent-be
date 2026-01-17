-- Remove performance indexes

DROP INDEX IF EXISTS idx_company_users_phone;
DROP INDEX IF EXISTS idx_clients_whatsapp_status;
DROP INDEX IF EXISTS idx_clients_status;
DROP INDEX IF EXISTS idx_conversations_client_id;
DROP INDEX IF EXISTS idx_conversations_client_customer;
DROP INDEX IF EXISTS idx_transactions_client_id;
DROP INDEX IF EXISTS idx_products_client_id;
DROP INDEX IF EXISTS idx_cart_items_client_id;
