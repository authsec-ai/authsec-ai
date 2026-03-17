-- Add role_name to role_bindings for denormalized display and counts
ALTER TABLE role_bindings
ADD COLUMN IF NOT EXISTS role_name TEXT;
