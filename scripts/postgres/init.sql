-- postgres/init.sql
-- Runs once at first container startup (as the postgres superuser).
-- Creates the Hydra database on the same postgres instance.
-- The POSTGRES_USER (authsec by default) is created as a superuser by the
-- official postgres Docker image, so it can create tenant databases at runtime
-- without any extra grants.

CREATE DATABASE hydra;
