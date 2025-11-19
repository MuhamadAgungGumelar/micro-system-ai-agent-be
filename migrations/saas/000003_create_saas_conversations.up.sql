-- Create saas_conversations table
CREATE TABLE IF NOT EXISTS saas_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES saas_clients(id) ON DELETE CASCADE,
    customer_phone TEXT NOT NULL,
    message_type TEXT DEFAULT 'incoming',
    message_text TEXT,
    ai_response TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_saas_conversations_client ON saas_conversations(client_id);
CREATE INDEX IF NOT EXISTS idx_saas_conversations_phone ON saas_conversations(customer_phone);
CREATE INDEX IF NOT EXISTS idx_saas_conversations_created ON saas_conversations(created_at DESC);
