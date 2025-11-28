-- Create universal clients table (shared across all modules)
CREATE TABLE IF NOT EXISTS clients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Business info
  business_name TEXT NOT NULL,
  module TEXT NOT NULL DEFAULT 'saas'
    CHECK (module IN ('saas', 'umkm', 'farmasi', 'manufacturing')),

  -- WhatsApp configuration
  whatsapp_number TEXT,           -- Business WA number for customer chat
  whatsapp_session_id TEXT,       -- WAHA/provider session ID
  wa_device_id TEXT,

  -- Subscription
  subscription_status TEXT DEFAULT 'active'
    CHECK (subscription_status IN ('active', 'inactive', 'trial', 'suspended')),
  subscription_plan TEXT DEFAULT 'free'
    CHECK (subscription_plan IN ('free', 'basic', 'pro', 'enterprise')),

  -- Settings
  tone TEXT DEFAULT 'neutral',
  timezone TEXT DEFAULT 'Asia/Jakarta',

  -- Timestamps
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Create company_users table (users/employees of businesses)
CREATE TABLE IF NOT EXISTS company_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,

  -- User info
  phone_number TEXT NOT NULL,     -- Personal WA number
  email TEXT,
  name TEXT NOT NULL,

  -- Access control
  role TEXT NOT NULL DEFAULT 'staff'
    CHECK (role IN ('admin', 'manager', 'staff', 'viewer')),

  -- Status
  status TEXT DEFAULT 'active'
    CHECK (status IN ('active', 'inactive', 'pending')),
  invited_by UUID REFERENCES company_users(id),

  -- Timestamps
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  -- Constraints
  UNIQUE(client_id, phone_number)  -- One phone per company
);

-- Create indexes for performance
CREATE INDEX idx_clients_module_status ON clients(module, subscription_status);
CREATE INDEX idx_clients_whatsapp_number ON clients(whatsapp_number) WHERE whatsapp_number IS NOT NULL;
CREATE INDEX idx_clients_session_id ON clients(whatsapp_session_id) WHERE whatsapp_session_id IS NOT NULL;

CREATE INDEX idx_company_users_phone ON company_users(phone_number);
CREATE INDEX idx_company_users_client ON company_users(client_id);
CREATE INDEX idx_company_users_client_role ON company_users(client_id, role);

-- Migrate existing data from saas_clients
INSERT INTO clients (
  id,
  business_name,
  whatsapp_number,
  whatsapp_session_id,
  subscription_status,
  subscription_plan,
  tone,
  wa_device_id,
  module,
  created_at,
  updated_at
)
SELECT
  id,
  business_name,
  whatsapp_number,
  whatsapp_session_id,
  subscription_status,
  subscription_plan,
  tone,
  wa_device_id,
  'saas' as module,  -- Set module for existing data
  created_at,
  updated_at
FROM saas_clients
ON CONFLICT (id) DO NOTHING;

-- Update foreign keys for existing tables to reference new clients table
ALTER TABLE saas_transactions
  DROP CONSTRAINT IF EXISTS saas_transactions_client_id_fkey,
  ADD CONSTRAINT saas_transactions_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

ALTER TABLE saas_conversations
  DROP CONSTRAINT IF EXISTS saas_conversations_client_id_fkey,
  ADD CONSTRAINT saas_conversations_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

ALTER TABLE saas_knowledge_base
  DROP CONSTRAINT IF EXISTS saas_knowledge_base_client_id_fkey,
  ADD CONSTRAINT saas_knowledge_base_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

ALTER TABLE saas_workflows
  DROP CONSTRAINT IF EXISTS saas_workflows_client_id_fkey,
  ADD CONSTRAINT saas_workflows_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

-- Create function to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for auto-update
CREATE TRIGGER update_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_company_users_updated_at
    BEFORE UPDATE ON company_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
