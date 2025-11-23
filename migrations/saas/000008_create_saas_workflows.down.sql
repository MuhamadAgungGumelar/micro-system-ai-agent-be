-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_saas_workflows_updated_at ON saas_workflows;
DROP FUNCTION IF EXISTS update_saas_workflows_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_saas_workflow_executions_started_at;
DROP INDEX IF EXISTS idx_saas_workflow_executions_status;
DROP INDEX IF EXISTS idx_saas_workflow_executions_workflow_id;

DROP INDEX IF EXISTS idx_saas_workflows_created_at;
DROP INDEX IF EXISTS idx_saas_workflows_is_active;
DROP INDEX IF EXISTS idx_saas_workflows_trigger_type;
DROP INDEX IF EXISTS idx_saas_workflows_client_id;

-- Drop tables
DROP TABLE IF EXISTS saas_workflow_executions;
DROP TABLE IF EXISTS saas_workflows;
