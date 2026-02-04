-- ============================================
-- Rollback Initial Database Schema
-- ============================================

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS blocked_users CASCADE;
DROP TABLE IF EXISTS user_violations CASCADE;
DROP TABLE IF EXISTS registered_users CASCADE;
DROP TABLE IF EXISTS registration_drafts CASCADE;
DROP TABLE IF EXISTS job_bookings CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
DROP SEQUENCE IF EXISTS job_order_number_seq CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop the function
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
