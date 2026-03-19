-- Remove admin_group_message_id column from job_bookings
DROP INDEX IF EXISTS idx_job_bookings_admin_group_msg;
ALTER TABLE job_bookings DROP COLUMN admin_group_message_id;
