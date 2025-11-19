-- Create saas_knowledge_base table with JSONB content
CREATE TABLE IF NOT EXISTS saas_knowledge_base (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES saas_clients(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content JSONB NOT NULL,
    tags TEXT[],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_saas_kb_client_type ON saas_knowledge_base(client_id, type);
CREATE INDEX IF NOT EXISTS idx_saas_kb_tags ON saas_knowledge_base USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_saas_kb_content ON saas_knowledge_base USING GIN(content);

-- Add comment to explain JSONB usage
COMMENT ON COLUMN saas_knowledge_base.content IS 'Flexible JSONB content for FAQ, products, services, policies, etc.';
