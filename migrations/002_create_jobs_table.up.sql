-- Migration: Create jobs table
-- Table for storing job postings with slot management

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
    
    -- Capacity management (CRITICAL for race conditions)
    required_workers INT NOT NULL CHECK (required_workers > 0),
    reserved_slots INT NOT NULL DEFAULT 0 CHECK (reserved_slots >= 0),
    confirmed_slots INT NOT NULL DEFAULT 0 CHECK (confirmed_slots >= 0),
    
    -- Status: DRAFT, ACTIVE, FULL, COMPLETED, CANCELLED
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    
    -- Telegram channel integration
    channel_message_id BIGINT,
    
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

-- Indexes for performance
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_order_number ON jobs(order_number);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_work_date ON jobs(work_date);

-- Trigger to automatically update updated_at
CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create sequence for order numbers
CREATE SEQUENCE IF NOT EXISTS job_order_number_seq START 1000;
