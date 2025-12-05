-- Create shopping carts table
CREATE TABLE IF NOT EXISTS saas_carts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_phone TEXT NOT NULL,
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,

  -- Cart items (JSON array)
  items JSONB NOT NULL DEFAULT '[]',

  -- Total amount
  total_amount DECIMAL(12,2) DEFAULT 0,

  -- Cart status
  status TEXT DEFAULT 'active' CHECK (status IN ('active', 'checked_out', 'expired', 'cancelled')),

  -- Timestamps
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '24 hours',

  -- Constraints
  UNIQUE(customer_phone, client_id, status)
);

-- Indexes
CREATE INDEX idx_carts_customer_phone ON saas_carts(customer_phone);
CREATE INDEX idx_carts_client_id ON saas_carts(client_id);
CREATE INDEX idx_carts_status ON saas_carts(status);
CREATE INDEX idx_carts_expires_at ON saas_carts(expires_at);

-- Trigger for auto-update updated_at
CREATE TRIGGER update_carts_updated_at
    BEFORE UPDATE ON saas_carts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
