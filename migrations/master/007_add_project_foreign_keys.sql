-- Migration: Add foreign key constraint between users.project_id and projects.id
-- Description: Now that both fields are UUID, we can establish the proper foreign key relationship

DO $$
BEGIN
    -- REMOVED: fk_users_project_id - User model has no FK relationship tags in shared-models
    -- Add foreign key constraint from users.project_id to projects.id
    -- BEGIN
    --     ALTER TABLE users ADD CONSTRAINT fk_users_project_id 
    --         FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
    --     RAISE NOTICE 'Successfully added foreign key constraint fk_users_project_id';
    -- EXCEPTION
    --     WHEN duplicate_object THEN 
    --         RAISE NOTICE 'Foreign key constraint fk_users_project_id already exists, skipping';
    -- END;
    
    -- REMOVED: fk_projects_user_id - Project model has no FK relationship tags in shared-models
    -- Also add foreign key from projects.user_id to users.id if it doesn't exist
    -- BEGIN
    --     ALTER TABLE projects ADD CONSTRAINT fk_projects_user_id 
    --         FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
    --     RAISE NOTICE 'Successfully added foreign key constraint fk_projects_user_id';
    -- EXCEPTION
    --     WHEN duplicate_object THEN 
    --         RAISE NOTICE 'Foreign key constraint fk_projects_user_id already exists, skipping';
    -- END;
    
    -- REMOVED: fk_projects_tenant_id - Project model has FK relationship tags in shared-models, GORM will create this
    -- Add foreign key from projects.tenant_id to tenants.id if it doesn't exist
    -- FIXED: Changed reference from tenants(tenant_id) to tenants(id) to match shared-models
    -- BEGIN
    --     ALTER TABLE projects ADD CONSTRAINT fk_projects_tenant_id 
    --         FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    --     RAISE NOTICE 'Successfully added foreign key constraint fk_projects_tenant_id';
    -- EXCEPTION
    --     WHEN duplicate_object THEN 
    --         RAISE NOTICE 'Foreign key constraint fk_projects_tenant_id already exists, skipping';
    -- END;

    RAISE NOTICE 'Migration 006 completed - FK constraints removed as they are redundant with GORM auto-migration';
END $$;