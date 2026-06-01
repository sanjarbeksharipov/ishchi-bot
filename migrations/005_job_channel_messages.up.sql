-- Migration: Add job_channel_messages table for scalable multi-channel support
-- Instead of adding a new column per channel (channel2_message_id, channel3_message_id...),
-- we use a junction table: adding a new channel only requires a config change.

CREATE TABLE IF NOT EXISTS job_channel_messages (
    id         BIGSERIAL PRIMARY KEY,
    job_id     BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    channel_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, channel_id)
);

CREATE INDEX idx_job_channel_messages_job_id ON job_channel_messages(job_id);
