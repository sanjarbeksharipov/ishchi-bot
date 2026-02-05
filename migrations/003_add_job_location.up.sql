-- Add location field to jobs table
ALTER TABLE jobs ADD COLUMN location VARCHAR(500);

-- Create index for location searches
CREATE INDEX idx_jobs_location ON jobs(location) WHERE location IS NOT NULL;
