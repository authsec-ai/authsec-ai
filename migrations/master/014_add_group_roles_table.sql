-- Migration: Add group_roles junction table
-- This migration adds the missing group_roles table for group-role assignments

-- Create group_roles table (many-to-many: groups <-> roles)
CREATE TABLE IF NOT EXISTS group_roles (
    group_id UUID NOT NULL,
    role_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (group_id, role_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_group_roles_group_id ON group_roles(group_id);
CREATE INDEX IF NOT EXISTS idx_group_roles_role_id ON group_roles(role_id);