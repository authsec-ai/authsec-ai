-- Migration: Drop client_scopes table
-- Description: Removes the client_scopes table as it is no longer used.

DROP TABLE IF EXISTS client_scopes CASCADE;
