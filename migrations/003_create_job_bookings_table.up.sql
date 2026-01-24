-- Migration: Create job_bookings table
-- Table for tracking user job reservations and payments

CREATE TABLE IF NOT EXISTS job_bookings (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- State: SLOT_RESERVED, PAYMENT_SUBMITTED, CONFIRMED, REJECTED, EXPIRED, CANCELLED_BY_USER
    status VARCHAR(50) NOT NULL DEFAULT 'SLOT_RESERVED',
    
    -- Payment tracking
    payment_receipt_file_id VARCHAR(255),
    payment_receipt_message_id BIGINT,
    
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

-- Indexes for performance and queries
CREATE INDEX idx_job_bookings_job_id ON job_bookings(job_id);
CREATE INDEX idx_job_bookings_user_id ON job_bookings(user_id);
CREATE INDEX idx_job_bookings_status ON job_bookings(status);
CREATE INDEX idx_job_bookings_idempotency ON job_bookings(idempotency_key);

-- Critical index for expiry worker
CREATE INDEX idx_job_bookings_expiry ON job_bookings(expires_at, status) 
    WHERE status = 'SLOT_RESERVED';

-- Index for admin panel (pending approvals)
CREATE INDEX idx_job_bookings_pending_approval ON job_bookings(payment_submitted_at, status)
    WHERE status = 'PAYMENT_SUBMITTED';

-- Trigger to automatically update updated_at
CREATE TRIGGER update_job_bookings_updated_at BEFORE UPDATE ON job_bookings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
