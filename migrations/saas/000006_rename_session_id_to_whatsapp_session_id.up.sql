-- Rename session_id column to whatsapp_session_id for clarity
ALTER TABLE saas_clients RENAME COLUMN session_id TO whatsapp_session_id;
