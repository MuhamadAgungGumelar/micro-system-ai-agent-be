-- Create orders table
CREATE TABLE IF NOT EXISTS saas_orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  order_number TEXT UNIQUE NOT NULL,

  -- Customer Info
  customer_phone TEXT NOT NULL,
  customer_name TEXT,
  customer_email TEXT,

  -- Order Details
  items JSONB NOT NULL,
  total_amount DECIMAL(12,2) NOT NULL,
  currency TEXT DEFAULT 'IDR',

  -- Payment
  payment_method TEXT,
  payment_status TEXT DEFAULT 'pending',
  payment_gateway TEXT,
  payment_link TEXT,
  payment_token TEXT,
  payment_reference TEXT,
  paid_at TIMESTAMP,

  -- Fulfillment
  fulfillment_status TEXT DEFAULT 'pending',
  tracking_number TEXT,
  shipped_at TIMESTAMP,
  delivered_at TIMESTAMP,

  -- Shipping Address
  shipping_address TEXT,
  shipping_city TEXT,
  shipping_zip TEXT,

  -- Notes
  customer_notes TEXT,
  admin_notes TEXT,

  -- Metadata
  metadata JSONB,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Constraints
  CONSTRAINT valid_payment_status CHECK (payment_status IN ('pending', 'paid', 'failed', 'cancelled', 'refunded')),
  CONSTRAINT valid_fulfillment_status CHECK (fulfillment_status IN ('pending', 'processing', 'shipped', 'delivered', 'cancelled'))
);

-- Indexes for performance
CREATE INDEX idx_orders_client ON saas_orders(client_id);
CREATE INDEX idx_orders_customer_phone ON saas_orders(customer_phone);
CREATE INDEX idx_orders_order_number ON saas_orders(order_number);
CREATE INDEX idx_orders_payment_status ON saas_orders(payment_status);
CREATE INDEX idx_orders_fulfillment_status ON saas_orders(fulfillment_status);
CREATE INDEX idx_orders_created_at ON saas_orders(created_at DESC);

-- Trigger for auto-update updated_at
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON saas_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Note: update_updated_at_column() function should already exist from previous migrations
-- If not, create it:
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
