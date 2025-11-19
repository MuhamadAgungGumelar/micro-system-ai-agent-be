-- Create saas_credits table
CREATE TABLE IF NOT EXISTS saas_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES saas_clients(id) ON DELETE CASCADE,
    credits_used INT DEFAULT 0,
    period_start DATE DEFAULT CURRENT_DATE,
    period_end DATE DEFAULT CURRENT_DATE + INTERVAL '30 days'
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_saas_credits_client ON saas_credits(client_id);
CREATE INDEX IF NOT EXISTS idx_saas_credits_period ON saas_credits(period_start, period_end);
