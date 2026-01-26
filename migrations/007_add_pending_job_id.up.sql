-- Add pending_job_id to registration_drafts table
-- This allows saving a job ID during registration to redirect to booking after completion

ALTER TABLE registration_drafts 
ADD COLUMN pending_job_id BIGINT;

-- Add comment for clarity
COMMENT ON COLUMN registration_drafts.pending_job_id IS 'Job ID to redirect to after registration completes';
