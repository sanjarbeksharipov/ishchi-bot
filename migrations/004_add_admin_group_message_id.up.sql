-- Add admin_group_message_id to job_bookings to track payment receipt messages in admin group
ALTER TABLE job_bookings 
ADD COLUMN admin_group_message_id BIGINT;

CREATE INDEX idx_job_bookings_admin_group_msg ON job_bookings(admin_group_message_id) 
    WHERE admin_group_message_id IS NOT NULL;
