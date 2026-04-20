-- Migration: 025_add_role_assignment_requests.sql
-- Description: Add table for end-user role assignment requests

CREATE TABLE IF NOT EXISTS role_assignment_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_role_assignment_requests_user_id ON role_assignment_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_role_assignment_requests_role_id ON role_assignment_requests(role_id);
CREATE INDEX IF NOT EXISTS idx_role_assignment_requests_tenant_id ON role_assignment_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_role_assignment_requests_status ON role_assignment_requests(status);

-- Add comment for documentation
COMMENT ON TABLE role_assignment_requests IS 'End-user role assignment requests requiring admin approval';
COMMENT ON COLUMN role_assignment_requests.status IS 'Request status: pending, approved, rejected';