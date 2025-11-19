-- Create saas_clients table
CREATE TABLE IF NOT EXISTS saas_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    whatsapp_number TEXT NOT NULL,
    business_name TEXT NOT NULL,
    subscription_plan TEXT DEFAULT 'free',
    subscription_status TEXT DEFAULT 'active',
    tone TEXT DEFAULT 'neutral',
    wa_device_id TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_saas_clients_whatsapp ON saas_clients(whatsapp_number);
CREATE INDEX IF NOT EXISTS idx_saas_clients_status ON saas_clients(subscription_status);
