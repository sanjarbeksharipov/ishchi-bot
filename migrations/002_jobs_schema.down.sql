DROP TRIGGER IF EXISTS trigger_update_jobs_updated_at ON jobs;
DROP FUNCTION IF EXISTS update_jobs_updated_at();
DROP INDEX IF EXISTS idx_jobs_order_number;
DROP INDEX IF EXISTS idx_jobs_status;
DROP TABLE IF EXISTS jobs;
