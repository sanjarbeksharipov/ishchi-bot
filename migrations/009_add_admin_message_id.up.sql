-- Migration: Add admin_message_id to jobs table
-- This field stores the message ID of the admin job detail message
-- Ensures only one admin message exists per job

ALTER TABLE jobs ADD COLUMN admin_message_id BIGINT DEFAULT NULL;

-- Add index for faster lookups
CREATE INDEX idx_jobs_admin_message_id ON jobs(admin_message_id) WHERE admin_message_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN jobs.admin_message_id IS 'Telegram message ID for the admin job detail message. Only one admin message should exist per job.';
