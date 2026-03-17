-- Add full_permission_string to permissions for denormalized display and API returns
ALTER TABLE permissions
ADD COLUMN IF NOT EXISTS full_permission_string TEXT;

-- Backfill existing rows with resource:action
UPDATE permissions
SET full_permission_string = CONCAT(resource, ':', action)
WHERE full_permission_string IS NULL;
