-- ============================================
-- Initial Database Schema for Ishchi Bot
-- ============================================

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================
-- Users Table
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    state VARCHAR(50) NOT NULL DEFAULT 'idle',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_state ON users(state);

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Jobs Table
-- ============================================

-- Create sequence for order numbers
CREATE SEQUENCE IF NOT EXISTS job_order_number_seq START 1000;

CREATE TABLE IF NOT EXISTS jobs (
    id BIGSERIAL PRIMARY KEY,
    order_number INT NOT NULL UNIQUE,
    
    -- Job details
    salary VARCHAR(255) NOT NULL,
    food VARCHAR(255),
    work_time VARCHAR(255) NOT NULL,
    address VARCHAR(500) NOT NULL,
    service_fee INT NOT NULL,
    buses VARCHAR(255),
    additional_info TEXT,
    work_date VARCHAR(100) NOT NULL,
    employer_phone VARCHAR(20),
    
    -- Capacity management (CRITICAL for race conditions)
    required_workers INT NOT NULL CHECK (required_workers > 0),
    reserved_slots INT NOT NULL DEFAULT 0 CHECK (reserved_slots >= 0),
    confirmed_slots INT NOT NULL DEFAULT 0 CHECK (confirmed_slots >= 0),
    
    -- Status: DRAFT, ACTIVE, FULL, COMPLETED, CANCELLED
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    
    -- Telegram channel integration
    channel_message_id BIGINT,
    admin_message_id BIGINT,
    
    -- Admin who created the job
    created_by_admin_id BIGINT NOT NULL REFERENCES users(id),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Critical constraint: prevent overbooking
    CONSTRAINT check_slots_not_exceed_required CHECK (
        reserved_slots + confirmed_slots <= required_workers
    )
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_order_number ON jobs(order_number);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_work_date ON jobs(work_date);
CREATE INDEX idx_jobs_employer_phone ON jobs(employer_phone);
CREATE INDEX idx_jobs_admin_message_id ON jobs(admin_message_id) WHERE admin_message_id IS NOT NULL;

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Job Bookings Table
-- ============================================
CREATE TABLE IF NOT EXISTS job_bookings (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- State: SLOT_RESERVED, PAYMENT_SUBMITTED, CONFIRMED, REJECTED, EXPIRED, CANCELLED_BY_USER
    status VARCHAR(50) NOT NULL DEFAULT 'SLOT_RESERVED',
    
    -- Payment tracking
    payment_receipt_file_id VARCHAR(255),
    payment_receipt_message_id BIGINT,
    payment_instruction_message_id BIGINT,
    
    -- Timing (CRITICAL for expiry management)
    reserved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    payment_submitted_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    
    -- Admin review
    reviewed_by_admin_id BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMP,
    rejection_reason TEXT,
    
    -- Idempotency (CRITICAL for Telegram retries)
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Only one active booking per user per job
    CONSTRAINT unique_user_job_booking UNIQUE(job_id, user_id)
);

CREATE INDEX idx_job_bookings_job_id ON job_bookings(job_id);
CREATE INDEX idx_job_bookings_user_id ON job_bookings(user_id);
CREATE INDEX idx_job_bookings_status ON job_bookings(status);
CREATE INDEX idx_job_bookings_idempotency ON job_bookings(idempotency_key);
CREATE INDEX idx_job_bookings_expiry ON job_bookings(expires_at, status) 
    WHERE status = 'SLOT_RESERVED';
CREATE INDEX idx_job_bookings_pending_approval ON job_bookings(payment_submitted_at, status)
    WHERE status = 'PAYMENT_SUBMITTED';

CREATE TRIGGER update_job_bookings_updated_at BEFORE UPDATE ON job_bookings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- Registration Tables
-- ============================================

-- Registration drafts (incomplete registrations)
CREATE TABLE IF NOT EXISTS registration_drafts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL DEFAULT 'reg_public_offer',
    previous_state VARCHAR(50) NOT NULL DEFAULT 'reg_idle',
    pending_job_id BIGINT REFERENCES jobs(id) ON DELETE SET NULL,
    full_name VARCHAR(255),
    phone VARCHAR(50),
    age INT,
    weight INT,
    height INT,
    passport_photo_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_registration_drafts_user_id ON registration_drafts(user_id);
CREATE INDEX idx_registration_drafts_state ON registration_drafts(state);

CREATE TRIGGER update_registration_drafts_updated_at BEFORE UPDATE ON registration_drafts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Registered users (complete registrations)
CREATE TABLE IF NOT EXISTS registered_users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    age INT NOT NULL,
    weight INT NOT NULL,
    height INT NOT NULL,
    passport_photo_id VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_registered_users_user_id ON registered_users(user_id);
CREATE INDEX idx_registered_users_phone ON registered_users(phone);
CREATE INDEX idx_registered_users_is_active ON registered_users(is_active);

CREATE TRIGGER update_registered_users_updated_at BEFORE UPDATE ON registered_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- User Violations and Blocking Tables
-- ============================================

-- User violations tracking
CREATE TABLE IF NOT EXISTS user_violations (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    violation_type VARCHAR(50) NOT NULL,
    booking_id BIGINT REFERENCES job_bookings(id) ON DELETE SET NULL,
    admin_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_violations_user_id ON user_violations(user_id);
CREATE INDEX idx_user_violations_created_at ON user_violations(created_at);

-- Blocked users
CREATE TABLE IF NOT EXISTS blocked_users (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    blocked_until TIMESTAMP, -- NULL means permanent block
    total_violations INT NOT NULL DEFAULT 0,
    blocked_by_admin_id BIGINT,
    reason TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blocked_users_blocked_until ON blocked_users(blocked_until);

CREATE TRIGGER update_blocked_users_updated_at BEFORE UPDATE ON blocked_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
