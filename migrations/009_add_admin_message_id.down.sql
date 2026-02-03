-- Rollback: Remove admin_message_id from jobs table

DROP INDEX IF EXISTS idx_jobs_admin_message_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS admin_message_id;
