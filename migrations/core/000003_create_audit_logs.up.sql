-- Create audit_logs table for tracking all system changes
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    client_id UUID NOT NULL,

    -- Action details
    action VARCHAR(50) NOT NULL,  -- create, update, delete, view, login, etc.
    entity VARCHAR(100) NOT NULL, -- product, user, order, etc.
    entity_id VARCHAR(255),

    -- Change tracking (JSONB for flexibility)
    old_value JSONB,
    new_value JSONB,

    -- Request metadata
    ip_address VARCHAR(45),
    user_agent TEXT,
    method VARCHAR(10),     -- HTTP method (GET, POST, PUT, DELETE)
    endpoint VARCHAR(255),  -- API endpoint
    duration BIGINT,        -- Request duration in milliseconds

    -- Additional context
    description TEXT,
    metadata JSONB,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign key constraints
    CONSTRAINT fk_audit_logs_client FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE,
    CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES company_users(id) ON DELETE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX idx_audit_logs_client_id ON audit_logs(client_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity);
CREATE INDEX idx_audit_logs_entity_id ON audit_logs(entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Composite index for common queries
CREATE INDEX idx_audit_logs_client_entity ON audit_logs(client_id, entity, entity_id);
CREATE INDEX idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC);

-- Comment on table
COMMENT ON TABLE audit_logs IS 'Tracks all system changes and user activities for compliance and debugging';
