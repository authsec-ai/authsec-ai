-- Migration 016: Convert hydra_client_id unique constraint to partial unique index.
-- Allows AI agent clients to have empty hydra_client_id without violating uniqueness.

ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_hydra_client_id_key;
DROP INDEX IF EXISTS clients_hydra_client_id_key;
CREATE UNIQUE INDEX clients_hydra_client_id_key ON clients (hydra_client_id) WHERE hydra_client_id != '';
