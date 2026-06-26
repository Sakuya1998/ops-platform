ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_org_id_username_key;
DROP INDEX IF EXISTS idx_org_username;
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_org_username_active ON users(org_id, username) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_users_deleted_at ON users(deleted_at);
