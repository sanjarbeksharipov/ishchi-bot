-- ============================================
-- Admin Job Messages Table
-- Stores one message per admin per job for independent message control
-- ============================================
CREATE TABLE IF NOT EXISTS admin_job_messages (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    admin_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- One message per admin per job
    CONSTRAINT unique_admin_job_message UNIQUE(job_id, admin_id)
);

CREATE INDEX idx_admin_job_messages_job_id ON admin_job_messages(job_id);
CREATE INDEX idx_admin_job_messages_admin_id ON admin_job_messages(admin_id);
CREATE INDEX idx_admin_job_messages_message_id ON admin_job_messages(message_id);

CREATE TRIGGER update_admin_job_messages_updated_at BEFORE UPDATE ON admin_job_messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migrate existing admin_message_id data to new table
INSERT INTO admin_job_messages (job_id, admin_id, message_id, created_at, updated_at)
SELECT id, created_by_admin_id, admin_message_id, created_at, updated_at
FROM jobs
WHERE admin_message_id IS NOT NULL;
