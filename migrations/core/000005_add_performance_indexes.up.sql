-- Add performance indexes for frequently queried columns
-- These indexes significantly improve message resolution and tenant lookup performance

-- Index for phone number lookup in company_users (used in tenant resolution)
CREATE INDEX IF NOT EXISTS idx_company_users_phone ON company_users(phone_number);

-- Composite index for WhatsApp number + subscription status (used in message routing)
CREATE INDEX IF NOT EXISTS idx_clients_whatsapp_status ON clients(whatsapp_number, subscription_status);

-- Index for subscription status filtering
CREATE INDEX IF NOT EXISTS idx_clients_status ON clients(subscription_status);

-- Index for client ID lookups in conversations (frequently queried)
CREATE INDEX IF NOT EXISTS idx_conversations_client_id ON conversations(client_id);

-- Composite index for customer conversations lookup
CREATE INDEX IF NOT EXISTS idx_conversations_client_customer ON conversations(client_id, customer_phone);

-- Index for transaction lookups by client
CREATE INDEX IF NOT EXISTS idx_transactions_client_id ON transactions(client_id);

-- Index for product lookups by client
CREATE INDEX IF NOT EXISTS idx_products_client_id ON products(client_id);

-- Index for cart items by client
CREATE INDEX IF NOT EXISTS idx_cart_items_client_id ON cart_items(client_id);

COMMENT ON INDEX idx_company_users_phone IS 'Improves tenant resolution from phone number';
COMMENT ON INDEX idx_clients_whatsapp_status IS 'Optimizes message routing queries';
COMMENT ON INDEX idx_clients_status IS 'Fast subscription status filtering';
