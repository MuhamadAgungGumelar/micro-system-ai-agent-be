-- Create saas_workflows table
CREATE TABLE IF NOT EXISTS saas_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES saas_clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(50) NOT NULL, -- 'event', 'scheduled', 'manual'
    trigger_config JSONB NOT NULL DEFAULT '{}', -- Trigger-specific configuration
    conditions JSONB DEFAULT '[]', -- Array of conditions
    actions JSONB NOT NULL DEFAULT '[]', -- Array of actions to execute
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create saas_workflow_executions table
CREATE TABLE IF NOT EXISTS saas_workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES saas_workflows(id) ON DELETE CASCADE,
    trigger_data JSONB, -- Data that triggered the workflow
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    actions_completed INT DEFAULT 0,
    actions_failed INT DEFAULT 0,
    execution_log JSONB DEFAULT '[]', -- Detailed execution logs
    error_message TEXT,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INT -- Execution duration in milliseconds
);

-- Create indexes for better query performance
CREATE INDEX idx_saas_workflows_client_id ON saas_workflows(client_id);
CREATE INDEX idx_saas_workflows_trigger_type ON saas_workflows(trigger_type);
CREATE INDEX idx_saas_workflows_is_active ON saas_workflows(is_active);
CREATE INDEX idx_saas_workflows_created_at ON saas_workflows(created_at DESC);

CREATE INDEX idx_saas_workflow_executions_workflow_id ON saas_workflow_executions(workflow_id);
CREATE INDEX idx_saas_workflow_executions_status ON saas_workflow_executions(status);
CREATE INDEX idx_saas_workflow_executions_started_at ON saas_workflow_executions(started_at DESC);

-- Add updated_at trigger for saas_workflows
CREATE OR REPLACE FUNCTION update_saas_workflows_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_saas_workflows_updated_at
BEFORE UPDATE ON saas_workflows
FOR EACH ROW
EXECUTE FUNCTION update_saas_workflows_updated_at();

-- Add comments for documentation
COMMENT ON TABLE saas_workflows IS 'Stores workflow automation rules for clients';
COMMENT ON TABLE saas_workflow_executions IS 'Logs each workflow execution with results';
COMMENT ON COLUMN saas_workflows.trigger_type IS 'Type of trigger: event, scheduled, or manual';
COMMENT ON COLUMN saas_workflows.trigger_config IS 'Configuration for trigger (e.g., cron expression, event name)';
COMMENT ON COLUMN saas_workflows.conditions IS 'Array of conditions that must be met for workflow to execute';
COMMENT ON COLUMN saas_workflows.actions IS 'Array of actions to execute when workflow is triggered';
COMMENT ON COLUMN saas_workflow_executions.trigger_data IS 'Data context when workflow was triggered';
COMMENT ON COLUMN saas_workflow_executions.execution_log IS 'Detailed logs of each action execution';
