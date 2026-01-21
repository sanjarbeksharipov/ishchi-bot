-- Rollback migration - removes all schema changes

-- Drop trigger first
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_users_state;
DROP INDEX IF EXISTS idx_users_username;

-- Drop table
DROP TABLE IF EXISTS users;
