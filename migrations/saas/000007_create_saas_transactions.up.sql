-- Create transactions table for SAAS/UMKM module
CREATE TABLE IF NOT EXISTS saas_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES saas_clients(id) ON DELETE CASCADE,

    -- Transaction details
    total_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    transaction_date TIMESTAMP NOT NULL DEFAULT NOW(),
    store_name VARCHAR(255),

    -- Items as JSONB for flexibility
    items JSONB,

    -- Source tracking
    created_from VARCHAR(20) NOT NULL DEFAULT 'manual' CHECK (created_from IN ('ocr', 'manual')),
    source_type VARCHAR(20) NOT NULL DEFAULT 'manual' CHECK (source_type IN ('receipt', 'invoice', 'manual')),

    -- OCR metadata (optional)
    ocr_confidence FLOAT,
    ocr_raw_text TEXT,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_saas_transactions_client ON saas_transactions(client_id);
CREATE INDEX idx_saas_transactions_date ON saas_transactions(transaction_date DESC);
CREATE INDEX idx_saas_transactions_created_from ON saas_transactions(created_from);
CREATE INDEX idx_saas_transactions_source_type ON saas_transactions(source_type);

-- Add updated_at trigger
CREATE OR REPLACE FUNCTION update_saas_transactions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_saas_transactions_updated_at
    BEFORE UPDATE ON saas_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_saas_transactions_updated_at();
