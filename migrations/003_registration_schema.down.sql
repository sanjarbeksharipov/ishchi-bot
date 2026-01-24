-- Rollback registration tables

DROP TRIGGER IF EXISTS update_registered_users_updated_at ON registered_users;
DROP TRIGGER IF EXISTS update_registration_drafts_updated_at ON registration_drafts;

DROP INDEX IF EXISTS idx_registered_users_is_active;
DROP INDEX IF EXISTS idx_registered_users_phone;
DROP INDEX IF EXISTS idx_registered_users_user_id;
DROP INDEX IF EXISTS idx_registration_drafts_state;
DROP INDEX IF EXISTS idx_registration_drafts_user_id;

DROP TABLE IF EXISTS registered_users;
DROP TABLE IF EXISTS registration_drafts;
