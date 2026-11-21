-- 000003_add_user_roles.up.sql

-- Make email optional since login is purely username-based
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

-- Add role column
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';


