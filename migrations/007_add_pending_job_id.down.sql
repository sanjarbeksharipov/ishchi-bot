-- Remove pending_job_id from registration_drafts table

ALTER TABLE registration_drafts 
DROP COLUMN IF EXISTS pending_job_id;
