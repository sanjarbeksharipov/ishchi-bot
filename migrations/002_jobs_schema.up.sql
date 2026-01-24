-- Jobs table for storing job postings
CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY,
    order_number SERIAL UNIQUE,
    ish_haqqi VARCHAR(255) NOT NULL,
    ovqat VARCHAR(255),
    vaqt VARCHAR(255),
    manzil VARCHAR(255),
    xizmat_haqqi INTEGER DEFAULT 0,
    avtobuslar VARCHAR(255),
    qoshimcha TEXT,
    ish_kuni VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    kerakli_ishchilar INTEGER DEFAULT 1,
    band_ishchilar INTEGER DEFAULT 0,
    channel_message_id BIGINT,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for status filtering
CREATE INDEX idx_jobs_status ON jobs(status);

-- Index for order number lookups
CREATE INDEX idx_jobs_order_number ON jobs(order_number);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_jobs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_jobs_updated_at();
