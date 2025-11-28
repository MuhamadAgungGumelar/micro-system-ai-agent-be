-- Rollback migration: Drop universal clients and company_users tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_clients_updated_at ON clients;
DROP TRIGGER IF EXISTS update_company_users_updated_at ON company_users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Restore foreign keys to saas_clients
ALTER TABLE saas_workflows
  DROP CONSTRAINT IF EXISTS saas_workflows_client_id_fkey,
  ADD CONSTRAINT saas_workflows_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES saas_clients(id) ON DELETE CASCADE;

ALTER TABLE saas_knowledge_base
  DROP CONSTRAINT IF EXISTS saas_knowledge_base_client_id_fkey,
  ADD CONSTRAINT saas_knowledge_base_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES saas_clients(id) ON DELETE CASCADE;

ALTER TABLE saas_conversations
  DROP CONSTRAINT IF EXISTS saas_conversations_client_id_fkey,
  ADD CONSTRAINT saas_conversations_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES saas_clients(id) ON DELETE CASCADE;

ALTER TABLE saas_transactions
  DROP CONSTRAINT IF EXISTS saas_transactions_client_id_fkey,
  ADD CONSTRAINT saas_transactions_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES saas_clients(id) ON DELETE CASCADE;

-- Drop tables
DROP TABLE IF EXISTS company_users CASCADE;
DROP TABLE IF EXISTS clients CASCADE;
