-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_jobs_updated_at ON jobs;
DROP FUNCTION IF EXISTS update_jobs_updated_at();

-- Drop jobs table
DROP TABLE IF EXISTS jobs CASCADE;
