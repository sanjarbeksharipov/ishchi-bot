-- Drop location index
DROP INDEX IF EXISTS idx_jobs_location;

-- Remove location field from jobs table
ALTER TABLE jobs DROP COLUMN IF EXISTS location;
