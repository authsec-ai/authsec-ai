-- Migration: Update tenant domains to include 'app.' prefix
-- Purpose: Standardize all tenant domains to use app.authsec.dev format
-- Date: 2025-11-14
-- Status: COMPLETED - 11 tenants updated, 11 users updated

-- Update tenants table: Change X.authsec.dev to X.app.authsec.dev
-- Uses regex to only match numeric tenant IDs without existing 'app.' prefix
UPDATE tenants 
SET tenant_domain = REGEXP_REPLACE(tenant_domain, '([0-9]+)\.authsec\.dev', '\1.app.authsec.dev')
WHERE tenant_domain ~ '^[0-9]+\.authsec\.dev$';

-- Update users table: Change X.authsec.dev to X.app.authsec.dev
UPDATE users 
SET tenant_domain = REGEXP_REPLACE(tenant_domain, '([0-9]+)\.authsec\.dev', '\1.app.authsec.dev')
WHERE tenant_domain ~ '^[0-9]+\.authsec\.dev$';

-- Update pending_registrations table: Change X.authsec.dev to X.app.authsec.dev
UPDATE pending_registrations 
SET tenant_domain = REGEXP_REPLACE(tenant_domain, '([0-9]+)\.authsec\.dev', '\1.app.authsec.dev')
WHERE tenant_domain ~ '^[0-9]+\.authsec\.dev$';

-- Verify the updates
SELECT 'tenants' as table_name, COUNT(*) as count FROM tenants WHERE tenant_domain LIKE '%.app.authsec.dev'
UNION ALL
SELECT 'users' as table_name, COUNT(*) as count FROM users WHERE tenant_domain LIKE '%.app.authsec.dev'
UNION ALL
SELECT 'pending_registrations' as table_name, COUNT(*) as count FROM pending_registrations WHERE tenant_domain LIKE '%.app.authsec.dev';
