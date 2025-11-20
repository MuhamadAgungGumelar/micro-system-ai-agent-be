-- Revert: Rename back to session_id
ALTER TABLE saas_clients RENAME COLUMN whatsapp_session_id TO session_id;
