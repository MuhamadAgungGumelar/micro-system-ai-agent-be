-- Add session_id column to saas_clients table
-- This allows tracking which WhatsApp session belongs to which client

ALTER TABLE saas_clients
ADD COLUMN session_id TEXT;

-- Add index for faster session lookups
CREATE INDEX IF NOT EXISTS idx_saas_clients_session_id ON saas_clients(session_id);

COMMENT ON COLUMN saas_clients.session_id IS 'WhatsApp session identifier for WAHA/multi-session providers';
